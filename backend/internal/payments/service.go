package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
	"github.com/stripe/stripe-go/v78/webhook"
)

var ErrStripeNotConfigured = errors.New("Stripe is not configured")
var ErrInvalidWebhook = errors.New("invalid Stripe webhook")
var ErrInvalidPaymentRequest = errors.New("invalid payment request")
var ErrPaymentNotPayable = errors.New("payment is not payable")

type Service struct {
	repo          *Repository
	stripeSecret  string
	webhookSecret string
}

func NewService(repo *Repository, stripeSecret, webhookSecret string) *Service {
	return &Service{repo: repo, stripeSecret: stripeSecret, webhookSecret: webhookSecret}
}

type IntentResponse struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
}

func (s *Service) CreatePaymentIntent(ctx context.Context, userID, orderID string) (IntentResponse, error) {
	if s.repo == nil || strings.TrimSpace(s.stripeSecret) == "" {
		return IntentResponse{}, ErrStripeNotConfigured
	}
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(orderID) == "" {
		return IntentResponse{}, ErrInvalidPaymentRequest
	}
	payment, currency, err := s.repo.GetOrderPayment(ctx, userID, orderID)
	if err != nil {
		return IntentResponse{}, fmt.Errorf("load customer payment: %w", err)
	}
	if payment.AmountMinor <= 0 || currency != "INR" ||
		(payment.Status != "pending" && payment.Status != "failed") {
		return IntentResponse{}, ErrPaymentNotPayable
	}

	stripe.Key = s.stripeSecret
	if payment.ProviderRef != "" {
		intent, err := paymentintent.Get(payment.ProviderRef, &stripe.PaymentIntentParams{
			Params: stripe.Params{Context: ctx},
		})
		if err != nil {
			return IntentResponse{}, fmt.Errorf("retrieve Stripe payment intent: %w", err)
		}
		if err := validatePaymentIntent(intent, payment, orderID); err != nil {
			return IntentResponse{}, err
		}
		if intent.Status == stripe.PaymentIntentStatusCanceled || intent.ClientSecret == "" {
			return IntentResponse{}, errors.New("payment intent is no longer payable")
		}
		return IntentResponse{ID: intent.ID, ClientSecret: intent.ClientSecret}, nil
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(payment.AmountMinor),
		Currency: stripe.String(strings.ToLower(currency)),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		Params: stripe.Params{Context: ctx},
	}
	params.SetIdempotencyKey("omniretail-payment-" + payment.ID)
	params.AddMetadata("order_id", payment.OrderID)
	params.AddMetadata("payment_id", payment.ID)
	intent, err := paymentintent.New(params)
	if err != nil {
		return IntentResponse{}, fmt.Errorf("create Stripe payment intent: %w", err)
	}
	if err := validatePaymentIntent(intent, payment, orderID); err != nil {
		return IntentResponse{}, err
	}
	if intent.ClientSecret == "" {
		return IntentResponse{}, errors.New("Stripe returned an empty client secret")
	}
	if err := s.repo.SaveProviderRef(ctx, payment.ID, intent.ID); err != nil {
		return IntentResponse{}, fmt.Errorf("save Stripe payment reference: %w", err)
	}
	return IntentResponse{ID: intent.ID, ClientSecret: intent.ClientSecret}, nil
}

func validatePaymentIntent(intent *stripe.PaymentIntent, payment Payment, orderID string) error {
	if intent == nil || intent.ID == "" ||
		intent.Amount != payment.AmountMinor ||
		!strings.EqualFold(string(intent.Currency), "inr") ||
		intent.Metadata["order_id"] != orderID ||
		intent.Metadata["payment_id"] != payment.ID {
		return errors.New("Stripe payment intent does not match the stored payment")
	}
	return nil
}

func (s *Service) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	if s.repo == nil || strings.TrimSpace(s.webhookSecret) == "" {
		return ErrStripeNotConfigured
	}
	if strings.TrimSpace(signature) == "" {
		return fmt.Errorf("%w: signature is required", ErrInvalidWebhook)
	}
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return fmt.Errorf("%w: signature verification failed", ErrInvalidWebhook)
	}
	if event.ID == "" {
		return fmt.Errorf("%w: event ID is required", ErrInvalidWebhook)
	}
	switch event.Type {
	case "payment_intent.succeeded", "payment_intent.payment_failed", "payment_intent.canceled":
	default:
		return nil
	}
	var intent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &intent); err != nil {
		return fmt.Errorf("%w: invalid event payload", ErrInvalidWebhook)
	}
	err = s.repo.ProcessStripeEvent(ctx, event.ID, string(event.Type), StripeEventIntent{
		ID:             intent.ID,
		Amount:         intent.Amount,
		AmountReceived: intent.AmountReceived,
		Currency:       strings.ToLower(string(intent.Currency)),
		OrderID:        intent.Metadata["order_id"],
		PaymentID:      intent.Metadata["payment_id"],
	})
	if err != nil && errors.Is(err, ErrInvalidStripeEvent) {
		return fmt.Errorf("%w: payment association or transition failed", ErrInvalidWebhook)
	}
	return err
}
