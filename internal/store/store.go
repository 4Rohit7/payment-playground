// Package store persists Razorpay entities in Postgres.
// Every upsert keeps the full Razorpay JSON in a `raw` JSONB column,
// plus a few typed columns that are handy to query in pgAdmin.
package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"payment-playground/internal/razorpay"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func rawJSON(r json.RawMessage) string {
	if len(r) == 0 {
		return "{}"
	}
	return string(r)
}

func (s *Store) UpsertCustomer(ctx context.Context, c *razorpay.Customer) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rp_customers (id, name, email, contact, raw)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, email = EXCLUDED.email, contact = EXCLUDED.contact,
			raw = EXCLUDED.raw, updated_at = now()`,
		c.ID, c.Name, c.Email, c.Contact, rawJSON(c.Raw))
	return err
}

func (s *Store) UpsertOrder(ctx context.Context, o *razorpay.Order) error {
	// The WHERE guard stops a late, out-of-order event from moving a paid order backwards.
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rp_orders (id, amount, amount_paid, amount_due, currency, receipt, status, attempts, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			amount_paid = EXCLUDED.amount_paid, amount_due = EXCLUDED.amount_due,
			status = EXCLUDED.status, attempts = EXCLUDED.attempts,
			raw = EXCLUDED.raw, updated_at = now()
		WHERE NOT (rp_orders.status = 'paid' AND EXCLUDED.status <> 'paid')`,
		o.ID, o.Amount, o.AmountPaid, o.AmountDue, o.Currency, o.Receipt, o.Status, o.Attempts, rawJSON(o.Raw))
	return err
}

func (s *Store) UpsertPayment(ctx context.Context, p *razorpay.Payment) error {
	// A payment can go failed -> authorized (late authorization), so only
	// captured/refunded are protected from being overwritten by older states.
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rp_payments (id, order_id, amount, amount_refunded, currency, status, method, captured,
		                         email, contact, bank, wallet, vpa, error_code, error_description, raw)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, NULLIF($7, ''), $8,
		        NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''),
		        NULLIF($14, ''), NULLIF($15, ''), $16)
		ON CONFLICT (id) DO UPDATE SET
			order_id = COALESCE(EXCLUDED.order_id, rp_payments.order_id),
			amount_refunded = EXCLUDED.amount_refunded, status = EXCLUDED.status,
			method = COALESCE(EXCLUDED.method, rp_payments.method), captured = EXCLUDED.captured,
			email = EXCLUDED.email, contact = EXCLUDED.contact, bank = EXCLUDED.bank,
			wallet = EXCLUDED.wallet, vpa = EXCLUDED.vpa,
			error_code = EXCLUDED.error_code, error_description = EXCLUDED.error_description,
			raw = EXCLUDED.raw, updated_at = now()
		WHERE NOT (rp_payments.status IN ('captured', 'refunded') AND EXCLUDED.status IN ('created', 'authorized'))`,
		p.ID, p.OrderID, p.Amount, p.AmountRefunded, p.Currency, p.Status, p.Method, p.Captured,
		p.Email, p.Contact, p.Bank, p.Wallet, p.VPA, p.ErrorCode, p.ErrorDescription, rawJSON(p.Raw))
	return err
}

func (s *Store) UpsertRefund(ctx context.Context, r *razorpay.Refund) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rp_refunds (id, payment_id, amount, currency, status, speed_requested, speed_processed, raw)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			speed_processed = COALESCE(EXCLUDED.speed_processed, rp_refunds.speed_processed),
			raw = EXCLUDED.raw, updated_at = now()
		WHERE NOT (rp_refunds.status IN ('processed', 'failed') AND EXCLUDED.status = 'pending')`,
		r.ID, r.PaymentID, r.Amount, r.Currency, r.Status, r.SpeedRequested, r.SpeedProcessed, rawJSON(r.Raw))
	return err
}

