package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	"payment-playground/internal/razorpay"
)

// RazorpayWebhook receives events configured in Dashboard > Webhooks.
// Flow: verify signature on the raw body -> record event (idempotent) -> upsert every entity in the payload.
// Returning non-2xx makes Razorpay retry, so we only do that when processing actually failed.
func (h *Handler) RazorpayWebhook(c *fiber.Ctx) error {
	if h.cfg.RazorpayWebhookSecret == "" {
		return fiber.NewError(fiber.StatusServiceUnavailable, "RAZORPAY_WEBHOOK_SECRET is not set")
	}

	body := append([]byte(nil), c.Body()...) // copy: fasthttp reuses the buffer
	if !razorpay.VerifyWebhookSignature(body, c.Get("X-Razorpay-Signature"), h.cfg.RazorpayWebhookSecret) {
		log.Printf("webhook rejected: bad signature")
		return fiber.NewError(fiber.StatusBadRequest, "invalid webhook signature")
	}

	var evt razorpay.WebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return badRequest("invalid webhook JSON")
	}

	eventID := c.Get("X-Razorpay-Event-Id")
	if eventID == "" {
		sum := sha256.Sum256(body)
		eventID = "sha256_" + hex.EncodeToString(sum[:])
	}

	ctx := c.UserContext()
	done, err := h.store.RecordWebhookEvent(ctx, eventID, evt.Event, body)
	if err != nil {
		return fmt.Errorf("record webhook event: %w", err)
	}
	if done {
		log.Printf("webhook %s (%s) already processed, skipping", eventID, evt.Event)
		return c.JSON(fiber.Map{"status": "duplicate"})
	}

	procErr := h.applyWebhook(ctx, &evt)
	if err := h.store.FinishWebhookEvent(ctx, eventID, procErr); err != nil {
		log.Printf("WARN could not update webhook event %s: %v", eventID, err)
	}
	if procErr != nil {
		log.Printf("webhook %s (%s) failed: %v", eventID, evt.Event, procErr)
		return fiber.NewError(fiber.StatusInternalServerError, "processing failed; Razorpay will retry")
	}

	log.Printf("webhook %s processed: %s", eventID, evt.Event)
	return c.JSON(fiber.Map{"status": "ok"})
}

// applyWebhook saves every entity found in the payload. Unknown entities are ignored.
func (h *Handler) applyWebhook(ctx context.Context, evt *razorpay.WebhookEvent) error {
	for name, wrapped := range evt.Payload {
		if len(wrapped.Entity) == 0 {
			continue
		}
		var err error
		switch name {
		case "payment":
			var p razorpay.Payment
			if err = razorpay.DecodeEntity(wrapped.Entity, &p); err == nil {
				err = h.store.UpsertPayment(ctx, &p)
			}
		case "order":
			var o razorpay.Order
			if err = razorpay.DecodeEntity(wrapped.Entity, &o); err == nil {
				err = h.store.UpsertOrder(ctx, &o)
			}
		case "refund":
			var r razorpay.Refund
			if err = razorpay.DecodeEntity(wrapped.Entity, &r); err == nil {
				err = h.store.UpsertRefund(ctx, &r)
			}
		case "payment_link":
			var l razorpay.PaymentLink
			if err = razorpay.DecodeEntity(wrapped.Entity, &l); err == nil {
				err = h.store.UpsertPaymentLink(ctx, &l)
			}
		case "qr_code":
			var q razorpay.QRCode
			if err = razorpay.DecodeEntity(wrapped.Entity, &q); err == nil {
				err = h.store.UpsertQRCode(ctx, &q)
			}
		case "subscription":
			var s razorpay.Subscription
			if err = razorpay.DecodeEntity(wrapped.Entity, &s); err == nil {
				err = h.store.UpsertSubscription(ctx, &s)
			}
		default:
			log.Printf("webhook %s: ignoring payload entity %q", evt.Event, name)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}
