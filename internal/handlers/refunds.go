package handlers

import (
	"github.com/gofiber/fiber/v2"

	"payment-playground/internal/razorpay"
)

// CreateRefund refunds a captured payment. Empty body or no amount = full refund.
func (h *Handler) CreateRefund(c *fiber.Ctx) error {
	var req razorpay.CreateRefundRequest
	if err := parseOptionalBody(c, &req); err != nil {
		return err
	}
	if req.Amount != 0 && req.Amount < minAmount {
		return badRequest("amount is in paise and must be at least 100 (₹1); omit it for a full refund")
	}
	if req.Speed != "" && req.Speed != "normal" && req.Speed != "optimum" {
		return badRequest(`speed must be "normal" or "optimum"`)
	}
	ctx := c.UserContext()
	paymentID := c.Params("id")

	refund, err := h.rp.CreateRefund(ctx, paymentID, req)
	if err != nil {
		return err
	}
	logSave("refund", refund.ID, h.store.UpsertRefund(ctx, refund))
	// Refresh the payment so amount_refunded / status are current in the DB.
	if p, err := h.rp.FetchPayment(ctx, paymentID); err == nil {
		logSave("payment", p.ID, h.store.UpsertPayment(ctx, p))
	}
	return c.Status(fiber.StatusCreated).JSON(refund.Raw)
}

func (h *Handler) GetRefund(c *fiber.Ctx) error {
	r, err := h.rp.FetchRefund(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("refund", r.ID, h.store.UpsertRefund(c.UserContext(), r))
	return c.JSON(r.Raw)
}

func (h *Handler) GetPaymentRefunds(c *fiber.Ctx) error {
	refunds, raw, err := h.rp.FetchPaymentRefunds(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	for _, r := range refunds {
		logSave("refund", r.ID, h.store.UpsertRefund(c.UserContext(), r))
	}
	return c.Type("json").Send(raw)
}
