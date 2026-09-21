package handlers

import (
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	"razorpay-playground/internal/store"
)

// ListLocal shows what is stored in Postgres, e.g. GET /api/local/payments?limit=20
func (h *Handler) ListLocal(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	if limit < 1 || limit > 200 {
		limit = 20
	}
	resource := c.Params("resource")
	if !store.IsListable(resource) {
		names := store.ListableResources()
		sort.Strings(names)
		return fiber.NewError(fiber.StatusNotFound, "unknown resource; use one of: "+strings.Join(names, ", "))
	}
	rows, err := h.store.ListRecent(c.UserContext(), resource, limit)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"count": len(rows), "items": rows})
}
