package external

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/webbies/otel-fiber-demo/internal/infrastructure/config"
)

type MTNPayClient struct {
	client *resty.Client
	config *config.MTNPayConfig
	tracer trace.Tracer
}

func NewMTNPayClient(cfg *config.MTNPayConfig) *MTNPayClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-API-Key", cfg.APIKey).
		SetTimeout(30 * time.Second)

	return &MTNPayClient{
		client: client,
		config: cfg,
		tracer: otel.Tracer("mtnpay-client"),
	}
}

type MTNPayRequest struct {
	Amount      float64           `json:"amount"`
	Currency    string            `json:"currency"`
	PhoneNumber string            `json:"phone_number"`
	Reference   string            `json:"reference"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type MTNPayResponse struct {
	TransactionID string            `json:"transaction_id"`
	Status        string            `json:"status"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	Reference     string            `json:"reference"`
	Message       string            `json:"message,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

type MTNPayStatusResponse struct {
	TransactionID string     `json:"transaction_id"`
	Status        string     `json:"status"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	Reference     string     `json:"reference"`
	Message       string     `json:"message,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty"`
}

func (c *MTNPayClient) ProcessPayment(ctx context.Context, req MTNPayRequest) (*MTNPayResponse, error) {
	ctx, span := c.tracer.Start(ctx, "mtnpay.process_payment",
		trace.WithAttributes(
			attribute.Float64("payment.amount", req.Amount),
			attribute.String("payment.currency", req.Currency),
			attribute.String("payment.reference", req.Reference),
		),
	)
	defer span.End()

	var response MTNPayResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		SetError(&errorResp).
		Post("/payments")

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("MTN Pay API request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "POST"),
		attribute.String("http.url", "/payments"),
	)

	if resp.IsError() {
		err := fmt.Errorf("MTN Pay payment failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("mtnpay.transaction_id", response.TransactionID),
		attribute.String("mtnpay.status", response.Status),
	)

	return &response, nil
}

func (c *MTNPayClient) GetPaymentStatus(ctx context.Context, transactionID string) (*MTNPayStatusResponse, error) {
	ctx, span := c.tracer.Start(ctx, "mtnpay.get_payment_status",
		trace.WithAttributes(
			attribute.String("mtnpay.transaction_id", transactionID),
		),
	)
	defer span.End()

	var response MTNPayStatusResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetError(&errorResp).
		Get(fmt.Sprintf("/payments/%s", transactionID))

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("MTN Pay status request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "GET"),
	)

	if resp.IsError() {
		err := fmt.Errorf("MTN Pay status check failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("mtnpay.status", response.Status),
	)

	return &response, nil
}

func (c *MTNPayClient) GetBalance(ctx context.Context, phoneNumber string) (*BalanceResponse, error) {
	ctx, span := c.tracer.Start(ctx, "mtnpay.get_balance",
		trace.WithAttributes(
			attribute.String("mtnpay.phone_number", phoneNumber),
		),
	)
	defer span.End()

	var response BalanceResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetError(&errorResp).
		Get(fmt.Sprintf("/balance/%s", phoneNumber))

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("MTN Pay balance request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "GET"),
	)

	if resp.IsError() {
		err := fmt.Errorf("MTN Pay balance check failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Float64("mtnpay.balance", response.Balance),
		attribute.String("mtnpay.currency", response.Currency),
	)

	return &response, nil
}

type BalanceResponse struct {
	PhoneNumber string  `json:"phone_number"`
	Balance     float64 `json:"balance"`
	Currency    string  `json:"currency"`
	Status      string  `json:"status"`
}

// Simulate network errors and timeouts for demo purposes
func (c *MTNPayClient) SimulateError(ctx context.Context, errorType string) error {
	ctx, span := c.tracer.Start(ctx, "mtnpay.simulate_error",
		trace.WithAttributes(
			attribute.String("error.type", errorType),
		),
	)
	defer span.End()

	switch errorType {
	case "timeout":
		time.Sleep(35 * time.Second) // Longer than client timeout
		return fmt.Errorf("request timeout")
	case "network":
		err := fmt.Errorf("network connection failed")
		span.RecordError(err)
		return err
	case "server_error":
		err := fmt.Errorf("MTN Pay server error: 500 Internal Server Error")
		span.RecordError(err)
		return err
	case "rate_limit":
		err := fmt.Errorf("MTN Pay rate limit exceeded: 429 Too Many Requests")
		span.RecordError(err)
		return err
	default:
		return nil
	}
}
