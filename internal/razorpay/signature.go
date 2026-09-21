package razorpay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func hmacHex(secret string, msg []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(msg)
	return hex.EncodeToString(m.Sum(nil))
}

func equal(expected, got string) bool {
	return got != "" && hmac.Equal([]byte(expected), []byte(got))
}

// VerifyPaymentSignature checks the Checkout handler response for an order payment.
// Signed string: order_id + "|" + payment_id, keyed with the API key secret.
func VerifyPaymentSignature(orderID, paymentID, signature, keySecret string) bool {
	return equal(hmacHex(keySecret, []byte(orderID+"|"+paymentID)), signature)
}

// VerifySubscriptionSignature checks the Checkout response for a subscription payment.
// Signed string: payment_id + "|" + subscription_id.
func VerifySubscriptionSignature(paymentID, subscriptionID, signature, keySecret string) bool {
	return equal(hmacHex(keySecret, []byte(paymentID+"|"+subscriptionID)), signature)
}

// VerifyPaymentLinkSignature checks the query params Razorpay adds to a payment link callback.
// Signed string: link_id|reference_id|status|payment_id.
func VerifyPaymentLinkSignature(linkID, referenceID, status, paymentID, signature, keySecret string) bool {
	msg := linkID + "|" + referenceID + "|" + status + "|" + paymentID
	return equal(hmacHex(keySecret, []byte(msg)), signature)
}

// VerifyWebhookSignature checks X-Razorpay-Signature against the RAW request body,
// keyed with the webhook secret (not the API secret).
func VerifyWebhookSignature(body []byte, signature, webhookSecret string) bool {
	return equal(hmacHex(webhookSecret, body), signature)
}
