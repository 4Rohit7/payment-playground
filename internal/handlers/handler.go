package handlers

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"payment-playground/internal/config"
	"payment-playground/internal/razorpay"
	"payment-playground/internal/store"
)

type Handler struct {
	cfg   *config.Config
	rp    *razorpay.Client
	store *store.Store
}

func New(cfg *config.Config, rp *razorpay.Client, st *store.Store) *Handler {
	return &Handler{cfg: cfg, rp: rp, store: st}
}

func (h *Handler) Register(app *fiber.App) {
	app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

	api := app.Group("/api")
	api.Get("/config", h.PublicConfig)
	api.Get("/methods", h.Methods)

	api.Post("/customers", h.CreateCustomer)
	api.Get("/customers/:id", h.GetCustomer)

	api.Post("/orders", h.CreateOrder)
	api.Get("/orders/:id", h.GetOrder)
	api.Get("/orders/:id/payments", h.GetOrderPayments)

	api.Post("/payments/verify", h.VerifyPayment)
	api.Get("/payments/:id", h.GetPayment)
	api.Post("/payments/:id/capture", h.CapturePayment)
	api.Post("/payments/:id/refunds", h.CreateRefund)
	api.Get("/payments/:id/refunds", h.GetPaymentRefunds)
	api.Get("/refunds/:id", h.GetRefund)

	api.Post("/payment-links", h.CreatePaymentLink)
	api.Get("/payment-links/callback", h.PaymentLinkCallback) // must be before /:id
	api.Get("/payment-links/:id", h.GetPaymentLink)
	api.Post("/payment-links/:id/cancel", h.CancelPaymentLink)

	api.Post("/qr-codes", h.CreateQRCode)
	api.Get("/qr-codes/:id", h.GetQRCode)
	api.Get("/qr-codes/:id/payments", h.GetQRCodePayments)
	api.Post("/qr-codes/:id/close", h.CloseQRCode)

	api.Post("/plans", h.CreatePlan)
	api.Get("/plans/:id", h.GetPlan)
	api.Post("/subscriptions", h.CreateSubscription)
	api.Post("/subscriptions/verify", h.VerifySubscription) // must be before /:id
	api.Get("/subscriptions/:id", h.GetSubscription)
	api.Post("/subscriptions/:id/cancel", h.CancelSubscription)

	api.Get("/local/:resource", h.ListLocal)

	app.Post("/webhooks/razorpay", h.RazorpayWebhook)
}

// ErrorHandler turns Razorpay API errors and fiber errors into consistent JSON.
func ErrorHandler(c *fiber.Ctx, err error) error {
	var apiErr *razorpay.APIError
	if errors.As(err, &apiErr) {
		status := apiErr.StatusCode
		if status < 400 {
			status = fiber.StatusBadGateway
		}
		return c.Status(status).JSON(fiber.Map{"error": apiErr.Description, "razorpay_error": apiErr})
	}
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}
	log.Printf("internal error on %s %s: %v", c.Method(), c.Path(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
}

// logSave reports DB failures without failing the request: the object already
// exists in Razorpay, and a webhook or a later fetch will save it again.
func logSave(kind, id string, err error) {
	if err != nil {
		log.Printf("WARN could not save %s %s to postgres: %v", kind, id, err)
	}
}

// parseOptionalBody parses JSON only when a body was sent.
func parseOptionalBody(c *fiber.Ctx, out any) error {
	if len(c.Body()) == 0 {
		return nil
	}
	return parseBody(c, out)
}

func parseBody(c *fiber.Ctx, out any) error {
	if err := c.BodyParser(out); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body (send Content-Type: application/json): "+err.Error())
	}
	return nil
}

func badRequest(msg string) error { return fiber.NewError(fiber.StatusBadRequest, msg) }

const minAmount = 100 // ₹1 in paise
