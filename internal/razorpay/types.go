package razorpay

import "encoding/json"

// Amounts are always in the smallest currency unit (paise for INR). ₹1 = 100.
//
// Notes are json.RawMessage on purpose: Razorpay returns {} when notes exist
// but [] when empty, which would break a map[string]string field.

// ---------- Entities ----------

type Customer struct {
	withRaw
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	Contact   string          `json:"contact"`
	GSTIN     string          `json:"gstin"`
	Notes     json.RawMessage `json:"notes"`
	CreatedAt int64           `json:"created_at"`
}

type Order struct {
	withRaw
	ID         string          `json:"id"`
	Amount     int64           `json:"amount"`
	AmountPaid int64           `json:"amount_paid"`
	AmountDue  int64           `json:"amount_due"`
	Currency   string          `json:"currency"`
	Receipt    string          `json:"receipt"`
	Status     string          `json:"status"` // created | attempted | paid
	Attempts   int             `json:"attempts"`
	Notes      json.RawMessage `json:"notes"`
	CreatedAt  int64           `json:"created_at"`
}

type Payment struct {
	withRaw
	ID               string          `json:"id"`
	Amount           int64           `json:"amount"`
	Currency         string          `json:"currency"`
	Status           string          `json:"status"` // created | authorized | captured | refunded | failed
	OrderID          string          `json:"order_id"`
	InvoiceID        string          `json:"invoice_id"`
	Method           string          `json:"method"` // card | upi | netbanking | wallet | emi | paylater | ...
	AmountRefunded   int64           `json:"amount_refunded"`
	RefundStatus     string          `json:"refund_status"`
	Captured         bool            `json:"captured"`
	Description      string          `json:"description"`
	CardID           string          `json:"card_id"`
	Bank             string          `json:"bank"`
	Wallet           string          `json:"wallet"`
	VPA              string          `json:"vpa"`
	Email            string          `json:"email"`
	Contact          string          `json:"contact"`
	CustomerID       string          `json:"customer_id"`
	ErrorCode        string          `json:"error_code"`
	ErrorDescription string          `json:"error_description"`
	ErrorReason      string          `json:"error_reason"`
	Notes            json.RawMessage `json:"notes"`
	CreatedAt        int64           `json:"created_at"`
}

type Refund struct {
	withRaw
	ID             string          `json:"id"`
	PaymentID      string          `json:"payment_id"`
	Amount         int64           `json:"amount"`
	Currency       string          `json:"currency"`
	Status         string          `json:"status"` // pending | processed | failed
	Receipt        string          `json:"receipt"`
	SpeedRequested string          `json:"speed_requested"`
	SpeedProcessed string          `json:"speed_processed"`
	Notes          json.RawMessage `json:"notes"`
	CreatedAt      int64           `json:"created_at"`
}

type PaymentLink struct {
	withRaw
	ID          string          `json:"id"`
	Amount      int64           `json:"amount"`
	AmountPaid  int64           `json:"amount_paid"`
	Currency    string          `json:"currency"`
	Status      string          `json:"status"` // created | partially_paid | paid | cancelled | expired
	ReferenceID string          `json:"reference_id"`
	ShortURL    string          `json:"short_url"`
	Description string          `json:"description"`
	ExpireBy    int64           `json:"expire_by"`
	Notes       json.RawMessage `json:"notes"`
	CreatedAt   int64           `json:"created_at"`
}

type QRCode struct {
	withRaw
	ID                     string          `json:"id"`
	Name                   string          `json:"name"`
	Usage                  string          `json:"usage"` // single_use | multiple_use
	Type                   string          `json:"type"`  // upi_qr
	ImageURL               string          `json:"image_url"`
	PaymentAmount          int64           `json:"payment_amount"`
	FixedAmount            bool            `json:"fixed_amount"`
	Status                 string          `json:"status"` // active | closed
	Description            string          `json:"description"`
	CustomerID             string          `json:"customer_id"`
	PaymentsAmountReceived int64           `json:"payments_amount_received"`
	PaymentsCountReceived  int             `json:"payments_count_received"`
	CloseBy                int64           `json:"close_by"`
	ClosedAt               int64           `json:"closed_at"`
	CloseReason            string          `json:"close_reason"`
	Notes                  json.RawMessage `json:"notes"`
	CreatedAt              int64           `json:"created_at"`
}

type PlanItem struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description,omitempty"`
}

type Plan struct {
	withRaw
	ID        string          `json:"id"`
	Period    string          `json:"period"` // daily | weekly | monthly | yearly
	Interval  int             `json:"interval"`
	Item      PlanItem        `json:"item"`
	Notes     json.RawMessage `json:"notes"`
	CreatedAt int64           `json:"created_at"`
}

