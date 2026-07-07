package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	smsplatform "github.com/AmirAbaris/weeto-backend/internal/platform/sms"
	"github.com/joho/godotenv"
)

func main() {
	phone := flag.String("phone", "", "recipient mobile number (e.g. 0912XXXXXXX)")
	code := flag.String("code", "12345", "verification code value for sandbox template")
	flag.Parse()

	if *phone == "" {
		log.Fatal("usage: go run ./cmd/sms-test -phone 0912XXXXXXX")
	}

	_ = godotenv.Load()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.SMSAPIKey == "" {
		log.Fatal("SMS_API_KEY is not set")
	}
	if cfg.SMSTemplateID == 0 {
		log.Fatal("SMS_TEMPLATE_ID is not set")
	}

	mobile, err := smsplatform.NormalizeMobile(*phone)
	if err != nil {
		log.Fatalf("phone: %v", err)
	}

	sender := smsplatform.NewSender(cfg)
	messageID, cost, err := sender.VerifySend(context.Background(), mobile, cfg.SMSTemplateID, []smsplatform.Parameter{
		{Name: "Code", Value: *code},
	})
	if err != nil {
		log.Fatalf("send: %v", err)
	}

	fmt.Printf("sent to %s: messageId=%d cost=%.1f\n", mobile, messageID, cost)
	os.Exit(0)
}
