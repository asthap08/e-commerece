package payments

import (
	"context"
	"errors"
	"testing"
)

func TestHandleWebhookRequiresSecretAndSignature(t *testing.T) {
	service := &Service{}
	if err := service.HandleWebhook(context.Background(), []byte(`{}`), ""); err == nil {
		t.Fatal("expected webhook without configured secret to fail")
	}

	service.webhookSecret = "whsec_test"
	service.repo = &Repository{}
	if err := service.HandleWebhook(context.Background(), []byte(`{}`), ""); err == nil {
		t.Fatal("expected webhook without signature header to fail")
	}
	if err := service.HandleWebhook(context.Background(), []byte(`{}`), "t=1,v1=invalid"); !errors.Is(err, ErrInvalidWebhook) {
		t.Fatalf("invalid signature error = %v, want invalid webhook", err)
	}
}