type Subscription struct {
	withRaw
	ID           string          `json:"id"`
	PlanID       string          `json:"plan_id"`
	CustomerID   string          `json:"customer_id"`
	Status       string          `json:"status"` // created | authenticated | active | pending | halted | cancelled | completed | expired | paused
	TotalCount   int             `json:"total_count"`
	PaidCount    int             `json:"paid_count"`
	CurrentStart int64           `json:"current_start"`
	CurrentEnd   int64           `json:"current_end"`
	ChargeAt     int64           `json:"charge_at"`
	ShortURL     string          `json:"short_url"`
	Notes        json.RawMessage `json:"notes"`
	CreatedAt    int64           `json:"created_at"`
}

// ---------- Requests ----------

type CreateCustomerRequest struct {
	Name         string            `json:"name"`
	Email        string            `json:"email,omitempty"`
	Contact      string            `json:"contact,omitempty"`
	GSTIN        string            `json:"gstin,omitempty"`
	FailExisting string            `json:"fail_existing,omitempty"` // "0" returns the existing customer instead of erroring
	Notes        map[string]string `json:"notes,omitempty"`
}

type CreateOrderRequest struct {
	Amount                int64             `json:"amount"`
	Currency              string            `json:"currency"`
	Receipt               string            `json:"receipt,omitempty"`
	PartialPayment        bool              `json:"partial_payment,omitempty"`
	FirstPaymentMinAmount int64             `json:"first_payment_min_amount,omitempty"`
	Notes                 map[string]string `json:"notes,omitempty"`
}

type CapturePaymentRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type CreateRefundRequest struct {
	Amount  int64             `json:"amount,omitempty"` // omit for a full refund
	Speed   string            `json:"speed,omitempty"`  // normal | optimum
	Receipt string            `json:"receipt,omitempty"`
	Notes   map[string]string `json:"notes,omitempty"`
}

type LinkCustomer struct {
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Contact string `json:"contact,omitempty"`
}

type LinkNotify struct {
	SMS   bool `json:"sms"`
	Email bool `json:"email"`
}

type CreatePaymentLinkRequest struct {
	Amount                int64             `json:"amount"`
	Currency              string            `json:"currency"`
	AcceptPartial         bool              `json:"accept_partial,omitempty"`
	FirstMinPartialAmount int64             `json:"first_min_partial_amount,omitempty"`
	ExpireBy              int64             `json:"expire_by,omitempty"`
	ReferenceID           string            `json:"reference_id,omitempty"`
	Description           string            `json:"description,omitempty"`
	Customer              *LinkCustomer     `json:"customer,omitempty"`
	Notify                *LinkNotify       `json:"notify,omitempty"`
	ReminderEnable        bool              `json:"reminder_enable,omitempty"`
	Notes                 map[string]string `json:"notes,omitempty"`
	CallbackURL           string            `json:"callback_url,omitempty"`
	CallbackMethod        string            `json:"callback_method,omitempty"`
	// Passed through as-is, e.g. {"checkout":{"method":{"upi":"1","card":"0"}}} to restrict methods.
	Options json.RawMessage `json:"options,omitempty"`
}

type CreateQRCodeRequest struct {
	Type          string            `json:"type"`
	Name          string            `json:"name,omitempty"`
	Usage         string            `json:"usage"`
	FixedAmount   bool              `json:"fixed_amount"`
	PaymentAmount int64             `json:"payment_amount,omitempty"`
	Description   string            `json:"description,omitempty"`
	CustomerID    string            `json:"customer_id,omitempty"`
	CloseBy       int64             `json:"close_by,omitempty"`
	Notes         map[string]string `json:"notes,omitempty"`
}

type CreatePlanRequest struct {
	Period   string            `json:"period"`
	Interval int               `json:"interval"`
	Item     PlanItem          `json:"item"`
	Notes    map[string]string `json:"notes,omitempty"`
}

type CreateSubscriptionRequest struct {
	PlanID         string            `json:"plan_id"`
	TotalCount     int               `json:"total_count"`
	Quantity       int               `json:"quantity,omitempty"`
	CustomerNotify *bool             `json:"customer_notify,omitempty"`
	StartAt        int64             `json:"start_at,omitempty"`
	ExpireBy       int64             `json:"expire_by,omitempty"`
	Notes          map[string]string `json:"notes,omitempty"`
}

// ---------- Webhooks ----------

type WebhookEvent struct {
	Entity    string   `json:"entity"`
	AccountID string   `json:"account_id"`
	Event     string   `json:"event"` // e.g. payment.captured
	Contains  []string `json:"contains"`
	// Keys are entity names: payment, order, refund, payment_link, qr_code, subscription, ...
	Payload map[string]struct {
		Entity json.RawMessage `json:"entity"`
	} `json:"payload"`
	CreatedAt int64 `json:"created_at"`
}
