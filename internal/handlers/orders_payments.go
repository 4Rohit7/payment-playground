package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"payment-playground/internal/razorpay"
)

// PublicConfig exposes only what a browser may know: the key id, never the secret.
func (h *Handler) PublicConfig(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"key_id":        h.rp.KeyID(),
		"merchant_name": h.cfg.MerchantName,
		"test_mode":     len(h.rp.KeyID()) > 9 && h.rp.KeyID()[:9] == "rzp_test_",
	})
}

// Methods lists the payment methods enabled on your Razorpay account.
func (h *Handler) Methods(c *fiber.Ctx) error {
	raw, err := h.rp.FetchMethods(c.UserContext())
	if err != nil {
		return err
	}
	return c.Type("json").Send(raw)
}

// ---------- Customers ----------

func (h *Handler) CreateCustomer(c *fiber.Ctx) error {
	var req razorpay.CreateCustomerRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	if req.Name == "" {
		return badRequest("name is required")
	}
	if req.FailExisting == "" {
		req.FailExisting = "0" // return the existing customer for the same email/contact
	}
	cust, err := h.rp.CreateCustomer(c.UserContext(), req)
	if err != nil {
		return err
	}
	logSave("customer", cust.ID, h.store.UpsertCustomer(c.UserContext(), cust))
	return c.Status(fiber.StatusCreated).JSON(cust.Raw)
}

func (h *Handler) GetCustomer(c *fiber.Ctx) error {
	cust, err := h.rp.FetchCustomer(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("customer", cust.ID, h.store.UpsertCustomer(c.UserContext(), cust))
	return c.JSON(cust.Raw)
}

// ---------- Orders ----------

func (h *Handler) CreateOrder(c *fiber.Ctx) error {
	var req razorpay.CreateOrderRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	if req.Amount < minAmount {
		return badRequest("amount is in paise and must be at least 100 (₹1)")
	}
	if req.Currency == "" {
		req.Currency = "INR"
	}
	if req.Receipt == "" {
		req.Receipt = fmt.Sprintf("rcpt_%d", time.Now().UnixNano()) // max 40 chars
	}
	order, err := h.rp.CreateOrder(c.UserContext(), req)
	if err != nil {
		return err
	}
	logSave("order", order.ID, h.store.UpsertOrder(c.UserContext(), order))
	return c.Status(fiber.StatusCreated).JSON(order.Raw)
}

func (h *Handler) GetOrder(c *fiber.Ctx) error {
	order, err := h.rp.FetchOrder(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("order", order.ID, h.store.UpsertOrder(c.UserContext(), order))
	return c.JSON(order.Raw)
}

// GetOrderPayments shows every attempt on an order (failed ones included) and syncs them to the DB.
func (h *Handler) GetOrderPayments(c *fiber.Ctx) error {
	payments, raw, err := h.rp.FetchOrderPayments(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	for _, p := range payments {
		logSave("payment", p.ID, h.store.UpsertPayment(c.UserContext(), p))
	}
	return c.Type("json").Send(raw)
}

// ---------- Payments ----------

type verifyPaymentInput struct {
	OrderID   string `json:"razorpay_order_id"`
	PaymentID string `json:"razorpay_payment_id"`
	Signature string `json:"razorpay_signature"`
}

// VerifyPayment is called by the frontend with what Checkout's handler returned.
// Never mark an order paid without this check (or the webhook).
func (h *Handler) VerifyPayment(c *fiber.Ctx) error {
	var in verifyPaymentInput
	if err := parseBody(c, &in); err != nil {
		return err
	}
	if in.OrderID == "" || in.PaymentID == "" || in.Signature == "" {
		return badRequest("razorpay_order_id, razorpay_payment_id and razorpay_signature are required")
	}
	if !razorpay.VerifyPaymentSignature(in.OrderID, in.PaymentID, in.Signature, h.rp.KeySecret()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"verified": false, "error": "signature mismatch"})
	}

	ctx := c.UserContext()
	// Signature proves the pair is genuine; fetch the payment for its real status and method.
	payment, err := h.rp.FetchPayment(ctx, in.PaymentID)
	if err != nil {
		return err
	}
	if payment.OrderID != in.OrderID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"verified": false, "error": "payment does not belong to this order"})
	}
	logSave("payment", payment.ID, h.store.UpsertPayment(ctx, payment))
	if order, err := h.rp.FetchOrder(ctx, in.OrderID); err == nil {
		logSave("order", order.ID, h.store.UpsertOrder(ctx, order))
	}

	return c.JSON(fiber.Map{
		"verified":   true,
		"order_id":   in.OrderID,
		"payment_id": payment.ID,
		"status":     payment.Status,
		"method":     payment.Method,
		"amount":     payment.Amount,
		"captured":   payment.Captured,
		"payment":    payment.Raw,
	})
}

func (h *Handler) GetPayment(c *fiber.Ctx) error {
	p, err := h.rp.FetchPayment(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("payment", p.ID, h.store.UpsertPayment(c.UserContext(), p))
	return c.JSON(p.Raw)
}

// CapturePayment is only needed if auto-capture is off in your dashboard
// (payments then stay "authorized" until captured, and auto-refund if not captured in time).
func (h *Handler) CapturePayment(c *fiber.Ctx) error {
	var req razorpay.CapturePaymentRequest
	if err := parseOptionalBody(c, &req); err != nil {
		return err
	}
	ctx := c.UserContext()
	id := c.Params("id")
	if req.Amount == 0 {
		p, err := h.rp.FetchPayment(ctx, id)
		if err != nil {
			return err
		}
		req.Amount, req.Currency = p.Amount, p.Currency
	}
	if req.Currency == "" {
		req.Currency = "INR"
	}
	p, err := h.rp.CapturePayment(ctx, id, req)
	if err != nil {
		return err
	}
	logSave("payment", p.ID, h.store.UpsertPayment(ctx, p))
	return c.JSON(p.Raw)
}
