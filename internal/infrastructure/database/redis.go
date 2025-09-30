package database

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/webbies/otel-fiber-demo/internal/infrastructure/config"
)

type Redis struct {
	Client *redis.Client
	tracer trace.Tracer
}

func NewRedis(cfg *config.RedisConfig) (*Redis, error) {
	// Parse Redis URL
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Create Redis client
	client := redis.NewClient(opt)

	// Create Redis wrapper with OpenTelemetry
	r := &Redis{
		Client: client,
		tracer: otel.Tracer("redis-client"),
	}

	// Add tracing hook
	client.AddHook(&tracingHook{tracer: r.tracer})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return r, nil
}

func (r *Redis) Close() error {
	return r.Client.Close()
}

func (r *Redis) IsConnected(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Cache operations with OpenTelemetry tracing
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	ctx, span := r.tracer.Start(ctx, "redis.get",
		trace.WithAttributes(
			attribute.String("redis.key", key),
		),
	)
	defer span.End()

	result, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		span.RecordError(err)
		if err == redis.Nil {
			span.SetAttributes(attribute.Bool("redis.cache_miss", true))
		}
		return "", err
	}

	span.SetAttributes(attribute.Bool("redis.cache_hit", true))
	return result, nil
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	ctx, span := r.tracer.Start(ctx, "redis.set",
		trace.WithAttributes(
			attribute.String("redis.key", key),
			attribute.String("redis.expiration", expiration.String()),
		),
	)
	defer span.End()

	err := r.Client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	ctx, span := r.tracer.Start(ctx, "redis.del",
		trace.WithAttributes(
			attribute.StringSlice("redis.keys", keys),
		),
	)
	defer span.End()

	err := r.Client.Del(ctx, keys...).Err()
	if err != nil {
		span.RecordError(err)
	}
	return err
}

func (r *Redis) Exists(ctx context.Context, keys ...string) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "redis.exists",
		trace.WithAttributes(
			attribute.StringSlice("redis.keys", keys),
		),
	)
	defer span.End()

	result, err := r.Client.Exists(ctx, keys...).Result()
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "redis.incr",
		trace.WithAttributes(
			attribute.String("redis.key", key),
		),
	)
	defer span.End()

	result, err := r.Client.Incr(ctx, key).Result()
	if err != nil {
		span.RecordError(err)
	}
	return result, err
}

func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	ctx, span := r.tracer.Start(ctx, "redis.expire",
		trace.WithAttributes(
			attribute.String("redis.key", key),
			attribute.String("redis.expiration", expiration.String()),
		),
	)
	defer span.End()

	err := r.Client.Expire(ctx, key, expiration).Err()
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Pub/Sub operations
func (r *Redis) Publish(ctx context.Context, channel string, message interface{}) error {
	ctx, span := r.tracer.Start(ctx, "redis.publish",
		trace.WithAttributes(
			attribute.String("redis.channel", channel),
		),
	)
	defer span.End()

	err := r.Client.Publish(ctx, channel, message).Err()
	if err != nil {
		span.RecordError(err)
	}
	return err
}

// Rate limiting helper
func (r *Redis) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	ctx, span := r.tracer.Start(ctx, "redis.rate_limit",
		trace.WithAttributes(
			attribute.String("redis.key", key),
			attribute.Int("rate_limit.limit", limit),
			attribute.String("rate_limit.window", window.String()),
		),
	)
	defer span.End()

	pipe := r.Client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)

	if _, err := pipe.Exec(ctx); err != nil {
		span.RecordError(err)
		return false, err
	}

	count := incr.Val()
	allowed := count <= int64(limit)

	span.SetAttributes(
		attribute.Int64("rate_limit.current", count),
		attribute.Bool("rate_limit.allowed", allowed),
	)

	return allowed, nil
}

// Tracing hook for Redis operations
type tracingHook struct {
	tracer trace.Tracer
}

func (h *tracingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		ctx, span := h.tracer.Start(ctx, "redis.dial",
			trace.WithAttributes(
				attribute.String("redis.network", network),
				attribute.String("redis.addr", addr),
			),
		)
		defer span.End()

		conn, err := next(ctx, network, addr)
		if err != nil {
			span.RecordError(err)
		}
		return conn, err
	}
}

func (h *tracingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		ctx, span := h.tracer.Start(ctx, "redis."+cmd.Name(),
			trace.WithAttributes(
				attribute.String("redis.command", cmd.Name()),
			),
		)
		defer span.End()

		err := next(ctx, cmd)
		if err != nil {
			span.RecordError(err)
		}
		return err
	}
}

func (h *tracingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		ctx, span := h.tracer.Start(ctx, "redis.pipeline",
			trace.WithAttributes(
				attribute.Int("redis.pipeline_length", len(cmds)),
			),
		)
		defer span.End()

		err := next(ctx, cmds)
		if err != nil {
			span.RecordError(err)
		}
		return err
	}
}
