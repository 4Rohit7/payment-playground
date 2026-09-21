package razorpay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

func esc(id string) string { return url.PathEscape(id) }

// ---------- Payment methods enabled on the account ----------

// FetchMethods returns which payment methods (card networks, banks, wallets, UPI...)
// are enabled for this key. This endpoint uses only the key id for auth.
func (c *Client) FetchMethods(ctx context.Context) (json.RawMessage, error) {
	return c.send(ctx, http.MethodGet, "/methods", nil, true)
}

// ---------- Customers ----------

func (c *Client) CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*Customer, error) {
	return call[Customer](c, ctx, http.MethodPost, "/customers", req)
}

func (c *Client) FetchCustomer(ctx context.Context, id string) (*Customer, error) {
	return call[Customer](c, ctx, http.MethodGet, "/customers/"+esc(id), nil)
}

// ---------- Orders ----------

func (c *Client) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
	return call[Order](c, ctx, http.MethodPost, "/orders", req)
}

func (c *Client) FetchOrder(ctx context.Context, id string) (*Order, error) {
	return call[Order](c, ctx, http.MethodGet, "/orders/"+esc(id), nil)
}

func (c *Client) FetchOrderPayments(ctx context.Context, orderID string) ([]*Payment, json.RawMessage, error) {
	return list[Payment](c, ctx, "/orders/"+esc(orderID)+"/payments")
}

// ---------- Payments ----------

func (c *Client) FetchPayment(ctx context.Context, id string) (*Payment, error) {
	return call[Payment](c, ctx, http.MethodGet, "/payments/"+esc(id), nil)
}

func (c *Client) CapturePayment(ctx context.Context, id string, req CapturePaymentRequest) (*Payment, error) {
	return call[Payment](c, ctx, http.MethodPost, "/payments/"+esc(id)+"/capture", req)
}

// ---------- Refunds ----------

func (c *Client) CreateRefund(ctx context.Context, paymentID string, req CreateRefundRequest) (*Refund, error) {
	return call[Refund](c, ctx, http.MethodPost, "/payments/"+esc(paymentID)+"/refund", req)
}

func (c *Client) FetchRefund(ctx context.Context, id string) (*Refund, error) {
	return call[Refund](c, ctx, http.MethodGet, "/refunds/"+esc(id), nil)
}

func (c *Client) FetchPaymentRefunds(ctx context.Context, paymentID string) ([]*Refund, json.RawMessage, error) {
	return list[Refund](c, ctx, "/payments/"+esc(paymentID)+"/refunds")
}

// ---------- Payment Links ----------

func (c *Client) CreatePaymentLink(ctx context.Context, req CreatePaymentLinkRequest) (*PaymentLink, error) {
	return call[PaymentLink](c, ctx, http.MethodPost, "/payment_links", req)
}

func (c *Client) FetchPaymentLink(ctx context.Context, id string) (*PaymentLink, error) {
	return call[PaymentLink](c, ctx, http.MethodGet, "/payment_links/"+esc(id), nil)
}

func (c *Client) CancelPaymentLink(ctx context.Context, id string) (*PaymentLink, error) {
	return call[PaymentLink](c, ctx, http.MethodPost, "/payment_links/"+esc(id)+"/cancel", nil)
}

// ---------- UPI QR Codes ----------

func (c *Client) CreateQRCode(ctx context.Context, req CreateQRCodeRequest) (*QRCode, error) {
	return call[QRCode](c, ctx, http.MethodPost, "/payments/qr_codes", req)
}

func (c *Client) FetchQRCode(ctx context.Context, id string) (*QRCode, error) {
	return call[QRCode](c, ctx, http.MethodGet, "/payments/qr_codes/"+esc(id), nil)
}

func (c *Client) FetchQRCodePayments(ctx context.Context, id string) ([]*Payment, json.RawMessage, error) {
	return list[Payment](c, ctx, "/payments/qr_codes/"+esc(id)+"/payments")
}

func (c *Client) CloseQRCode(ctx context.Context, id string) (*QRCode, error) {
	return call[QRCode](c, ctx, http.MethodPost, "/payments/qr_codes/"+esc(id)+"/close", nil)
}

// ---------- Plans & Subscriptions ----------

func (c *Client) CreatePlan(ctx context.Context, req CreatePlanRequest) (*Plan, error) {
	return call[Plan](c, ctx, http.MethodPost, "/plans", req)
}

func (c *Client) FetchPlan(ctx context.Context, id string) (*Plan, error) {
	return call[Plan](c, ctx, http.MethodGet, "/plans/"+esc(id), nil)
}

func (c *Client) CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*Subscription, error) {
	return call[Subscription](c, ctx, http.MethodPost, "/subscriptions", req)
}

func (c *Client) FetchSubscription(ctx context.Context, id string) (*Subscription, error) {
	return call[Subscription](c, ctx, http.MethodGet, "/subscriptions/"+esc(id), nil)
}

func (c *Client) CancelSubscription(ctx context.Context, id string, atCycleEnd bool) (*Subscription, error) {
	body := map[string]bool{"cancel_at_cycle_end": atCycleEnd}
	return call[Subscription](c, ctx, http.MethodPost, "/subscriptions/"+esc(id)+"/cancel", body)
}
