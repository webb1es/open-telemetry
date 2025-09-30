package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Order struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     primitive.ObjectID `bson:"user_id" json:"user_id"`
	Items      []OrderItem        `bson:"items" json:"items"`
	Status     OrderStatus        `bson:"status" json:"status"`
	Total      float64            `bson:"total" json:"total"`
	Currency   string             `bson:"currency" json:"currency"`
	PaymentID  primitive.ObjectID `bson:"payment_id,omitempty" json:"payment_id,omitempty"`
	ShippingID string             `bson:"shipping_id,omitempty" json:"shipping_id,omitempty"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}

type OrderItem struct {
	ProductID string  `bson:"product_id" json:"product_id"`
	Name      string  `bson:"name" json:"name"`
	Quantity  int     `bson:"quantity" json:"quantity"`
	Price     float64 `bson:"price" json:"price"`
	Total     float64 `bson:"total" json:"total"`
}

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusRefunded   OrderStatus = "refunded"
)

type CreateOrderRequest struct {
	UserID   string             `json:"user_id" validate:"required"`
	Items    []OrderItemRequest `json:"items" validate:"required,dive"`
	Currency string             `json:"currency" validate:"required"`
}

type OrderItemRequest struct {
	ProductID string  `json:"product_id" validate:"required"`
	Quantity  int     `json:"quantity" validate:"required,gt=0"`
	Price     float64 `json:"price" validate:"required,gt=0"`
}

type OrderResponse struct {
	ID         string      `json:"id"`
	UserID     string      `json:"user_id"`
	Items      []OrderItem `json:"items"`
	Status     OrderStatus `json:"status"`
	Total      float64     `json:"total"`
	Currency   string      `json:"currency"`
	PaymentID  string      `json:"payment_id,omitempty"`
	ShippingID string      `json:"shipping_id,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}
