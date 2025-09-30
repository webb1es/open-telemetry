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

type MADAPIClient struct {
	client *resty.Client
	config *config.MADAPIConfig
	tracer trace.Tracer
}

func NewMADAPIClient(cfg *config.MADAPIConfig) *MADAPIClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+cfg.APIKey).
		SetTimeout(20 * time.Second)

	return &MADAPIClient{
		client: client,
		config: cfg,
		tracer: otel.Tracer("madapi-client"),
	}
}

type UserValidationRequest struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Document string `json:"document,omitempty"`
}

type UserValidationResponse struct {
	UserID      string            `json:"user_id"`
	IsValid     bool              `json:"is_valid"`
	Score       float64           `json:"score"`
	Reasons     []string          `json:"reasons,omitempty"`
	RiskLevel   string            `json:"risk_level"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ValidatedAt time.Time         `json:"validated_at"`
}

type PricingRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	UserID    string `json:"user_id,omitempty"`
	Region    string `json:"region,omitempty"`
}

type PricingResponse struct {
	ProductID    string    `json:"product_id"`
	BasePrice    float64   `json:"base_price"`
	FinalPrice   float64   `json:"final_price"`
	Discount     float64   `json:"discount,omitempty"`
	DiscountType string    `json:"discount_type,omitempty"`
	Currency     string    `json:"currency"`
	ValidUntil   time.Time `json:"valid_until"`
}

type RewardValidationRequest struct {
	UserID     string  `json:"user_id"`
	RewardType string  `json:"reward_type"`
	Points     int64   `json:"points"`
	Amount     float64 `json:"amount,omitempty"`
}

type RewardValidationResponse struct {
	IsValid        bool              `json:"is_valid"`
	EligibleAmount float64           `json:"eligible_amount"`
	Reason         string            `json:"reason,omitempty"`
	Limits         map[string]string `json:"limits,omitempty"`
	ValidatedAt    time.Time         `json:"validated_at"`
}

func (c *MADAPIClient) ValidateUser(ctx context.Context, req UserValidationRequest) (*UserValidationResponse, error) {
	ctx, span := c.tracer.Start(ctx, "madapi.validate_user",
		trace.WithAttributes(
			attribute.String("user.id", req.UserID),
			attribute.String("user.email", req.Email),
		),
	)
	defer span.End()

	var response UserValidationResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    int    `json:"code"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		SetError(&errorResp).
		Post("/validate/user")

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("MADAPI user validation request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "POST"),
		attribute.String("http.url", "/validate/user"),
	)

	if resp.IsError() {
		err := fmt.Errorf("MADAPI user validation failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Bool("madapi.user_valid", response.IsValid),
		attribute.Float64("madapi.validation_score", response.Score),
		attribute.String("madapi.risk_level", response.RiskLevel),
	)

	return &response, nil
}

func (c *MADAPIClient) GetPricing(ctx context.Context, req PricingRequest) (*PricingResponse, error) {
	ctx, span := c.tracer.Start(ctx, "madapi.get_pricing",
		trace.WithAttributes(
			attribute.String("product.id", req.ProductID),
			attribute.Int("product.quantity", req.Quantity),
		),
	)
	defer span.End()

	var response PricingResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    int    `json:"code"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		SetError(&errorResp).
		Post("/pricing")

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("MADAPI pricing request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "POST"),
		attribute.String("http.url", "/pricing"),
	)

	if resp.IsError() {
		err := fmt.Errorf("MADAPI pricing failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Float64("madapi.base_price", response.BasePrice),
		attribute.Float64("madapi.final_price", response.FinalPrice),
		attribute.Float64("madapi.discount", response.Discount),
		attribute.String("madapi.currency", response.Currency),
	)

	return &response, nil
}

func (c *MADAPIClient) ValidateReward(ctx context.Context, req RewardValidationRequest) (*RewardValidationResponse, error) {
	ctx, span := c.tracer.Start(ctx, "madapi.validate_reward",
		trace.WithAttributes(
			attribute.String("user.id", req.UserID),
			attribute.String("reward.type", req.RewardType),
			attribute.Int64("reward.points", req.Points),
		),
	)
	defer span.End()

	var response RewardValidationResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    int    `json:"code"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		SetError(&errorResp).
		Post("/validate/reward")

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("MADAPI reward validation request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "POST"),
		attribute.String("http.url", "/validate/reward"),
	)

	if resp.IsError() {
		err := fmt.Errorf("MADAPI reward validation failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Bool("madapi.reward_valid", response.IsValid),
		attribute.Float64("madapi.eligible_amount", response.EligibleAmount),
	)

	return &response, nil
}

func (c *MADAPIClient) GetUserProfile(ctx context.Context, userID string) (*UserProfileResponse, error) {
	ctx, span := c.tracer.Start(ctx, "madapi.get_user_profile",
		trace.WithAttributes(
			attribute.String("user.id", userID),
		),
	)
	defer span.End()

	var response UserProfileResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    int    `json:"code"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetError(&errorResp).
		Get(fmt.Sprintf("/users/%s/profile", userID))

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("MADAPI user profile request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "GET"),
	)

	if resp.IsError() {
		err := fmt.Errorf("MADAPI user profile failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("madapi.profile_tier", response.Tier),
		attribute.Bool("madapi.profile_verified", response.IsVerified),
	)

	return &response, nil
}

type UserProfileResponse struct {
	UserID       string            `json:"user_id"`
	Tier         string            `json:"tier"`
	IsVerified   bool              `json:"is_verified"`
	CreditScore  int               `json:"credit_score"`
	Preferences  map[string]string `json:"preferences,omitempty"`
	LastActivity time.Time         `json:"last_activity"`
	CreatedAt    time.Time         `json:"created_at"`
}

// Simulate rate limiting for demo purposes
func (c *MADAPIClient) SimulateRateLimit(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "madapi.simulate_rate_limit")
	defer span.End()

	err := fmt.Errorf("MADAPI rate limit exceeded: 429 Too Many Requests")
	span.RecordError(err)
	span.SetAttributes(
		attribute.Int("http.status_code", 429),
		attribute.String("error.type", "rate_limit"),
	)

	return err
}
