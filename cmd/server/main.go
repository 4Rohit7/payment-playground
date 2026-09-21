package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"payment-playground/internal/config"
	"payment-playground/internal/db"
	"payment-playground/internal/handlers"
	"payment-playground/internal/razorpay"
	"payment-playground/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	h := handlers.New(cfg, razorpay.NewClient(cfg.RazorpayKeyID, cfg.RazorpayKeySecret), store.New(pool))

	app := fiber.New(fiber.Config{
		AppName:      "payment-playground",
		ErrorHandler: handlers.ErrorHandler,
	})
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use("/api", cors.New(cors.Config{AllowOrigins: cfg.CORSOrigins}))

	h.Register(app)
	app.Static("/", "./web", fiber.Static{Index: "checkout.html"}) // test page at http://localhost:PORT/

	go func() {
		<-ctx.Done()
		log.Println("shutting down...")
		_ = app.ShutdownWithTimeout(5 * time.Second)
	}()

	if cfg.RazorpayWebhookSecret == "" {
		log.Println("WARN RAZORPAY_WEBHOOK_SECRET is empty: /webhooks/razorpay will reject events")
	}
	log.Printf("listening on :%s (test page: http://localhost:%s/)", cfg.Port, cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
