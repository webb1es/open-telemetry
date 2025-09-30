package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Payment struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID `bson:"user_id" json:"user_id"`
	OrderID       primitive.ObjectID `bson:"order_id,omitempty" json:"order_id,omitempty"`
	Amount        float64            `bson:"amount" json:"amount"`
	Currency      string             `bson:"currency" json:"currency"`
	Method        PaymentMethod      `bson:"method" json:"method"`
	Status        PaymentStatus      `bson:"status" json:"status"`
	ExternalTxnID string             `bson:"external_txn_id,omitempty" json:"external_txn_id,omitempty"`
	Reference     string             `bson:"reference" json:"reference"`
	Description   string             `bson:"description,omitempty" json:"description,omitempty"`
	Metadata      map[string]string  `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

type PaymentMethod string

const (
	PaymentMethodMTNPay PaymentMethod = "mtn_pay"
	PaymentMethodCard   PaymentMethod = "card"
	PaymentMethodWallet PaymentMethod = "wallet"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusCompleted  PaymentStatus = "completed"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCancelled  PaymentStatus = "cancelled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

type CreatePaymentRequest struct {
	UserID      string            `json:"user_id" validate:"required"`
	OrderID     string            `json:"order_id,omitempty"`
	Amount      float64           `json:"amount" validate:"required,gt=0"`
	Currency    string            `json:"currency" validate:"required"`
	Method      PaymentMethod     `json:"method" validate:"required"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type PaymentResponse struct {
	ID            string            `json:"id"`
	UserID        string            `json:"user_id"`
	OrderID       string            `json:"order_id,omitempty"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	Method        PaymentMethod     `json:"method"`
	Status        PaymentStatus     `json:"status"`
	ExternalTxnID string            `json:"external_txn_id,omitempty"`
	Reference     string            `json:"reference"`
	Description   string            `json:"description,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}