func (s *Store) UpsertPaymentLink(ctx context.Context, l *razorpay.PaymentLink) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rp_payment_links (id, amount, amount_paid, currency, status, reference_id, short_url, raw)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			amount_paid = EXCLUDED.amount_paid, status = EXCLUDED.status,
			raw = EXCLUDED.raw, updated_at = now()
		WHERE NOT (rp_payment_links.status IN ('paid', 'cancelled', 'expired')
		           AND EXCLUDED.status IN ('created', 'partially_paid'))`,
		l.ID, l.Amount, l.AmountPaid, l.Currency, l.Status, l.ReferenceID, l.ShortURL, rawJSON(l.Raw))
	return err
}

func (s *Store) UpsertQRCode(ctx context.Context, q *razorpay.QRCode) error {
	var paymentAmount *int64
	if q.FixedAmount {
		paymentAmount = &q.PaymentAmount
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rp_qr_codes (id, usage, fixed_amount, payment_amount, payments_amount_received,
		                         payments_count_received, status, image_url, close_reason, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), $10)
		ON CONFLICT (id) DO UPDATE SET
			payments_amount_received = EXCLUDED.payments_amount_received,
			payments_count_received = EXCLUDED.payments_count_received,
			status = EXCLUDED.status, close_reason = EXCLUDED.close_reason,
			raw = EXCLUDED.raw, updated_at = now()
		WHERE NOT (rp_qr_codes.status = 'closed' AND EXCLUDED.status = 'active')`,
		q.ID, q.Usage, q.FixedAmount, paymentAmount, q.PaymentsAmountReceived,
		q.PaymentsCountReceived, q.Status, q.ImageURL, q.CloseReason, rawJSON(q.Raw))
	return err
}

func (s *Store) UpsertPlan(ctx context.Context, p *razorpay.Plan) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rp_plans (id, period, billing_interval, item_name, item_amount, currency, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET raw = EXCLUDED.raw, updated_at = now()`,
		p.ID, p.Period, p.Interval, p.Item.Name, p.Item.Amount, p.Item.Currency, rawJSON(p.Raw))
	return err
}

func (s *Store) UpsertSubscription(ctx context.Context, sub *razorpay.Subscription) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rp_subscriptions (id, plan_id, customer_id, status, total_count, paid_count, short_url, raw)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, rp_subscriptions.customer_id),
			status = EXCLUDED.status, paid_count = EXCLUDED.paid_count,
			raw = EXCLUDED.raw, updated_at = now()`,
		sub.ID, sub.PlanID, sub.CustomerID, sub.Status, sub.TotalCount, sub.PaidCount, sub.ShortURL, rawJSON(sub.Raw))
	return err
}

// ---------- Webhook events ----------

// RecordWebhookEvent saves a delivery (or bumps its attempt count on a retry)
// and reports whether that event was already processed successfully.
func (s *Store) RecordWebhookEvent(ctx context.Context, eventID, event string, payload []byte) (alreadyProcessed bool, err error) {
	err = s.pool.QueryRow(ctx, `
		INSERT INTO rp_webhook_events (event_id, event, payload)
		VALUES ($1, $2, $3)
		ON CONFLICT (event_id) DO UPDATE SET attempts = rp_webhook_events.attempts + 1
		RETURNING processed`,
		eventID, event, string(payload)).Scan(&alreadyProcessed)
	return alreadyProcessed, err
}

func (s *Store) FinishWebhookEvent(ctx context.Context, eventID string, procErr error) error {
	if procErr != nil {
		_, err := s.pool.Exec(ctx,
			`UPDATE rp_webhook_events SET last_error = $2 WHERE event_id = $1`, eventID, procErr.Error())
		return err
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE rp_webhook_events SET processed = true, processed_at = now(), last_error = NULL
		WHERE event_id = $1`, eventID)
	return err
}

// ---------- Local listing (to inspect what was stored) ----------

var listable = map[string]struct{ table, orderBy string }{
	"customers":      {"rp_customers", "updated_at"},
	"orders":         {"rp_orders", "updated_at"},
	"payments":       {"rp_payments", "updated_at"},
	"refunds":        {"rp_refunds", "updated_at"},
	"payment-links":  {"rp_payment_links", "updated_at"},
	"qr-codes":       {"rp_qr_codes", "updated_at"},
	"plans":          {"rp_plans", "updated_at"},
	"subscriptions":  {"rp_subscriptions", "updated_at"},
	"webhook-events": {"rp_webhook_events", "received_at"},
}

func IsListable(resource string) bool {
	_, ok := listable[resource]
	return ok
}

func ListableResources() []string {
	out := make([]string, 0, len(listable))
	for k := range listable {
		out = append(out, k)
	}
	return out
}

// ListRecent returns the latest rows of a resource as JSON, without the bulky raw/payload columns.
func (s *Store) ListRecent(ctx context.Context, resource string, limit int) ([]json.RawMessage, error) {
	t, ok := listable[resource]
	if !ok {
		return nil, fmt.Errorf("unknown resource %q", resource)
	}
	// Table and column names come from the whitelist above, never from user input.
	q := fmt.Sprintf(`SELECT (to_jsonb(t) - 'raw' - 'payload')::text FROM %s t ORDER BY %s DESC LIMIT $1`, t.table, t.orderBy)
	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []json.RawMessage{}
	for rows.Next() {
		var row string
		if err := rows.Scan(&row); err != nil {
			return nil, err
		}
		out = append(out, json.RawMessage(row))
	}
	return out, rows.Err()
}
