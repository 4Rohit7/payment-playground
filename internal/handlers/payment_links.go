package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"payment-playground/internal/razorpay"
)

func (h *Handler) CreatePaymentLink(c *fiber.Ctx) error {
	var req razorpay.CreatePaymentLinkRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	if req.Amount < minAmount {
		return badRequest("amount is in paise and must be at least 100 (₹1)")
	}
	if req.Currency == "" {
		req.Currency = "INR"
	}
	if req.ReferenceID == "" {
		req.ReferenceID = fmt.Sprintf("link_%d", time.Now().UnixNano())
	}
	if req.CallbackURL == "" {
		req.CallbackURL = h.cfg.PublicBaseURL + "/api/payment-links/callback"
		req.CallbackMethod = "get"
	}
	link, err := h.rp.CreatePaymentLink(c.UserContext(), req)
	if err != nil {
		return err
	}
	logSave("payment link", link.ID, h.store.UpsertPaymentLink(c.UserContext(), link))
	return c.Status(fiber.StatusCreated).JSON(link.Raw)
}

func (h *Handler) GetPaymentLink(c *fiber.Ctx) error {
	link, err := h.rp.FetchPaymentLink(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("payment link", link.ID, h.store.UpsertPaymentLink(c.UserContext(), link))
	return c.JSON(link.Raw)
}

func (h *Handler) CancelPaymentLink(c *fiber.Ctx) error {
	link, err := h.rp.CancelPaymentLink(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("payment link", link.ID, h.store.UpsertPaymentLink(c.UserContext(), link))
	return c.JSON(link.Raw)
}

// PaymentLinkCallback is where Razorpay redirects the customer's browser after paying a link.
func (h *Handler) PaymentLinkCallback(c *fiber.Ctx) error {
	paymentID := c.Query("razorpay_payment_id")
	linkID := c.Query("razorpay_payment_link_id")
	refID := c.Query("razorpay_payment_link_reference_id")
	status := c.Query("razorpay_payment_link_status")
	sig := c.Query("razorpay_signature")

	if !razorpay.VerifyPaymentLinkSignature(linkID, refID, status, paymentID, sig, h.rp.KeySecret()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"verified": false, "error": "signature mismatch"})
	}

	ctx := c.UserContext()
	if link, err := h.rp.FetchPaymentLink(ctx, linkID); err == nil {
		logSave("payment link", link.ID, h.store.UpsertPaymentLink(ctx, link))
	}
	if paymentID != "" {
		if p, err := h.rp.FetchPayment(ctx, paymentID); err == nil {
			logSave("payment", p.ID, h.store.UpsertPayment(ctx, p))
		}
	}
	return c.JSON(fiber.Map{
		"verified":        true,
		"payment_link_id": linkID,
		"reference_id":    refID,
		"status":          status,
		"payment_id":      paymentID,
	})
}
