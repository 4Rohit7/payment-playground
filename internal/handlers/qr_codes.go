package handlers

import (
	"github.com/gofiber/fiber/v2"

	"payment-playground/internal/razorpay"
)

func (h *Handler) CreateQRCode(c *fiber.Ctx) error {
	var req razorpay.CreateQRCodeRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	if req.Type == "" {
		req.Type = "upi_qr"
	}
	if req.Usage == "" {
		req.Usage = "single_use"
	}
	if req.Usage != "single_use" && req.Usage != "multiple_use" {
		return badRequest(`usage must be "single_use" or "multiple_use"`)
	}
	if req.FixedAmount && req.PaymentAmount < minAmount {
		return badRequest("payment_amount (paise, min 100) is required when fixed_amount is true")
	}
	qr, err := h.rp.CreateQRCode(c.UserContext(), req)
	if err != nil {
		return err
	}
	logSave("qr code", qr.ID, h.store.UpsertQRCode(c.UserContext(), qr))
	return c.Status(fiber.StatusCreated).JSON(qr.Raw)
}

func (h *Handler) GetQRCode(c *fiber.Ctx) error {
	qr, err := h.rp.FetchQRCode(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("qr code", qr.ID, h.store.UpsertQRCode(c.UserContext(), qr))
	return c.JSON(qr.Raw)
}

func (h *Handler) GetQRCodePayments(c *fiber.Ctx) error {
	payments, raw, err := h.rp.FetchQRCodePayments(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	for _, p := range payments {
		logSave("payment", p.ID, h.store.UpsertPayment(c.UserContext(), p))
	}
	return c.Type("json").Send(raw)
}

func (h *Handler) CloseQRCode(c *fiber.Ctx) error {
	qr, err := h.rp.CloseQRCode(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("qr code", qr.ID, h.store.UpsertQRCode(c.UserContext(), qr))
	return c.JSON(qr.Raw)
}
