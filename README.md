# Razorpay Playground (Go + Fiber + Postgres)

A backend for testing every Razorpay flow in **test mode** before building a real frontend:
Orders + Checkout (card, UPI, netbanking, wallet, EMI, pay later), capture, refunds,
Payment Links, UPI QR codes, Plans + Subscriptions, Customers, and signed Webhooks.
Everything Razorpay returns is stored in Postgres so you can inspect it in pgAdmin.

## 1. Setup

1. **Database** – in pgAdmin create a database named `razorpay_playground`. Tables are created automatically on startup.
2. **Env** – `cp .env.example .env`, then fill in `DATABASE_URL`, `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET`.
3. **Run**
   ```bash
   go mod tidy
   go run ./cmd/server
   ```
4. Open **http://localhost:8080/** for the test console, or import `postman/razorpay-playground.postman_collection.json`.

The server refuses to start with a live key (`rzp_live_...`) unless `ALLOW_LIVE_KEYS=true`.

## 2. Webhooks (needed to see async updates)

Razorpay must reach your laptop, so expose the port with a tunnel:

```bash
ngrok http 8080              # or: cloudflared tunnel --url http://localhost:8080
```

In the Razorpay Dashboard (switch to **Test Mode**) go to Account & Settings > Webhooks > Add new webhook:

- URL: `https://<your-tunnel>/webhooks/razorpay`
- Secret: any long string, and put the same value in `RAZORPAY_WEBHOOK_SECRET`
- Events: `payment.authorized`, `payment.captured`, `payment.failed`, `order.paid`,
  `refund.created`, `refund.processed`, `refund.failed`,
  `payment_link.paid`, `payment_link.partially_paid`, `payment_link.cancelled`, `payment_link.expired`,
  `qr_code.credited`, `qr_code.closed`,
  `subscription.authenticated`, `subscription.activated`, `subscription.charged`,
  `subscription.pending`, `subscription.halted`, `subscription.cancelled`, `subscription.completed`

Every delivery is saved in `rp_webhook_events` and deduplicated by `X-Razorpay-Event-Id`.
If processing fails the server returns 500, so Razorpay retries.

Payment link callbacks are browser redirects, so `PUBLIC_BASE_URL=http://localhost:8080` works for them.

## 3. Test data per payment method
## Test credentials (test mode only)

### Card
| Field | Value |
|---|---|
| Card number | `5180 2872 0009 1001` |
| Expiry | `12/30` (or any future date) |
| CVV | `123` |
| OTP on bank page | `123456` |

> ⚠️ When Checkout asks **"Save card / Secure my card"**, always pick **"Maybe later"**.
> If it still asks for an OTP, enter `123456`.

### UPI
| VPA | Result |
|---|---|
| `success@razorpay` | Succeeds |
| `failure@razorpay` | Fails |

> ℹ️ After entering the UPI ID, a QR appears briefly and the payment auto-accepts — no UPI app needed in test mode.

### Netbanking and wallets
Pick any bank or wallet — Razorpay shows a mock page with **Success** and **Failure** buttons. No real login needed.

### For live mode
Replace test keys with live keys and set `ALLOW_LIVE_KEYS=true`.

| Method | How to test |
|---|---|
| UPI | VPA `success@razorpay` for success, `failure@razorpay` for failure |
| Netbanking | Pick any bank; the test bank page lets you choose Success or Failure |
| Card | Use the cards from Razorpay's "Test Card Details" docs page; any future expiry, any CVV; the test OTP page lets you choose success or failure |
| Wallet / Pay later / EMI | Pick any provider; a test page lets you choose the outcome |

On the test console, the **Method** dropdown opens Checkout directly on one method so you can test them one by one.
`GET /api/methods` shows which methods your account actually has enabled.

## 4. API

| Method | Path | What it does |
|---|---|---|
| GET | `/api/config` | Public key id for Checkout (never the secret) |
| GET | `/api/methods` | Payment methods enabled on the account |
| POST | `/api/customers` | Create (or fetch existing) customer |
| GET | `/api/customers/:id` | Fetch customer |
| POST | `/api/orders` | Create order (`amount` in paise) |
| GET | `/api/orders/:id` | Fetch order |
| GET | `/api/orders/:id/payments` | All attempts on an order, failed ones included |
| POST | `/api/payments/verify` | Verify Checkout signature, then fetch and save the payment |
| GET | `/api/payments/:id` | Fetch payment |
| POST | `/api/payments/:id/capture` | Capture an authorized payment (empty body = full amount) |
| POST | `/api/payments/:id/refunds` | Refund (empty body = full; `amount`, `speed`: normal/optimum) |
| GET | `/api/payments/:id/refunds` | Refunds of a payment |
| GET | `/api/refunds/:id` | Fetch refund |
| POST | `/api/payment-links` | Create payment link (callback defaults to this server) |
| GET | `/api/payment-links/callback` | Browser redirect after paying a link (signature verified) |
| GET | `/api/payment-links/:id` | Fetch link |
| POST | `/api/payment-links/:id/cancel` | Cancel link |
| POST | `/api/qr-codes` | Create UPI QR (fixed or any amount, single or multiple use) |
| GET | `/api/qr-codes/:id` | Fetch QR |
| GET | `/api/qr-codes/:id/payments` | Payments received on a QR |
| POST | `/api/qr-codes/:id/close` | Close QR |
| POST | `/api/plans` | Create plan |
| GET | `/api/plans/:id` | Fetch plan |
| POST | `/api/subscriptions` | Create subscription |
| POST | `/api/subscriptions/verify` | Verify Checkout signature for a subscription |
| GET | `/api/subscriptions/:id` | Fetch subscription |
| POST | `/api/subscriptions/:id/cancel` | Cancel (`cancel_at_cycle_end` optional) |
| GET | `/api/local/:resource` | What is stored in Postgres (`payments`, `orders`, `refunds`, `payment-links`, `qr-codes`, `plans`, `subscriptions`, `customers`, `webhook-events`) |
| POST | `/webhooks/razorpay` | Razorpay webhook receiver |

Razorpay errors are passed through with their status code, for example:
`{"error": "...", "razorpay_error": {"code": "BAD_REQUEST_ERROR", "field": "amount", ...}}`

## 5. Things worth knowing

- **Amounts are in paise.** ₹500 is `50000`; the minimum is `100`.
- **Never trust the frontend alone.** A payment is only "paid" after `/api/payments/verify` or a webhook.
- **Auto-capture** is set in Dashboard > Account & Settings > Payment capture. If it is off, payments stay
  `authorized` until you call capture, and Razorpay auto-refunds them if you do not.
- **Refunds** only work on captured payments.
- **QR codes and Subscriptions** may need to be enabled on your account. If the API says the feature is not
  enabled, turn it on from the dashboard or ask Razorpay support.
- **Out-of-order webhooks** are handled: a late `payment.authorized` will not overwrite a `captured` payment,
  and a `paid` order will not go back to `attempted`.
- RazorpayX Payouts use a separate product and account and are not covered here.

## 6. Project structure

```
cmd/server/main.go              entry point: config, DB, routes, graceful shutdown
internal/config/                env loading and test-key safety check
internal/db/                    pgx pool + embedded SQL migrations
internal/razorpay/              small REST client, entity types, signature helpers
internal/store/                 Postgres upserts, webhook idempotency, record listing
internal/handlers/              HTTP handlers per feature + webhook receiver
web/checkout.html               test console (Checkout, links, QR, refunds, subscriptions)
postman/                        Postman collection with auto-saved IDs
```
