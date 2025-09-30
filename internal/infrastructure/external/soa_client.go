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

type SOAClient struct {
	client *resty.Client
	config *config.SOAConfig
	tracer trace.Tracer
}

func NewSOAClient(cfg *config.SOAConfig) *SOAClient {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetHeader("Content-Type", "application/json").
		SetHeader("X-API-Key", cfg.APIKey).
		SetTimeout(25 * time.Second)

	return &SOAClient{
		client: client,
		config: cfg,
		tracer: otel.Tracer("soa-client"),
	}
}

type InventoryRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Location  string `json:"location,omitempty"`
}

type InventoryResponse struct {
	ProductID     string     `json:"product_id"`
	Available     bool       `json:"available"`
	StockLevel    int        `json:"stock_level"`
	ReservedStock int        `json:"reserved_stock"`
	Location      string     `json:"location"`
	NextRestock   *time.Time `json:"next_restock,omitempty"`
}

type ShippingRequest struct {
	OrderID    string         `json:"order_id"`
	UserID     string         `json:"user_id"`
	Items      []ShippingItem `json:"items"`
	Address    Address        `json:"address"`
	Weight     float64        `json:"weight,omitempty"`
	Dimensions Dimensions     `json:"dimensions,omitempty"`
}

type ShippingItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Weight    float64 `json:"weight"`
}

type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type Dimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type ShippingResponse struct {
	OrderID        string    `json:"order_id"`
	ShippingID     string    `json:"shipping_id"`
	TrackingNumber string    `json:"tracking_number"`
	Carrier        string    `json:"carrier"`
	Service        string    `json:"service"`
	Cost           float64   `json:"cost"`
	Currency       string    `json:"currency"`
	EstimatedDays  int       `json:"estimated_days"`
	CreatedAt      time.Time `json:"created_at"`
}

type ProductCatalogRequest struct {
	Category string   `json:"category,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	MinPrice float64  `json:"min_price,omitempty"`
	MaxPrice float64  `json:"max_price,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Offset   int      `json:"offset,omitempty"`
}

type ProductCatalogResponse struct {
	Products []Product `json:"products"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PerPage  int       `json:"per_page"`
	HasMore  bool      `json:"has_more"`
}

type Product struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	SKU         string            `json:"sku"`
	Description string            `json:"description"`
	Price       float64           `json:"price"`
	Currency    string            `json:"currency"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`
	Images      []string          `json:"images"`
	Attributes  map[string]string `json:"attributes"`
	InStock     bool              `json:"in_stock"`
	StockLevel  int               `json:"stock_level"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func (c *SOAClient) CheckInventory(ctx context.Context, req InventoryRequest) (*InventoryResponse, error) {
	ctx, span := c.tracer.Start(ctx, "soa.check_inventory",
		trace.WithAttributes(
			attribute.String("product.id", req.ProductID),
			attribute.Int("product.quantity", req.Quantity),
		),
	)
	defer span.End()

	var response InventoryResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		SetError(&errorResp).
		Post("/inventory/check")

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("SOA inventory check request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "POST"),
		attribute.String("http.url", "/inventory/check"),
	)

	if resp.IsError() {
		err := fmt.Errorf("SOA inventory check failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Bool("soa.product_available", response.Available),
		attribute.Int("soa.stock_level", response.StockLevel),
		attribute.String("soa.location", response.Location),
	)

	return &response, nil
}

func (c *SOAClient) CreateShipping(ctx context.Context, req ShippingRequest) (*ShippingResponse, error) {
	ctx, span := c.tracer.Start(ctx, "soa.create_shipping",
		trace.WithAttributes(
			attribute.String("order.id", req.OrderID),
			attribute.String("user.id", req.UserID),
			attribute.Int("shipping.items_count", len(req.Items)),
		),
	)
	defer span.End()

	var response ShippingResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		SetError(&errorResp).
		Post("/shipping")

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("SOA shipping request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "POST"),
		attribute.String("http.url", "/shipping"),
	)

	if resp.IsError() {
		err := fmt.Errorf("SOA shipping failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("soa.shipping_id", response.ShippingID),
		attribute.String("soa.tracking_number", response.TrackingNumber),
		attribute.String("soa.carrier", response.Carrier),
		attribute.Float64("soa.shipping_cost", response.Cost),
		attribute.Int("soa.estimated_days", response.EstimatedDays),
	)

	return &response, nil
}

func (c *SOAClient) GetProductCatalog(ctx context.Context, req ProductCatalogRequest) (*ProductCatalogResponse, error) {
	ctx, span := c.tracer.Start(ctx, "soa.get_product_catalog",
		trace.WithAttributes(
			attribute.String("catalog.category", req.Category),
			attribute.Int("catalog.limit", req.Limit),
			attribute.Int("catalog.offset", req.Offset),
		),
	)
	defer span.End()

	var response ProductCatalogResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&response).
		SetError(&errorResp).
		Post("/catalog/products")

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("SOA catalog request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "POST"),
		attribute.String("http.url", "/catalog/products"),
	)

	if resp.IsError() {
		err := fmt.Errorf("SOA catalog failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("soa.products_count", len(response.Products)),
		attribute.Int("soa.total_products", response.Total),
		attribute.Bool("soa.has_more", response.HasMore),
	)

	return &response, nil
}

func (c *SOAClient) GetShippingStatus(ctx context.Context, shippingID string) (*ShippingStatusResponse, error) {
	ctx, span := c.tracer.Start(ctx, "soa.get_shipping_status",
		trace.WithAttributes(
			attribute.String("soa.shipping_id", shippingID),
		),
	)
	defer span.End()

	var response ShippingStatusResponse
	var errorResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}

	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetError(&errorResp).
		Get(fmt.Sprintf("/shipping/%s/status", shippingID))

	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("SOA shipping status request failed: %w", err)
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode()),
		attribute.String("http.method", "GET"),
	)

	if resp.IsError() {
		err := fmt.Errorf("SOA shipping status failed: %s - %s", errorResp.Error, errorResp.Message)
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("soa.shipping_status", response.Status),
		attribute.String("soa.tracking_number", response.TrackingNumber),
	)

	return &response, nil
}

type ShippingStatusResponse struct {
	ShippingID        string          `json:"shipping_id"`
	OrderID           string          `json:"order_id"`
	Status            string          `json:"status"`
	TrackingNumber    string          `json:"tracking_number"`
	Carrier           string          `json:"carrier"`
	LastUpdate        time.Time       `json:"last_update"`
	EstimatedDelivery *time.Time      `json:"estimated_delivery,omitempty"`
	DeliveredAt       *time.Time      `json:"delivered_at,omitempty"`
	Events            []TrackingEvent `json:"events,omitempty"`
}

type TrackingEvent struct {
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Location    string    `json:"location,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// Simulate service degradation for demo purposes
func (c *SOAClient) SimulateDegradation(ctx context.Context, delayMs int) error {
	ctx, span := c.tracer.Start(ctx, "soa.simulate_degradation",
		trace.WithAttributes(
			attribute.Int("degradation.delay_ms", delayMs),
		),
	)
	defer span.End()

	time.Sleep(time.Duration(delayMs) * time.Millisecond)

	span.SetAttributes(
		attribute.String("simulation.type", "service_degradation"),
	)

	return nil
}
