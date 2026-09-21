package handlers

import (
	"github.com/gofiber/fiber/v2"

	"payment-playground/internal/razorpay"
)

func (h *Handler) CreatePlan(c *fiber.Ctx) error {
	var req razorpay.CreatePlanRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	switch req.Period {
	case "daily", "weekly", "monthly", "yearly":
	default:
		return badRequest(`period must be one of daily, weekly, monthly, yearly`)
	}
	if req.Interval < 1 {
		req.Interval = 1
	}
	if req.Item.Name == "" || req.Item.Amount < minAmount {
		return badRequest("item.name and item.amount (paise, min 100) are required")
	}
	if req.Item.Currency == "" {
		req.Item.Currency = "INR"
	}
	plan, err := h.rp.CreatePlan(c.UserContext(), req)
	if err != nil {
		return err
	}
	logSave("plan", plan.ID, h.store.UpsertPlan(c.UserContext(), plan))
	return c.Status(fiber.StatusCreated).JSON(plan.Raw)
}

func (h *Handler) GetPlan(c *fiber.Ctx) error {
	plan, err := h.rp.FetchPlan(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("plan", plan.ID, h.store.UpsertPlan(c.UserContext(), plan))
	return c.JSON(plan.Raw)
}

func (h *Handler) CreateSubscription(c *fiber.Ctx) error {
	var req razorpay.CreateSubscriptionRequest
	if err := parseBody(c, &req); err != nil {
		return err
	}
	if req.PlanID == "" || req.TotalCount < 1 {
		return badRequest("plan_id and total_count (>= 1) are required")
	}
	sub, err := h.rp.CreateSubscription(c.UserContext(), req)
	if err != nil {
		return err
	}
	logSave("subscription", sub.ID, h.store.UpsertSubscription(c.UserContext(), sub))
	return c.Status(fiber.StatusCreated).JSON(sub.Raw)
}

func (h *Handler) GetSubscription(c *fiber.Ctx) error {
	sub, err := h.rp.FetchSubscription(c.UserContext(), c.Params("id"))
	if err != nil {
		return err
	}
	logSave("subscription", sub.ID, h.store.UpsertSubscription(c.UserContext(), sub))
	return c.JSON(sub.Raw)
}

func (h *Handler) CancelSubscription(c *fiber.Ctx) error {
	var body struct {
		CancelAtCycleEnd bool `json:"cancel_at_cycle_end"`
	}
	if err := parseOptionalBody(c, &body); err != nil {
		return err
	}
	sub, err := h.rp.CancelSubscription(c.UserContext(), c.Params("id"), body.CancelAtCycleEnd)
	if err != nil {
		return err
	}
	logSave("subscription", sub.ID, h.store.UpsertSubscription(c.UserContext(), sub))
	return c.JSON(sub.Raw)
}

type verifySubscriptionInput struct {
	PaymentID      string `json:"razorpay_payment_id"`
	SubscriptionID string `json:"razorpay_subscription_id"`
	Signature      string `json:"razorpay_signature"`
}

// VerifySubscription checks the Checkout response after the customer authorises a subscription.
func (h *Handler) VerifySubscription(c *fiber.Ctx) error {
	var in verifySubscriptionInput
	if err := parseBody(c, &in); err != nil {
		return err
	}
	if in.PaymentID == "" || in.SubscriptionID == "" || in.Signature == "" {
		return badRequest("razorpay_payment_id, razorpay_subscription_id and razorpay_signature are required")
	}
	if !razorpay.VerifySubscriptionSignature(in.PaymentID, in.SubscriptionID, in.Signature, h.rp.KeySecret()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"verified": false, "error": "signature mismatch"})
	}
	ctx := c.UserContext()
	sub, err := h.rp.FetchSubscription(ctx, in.SubscriptionID)
	if err != nil {
		return err
	}
	logSave("subscription", sub.ID, h.store.UpsertSubscription(ctx, sub))
	if p, err := h.rp.FetchPayment(ctx, in.PaymentID); err == nil {
		logSave("payment", p.ID, h.store.UpsertPayment(ctx, p))
	}
	return c.JSON(fiber.Map{
		"verified":        true,
		"subscription_id": sub.ID,
		"status":          sub.Status,
		"payment_id":      in.PaymentID,
	})
}
