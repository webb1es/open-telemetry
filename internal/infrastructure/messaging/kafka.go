package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/webbies/otel-fiber-demo/internal/infrastructure/config"
)

type KafkaManager struct {
	config *config.KafkaConfig
	tracer trace.Tracer
}

func NewKafkaManager(cfg *config.KafkaConfig) *KafkaManager {
	return &KafkaManager{
		config: cfg,
		tracer: otel.Tracer("kafka-client"),
	}
}

// Publisher for sending messages
type Publisher struct {
	writer *kafka.Writer
	tracer trace.Tracer
}

func (km *KafkaManager) NewPublisher(topic string) *Publisher {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(km.config.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	return &Publisher{
		writer: writer,
		tracer: km.tracer,
	}
}

func (p *Publisher) PublishMessage(ctx context.Context, key string, value interface{}) error {
	ctx, span := p.tracer.Start(ctx, "kafka.publish",
		trace.WithAttributes(
			attribute.String("kafka.topic", p.writer.Topic),
			attribute.String("kafka.key", key),
		),
	)
	defer span.End()

	// Serialize value to JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Create message with tracing headers
	headers := make([]kafka.Header, 0)

	// Inject trace context into headers
	carrier := &headerCarrier{headers: &headers}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	msg := kafka.Message{
		Key:     []byte(key),
		Value:   valueBytes,
		Headers: headers,
		Time:    time.Now(),
	}

	// Send message
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	span.SetAttributes(
		attribute.Int("kafka.message_size", len(valueBytes)),
		attribute.String("kafka.operation", "publish"),
	)

	return nil
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}

// Consumer for receiving messages
type Consumer struct {
	reader *kafka.Reader
	tracer trace.Tracer
}

func (km *KafkaManager) NewConsumer(topic, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        km.config.Brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})

	return &Consumer{
		reader: reader,
		tracer: km.tracer,
	}
}

type MessageHandler func(ctx context.Context, key string, value []byte) error

func (c *Consumer) StartConsuming(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				return fmt.Errorf("failed to read message: %w", err)
			}

			// Extract trace context from headers
			carrier := &headerCarrier{headers: &msg.Headers}
			msgCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)

			// Start span for message processing
			msgCtx, span := c.tracer.Start(msgCtx, "kafka.consume",
				trace.WithAttributes(
					attribute.String("kafka.topic", msg.Topic),
					attribute.Int("kafka.partition", msg.Partition),
					attribute.Int64("kafka.offset", msg.Offset),
					attribute.String("kafka.key", string(msg.Key)),
					attribute.Int("kafka.message_size", len(msg.Value)),
				),
			)

			// Process message
			if err := handler(msgCtx, string(msg.Key), msg.Value); err != nil {
				span.RecordError(err)
				// In production, you might want to send to a dead letter queue
			}

			span.End()
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

// Event structures for different message types
type UserCreatedEvent struct {
	UserID    string            `json:"user_id"`
	Email     string            `json:"email"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

type PaymentProcessedEvent struct {
	PaymentID     string            `json:"payment_id"`
	UserID        string            `json:"user_id"`
	OrderID       string            `json:"order_id,omitempty"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	Status        string            `json:"status"`
	ExternalTxnID string            `json:"external_txn_id,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
}

type OrderCreatedEvent struct {
	OrderID   string            `json:"order_id"`
	UserID    string            `json:"user_id"`
	Total     float64           `json:"total"`
	Currency  string            `json:"currency"`
	Status    string            `json:"status"`
	ItemCount int               `json:"item_count"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

type RewardProcessedEvent struct {
	RewardID  string            `json:"reward_id"`
	UserID    string            `json:"user_id"`
	Type      string            `json:"type"`
	Points    int64             `json:"points"`
	Value     float64           `json:"value,omitempty"`
	Currency  string            `json:"currency,omitempty"`
	Source    string            `json:"source"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Header carrier for trace context propagation
type headerCarrier struct {
	headers *[]kafka.Header
}

func (hc *headerCarrier) Get(key string) string {
	for _, header := range *hc.headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}

func (hc *headerCarrier) Set(key, value string) {
	// Remove existing header with the same key
	for i, header := range *hc.headers {
		if header.Key == key {
			(*hc.headers)[i] = kafka.Header{Key: key, Value: []byte(value)}
			return
		}
	}
	// Add new header
	*hc.headers = append(*hc.headers, kafka.Header{Key: key, Value: []byte(value)})
}

func (hc *headerCarrier) Keys() []string {
	keys := make([]string, len(*hc.headers))
	for i, header := range *hc.headers {
		keys[i] = header.Key
	}
	return keys
}

// Publisher helpers for specific event types
func (km *KafkaManager) PublishUserCreated(ctx context.Context, event UserCreatedEvent) error {
	publisher := km.NewPublisher(km.config.Topics.Users)
	defer publisher.Close()

	return publisher.PublishMessage(ctx, event.UserID, event)
}

func (km *KafkaManager) PublishPaymentProcessed(ctx context.Context, event PaymentProcessedEvent) error {
	publisher := km.NewPublisher(km.config.Topics.Payments)
	defer publisher.Close()

	return publisher.PublishMessage(ctx, event.PaymentID, event)
}

func (km *KafkaManager) PublishOrderCreated(ctx context.Context, event OrderCreatedEvent) error {
	publisher := km.NewPublisher(km.config.Topics.Orders)
	defer publisher.Close()

	return publisher.PublishMessage(ctx, event.OrderID, event)
}

func (km *KafkaManager) PublishRewardProcessed(ctx context.Context, event RewardProcessedEvent) error {
	publisher := km.NewPublisher(km.config.Topics.Rewards)
	defer publisher.Close()

	return publisher.PublishMessage(ctx, event.RewardID, event)
}
