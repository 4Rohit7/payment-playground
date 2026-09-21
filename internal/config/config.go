package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                  string
	PublicBaseURL         string
	CORSOrigins           string
	MerchantName          string
	DatabaseURL           string
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string
}

func Load() (*Config, error) {
	// .env is optional; real environment variables win.
	_ = godotenv.Load()

	c := &Config{
		Port:                  env("PORT", "8080"),
		PublicBaseURL:         strings.TrimRight(env("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		CORSOrigins:           env("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"),
		MerchantName:          env("MERCHANT_NAME", "Razorpay Playground"),
		DatabaseURL:           env("DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5432/razorpay_playground?sslmode=disable"),
		RazorpayKeyID:         os.Getenv("RAZORPAY_KEY_ID"),
		RazorpayKeySecret:     os.Getenv("RAZORPAY_KEY_SECRET"),
		RazorpayWebhookSecret: os.Getenv("RAZORPAY_WEBHOOK_SECRET"),
	}

	var missing []string
	for _, kv := range [][2]string{
		{"RAZORPAY_KEY_ID", c.RazorpayKeyID},
		{"RAZORPAY_KEY_SECRET", c.RazorpayKeySecret},
	} {
		if kv[1] == "" {
			missing = append(missing, kv[0])
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	if !strings.HasPrefix(c.RazorpayKeyID, "rzp_test_") && os.Getenv("ALLOW_LIVE_KEYS") != "true" {
		return nil, errors.New("RAZORPAY_KEY_ID is not a test key (rzp_test_...); set ALLOW_LIVE_KEYS=true only if you really mean it")
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
