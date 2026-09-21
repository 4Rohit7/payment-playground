-- No foreign keys on purpose: webhooks can arrive in any order
-- (e.g. a payment before its order is saved), so rows must stand alone.

CREATE TABLE IF NOT EXISTS rp_customers (
    id          TEXT PRIMARY KEY,
    name        TEXT,
    email       TEXT,
    contact     TEXT,
    raw         JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rp_orders (
    id          TEXT PRIMARY KEY,
    amount      BIGINT NOT NULL,
    amount_paid BIGINT NOT NULL DEFAULT 0,
    amount_due  BIGINT NOT NULL DEFAULT 0,
    currency    TEXT NOT NULL,
    receipt     TEXT,
    status      TEXT NOT NULL,
    attempts    INT NOT NULL DEFAULT 0,
    raw         JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rp_payments (
    id                TEXT PRIMARY KEY,
    order_id          TEXT,
    amount            BIGINT NOT NULL,
    amount_refunded   BIGINT NOT NULL DEFAULT 0,
    currency          TEXT NOT NULL,
    status            TEXT NOT NULL,
    method            TEXT,
    captured          BOOLEAN NOT NULL DEFAULT false,
    email             TEXT,
    contact           TEXT,
    bank              TEXT,
    wallet            TEXT,
    vpa               TEXT,
    error_code        TEXT,
    error_description TEXT,
    raw               JSONB NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_rp_payments_order_id ON rp_payments (order_id);
CREATE INDEX IF NOT EXISTS idx_rp_payments_method   ON rp_payments (method);

CREATE TABLE IF NOT EXISTS rp_refunds (
    id              TEXT PRIMARY KEY,
    payment_id      TEXT NOT NULL,
    amount          BIGINT NOT NULL,
    currency        TEXT NOT NULL,
    status          TEXT NOT NULL,
    speed_requested TEXT,
    speed_processed TEXT,
    raw             JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_rp_refunds_payment_id ON rp_refunds (payment_id);

CREATE TABLE IF NOT EXISTS rp_payment_links (
    id           TEXT PRIMARY KEY,
    amount       BIGINT NOT NULL,
    amount_paid  BIGINT NOT NULL DEFAULT 0,
    currency     TEXT NOT NULL,
    status       TEXT NOT NULL,
    reference_id TEXT,
    short_url    TEXT,
    raw          JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rp_qr_codes (
    id                       TEXT PRIMARY KEY,
    usage                    TEXT,
    fixed_amount             BOOLEAN NOT NULL DEFAULT false,
    payment_amount           BIGINT,
    payments_amount_received BIGINT NOT NULL DEFAULT 0,
    payments_count_received  INT NOT NULL DEFAULT 0,
    status                   TEXT NOT NULL,
    image_url                TEXT,
    close_reason             TEXT,
    raw                      JSONB NOT NULL,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rp_plans (
    id               TEXT PRIMARY KEY,
    period           TEXT NOT NULL,
    billing_interval INT NOT NULL,
    item_name        TEXT,
    item_amount      BIGINT,
    currency         TEXT,
    raw              JSONB NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rp_subscriptions (
    id          TEXT PRIMARY KEY,
    plan_id     TEXT,
    customer_id TEXT,
    status      TEXT NOT NULL,
    total_count INT,
    paid_count  INT NOT NULL DEFAULT 0,
    short_url   TEXT,
    raw         JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every webhook delivery, keyed by X-Razorpay-Event-Id for idempotency.
CREATE TABLE IF NOT EXISTS rp_webhook_events (
    event_id     TEXT PRIMARY KEY,
    event        TEXT NOT NULL,
    payload      JSONB NOT NULL,
    processed    BOOLEAN NOT NULL DEFAULT false,
    attempts     INT NOT NULL DEFAULT 1,
    last_error   TEXT,
    received_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_rp_webhook_events_event ON rp_webhook_events (event);
