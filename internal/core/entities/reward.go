package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Reward struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	Type        RewardType         `bson:"type" json:"type"`
	Points      int64              `bson:"points" json:"points"`
	Value       float64            `bson:"value,omitempty" json:"value,omitempty"`
	Currency    string             `bson:"currency,omitempty" json:"currency,omitempty"`
	Status      RewardStatus       `bson:"status" json:"status"`
	ExpiresAt   *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	RedeemedAt  *time.Time         `bson:"redeemed_at,omitempty" json:"redeemed_at,omitempty"`
	Source      RewardSource       `bson:"source" json:"source"`
	Reference   string             `bson:"reference,omitempty" json:"reference,omitempty"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type RewardType string

const (
	RewardTypePoints   RewardType = "points"
	RewardTypeCashback RewardType = "cashback"
	RewardTypeDiscount RewardType = "discount"
	RewardTypeBonus    RewardType = "bonus"
)

type RewardStatus string

const (
	RewardStatusActive   RewardStatus = "active"
	RewardStatusRedeemed RewardStatus = "redeemed"
	RewardStatusExpired  RewardStatus = "expired"
	RewardStatusRevoked  RewardStatus = "revoked"
)

type RewardSource string

const (
	RewardSourcePurchase  RewardSource = "purchase"
	RewardSourceReferral  RewardSource = "referral"
	RewardSourcePromotion RewardSource = "promotion"
	RewardSourceBonus     RewardSource = "bonus"
)

type CreateRewardRequest struct {
	UserID      string       `json:"user_id" validate:"required"`
	Type        RewardType   `json:"type" validate:"required"`
	Points      int64        `json:"points" validate:"required,gt=0"`
	Value       float64      `json:"value,omitempty"`
	Currency    string       `json:"currency,omitempty"`
	Source      RewardSource `json:"source" validate:"required"`
	Reference   string       `json:"reference,omitempty"`
	Description string       `json:"description,omitempty"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
}

type RewardResponse struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	Type        RewardType   `json:"type"`
	Points      int64        `json:"points"`
	Value       float64      `json:"value,omitempty"`
	Currency    string       `json:"currency,omitempty"`
	Status      RewardStatus `json:"status"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
	RedeemedAt  *time.Time   `json:"redeemed_at,omitempty"`
	Source      RewardSource `json:"source"`
	Reference   string       `json:"reference,omitempty"`
	Description string       `json:"description,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type UserRewardsSummary struct {
	UserID          string     `json:"user_id"`
	TotalPoints     int64      `json:"total_points"`
	AvailablePoints int64      `json:"available_points"`
	RedeemedPoints  int64      `json:"redeemed_points"`
	TotalCashback   float64    `json:"total_cashback"`
	Currency        string     `json:"currency"`
	RewardsCount    int64      `json:"rewards_count"`
	LastRewardDate  *time.Time `json:"last_reward_date,omitempty"`
}
