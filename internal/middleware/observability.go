package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/webbies/otel-fiber-demo/internal/infrastructure/config"
	"github.com/webbies/otel-fiber-demo/internal/infrastructure/database"
	"github.com/webbies/otel-fiber-demo/internal/infrastructure/observability"
)

// RequestTracing middleware adds OpenTelemetry tracing to all requests
func RequestTracing(tracer trace.Tracer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.UserContext()

		// Start span for the request
		ctx, span := tracer.Start(ctx, c.Method()+" "+c.Route().Path,
			trace.WithAttributes(
				attribute.String("http.method", c.Method()),
				attribute.String("http.url", string(c.Request().RequestURI())),
				attribute.String("http.scheme", c.Protocol()),
				attribute.String("http.host", c.Hostname()),
				attribute.String("http.user_agent", c.Get("User-Agent")),
				attribute.String("http.route", c.Route().Path),
			),
		)
		defer span.End()

		// Set the updated context back
		c.SetUserContext(ctx)

		// Process request
		err := c.Next()

		// Set response attributes
		span.SetAttributes(
			attribute.Int("http.status_code", c.Response().StatusCode()),
			attribute.Int("http.response_size", len(c.Response().Body())),
		)

		// Record error if present
		if err != nil {
			span.RecordError(err)
		}

		return err
	}
}

// RequestMetrics middleware collects metrics for all requests
func RequestMetrics(metrics *observability.BusinessMetrics) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Record metrics
		duration := time.Since(start)

		attrs := metric.WithAttributes(
			attribute.String("method", c.Method()),
			attribute.String("route", c.Route().Path),
			attribute.String("status", strconv.Itoa(c.Response().StatusCode())),
		)

		// Increment request counter
		metrics.RequestCounter.Add(c.UserContext(), 1, attrs)

		// Record request duration
		metrics.RequestDuration.Record(c.UserContext(), duration.Seconds(), attrs)

		return err
	}
}

// RequestLogging middleware logs all requests with trace correlation
func RequestLogging(logger *observability.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Log request with trace correlation
		duration := time.Since(start)
		ctx := c.UserContext()

		logFields := []interface{}{
			"method", c.Method(),
			"path", string(c.Request().RequestURI()),
			"status", c.Response().StatusCode(),
			"duration_ms", duration.Milliseconds(),
			"ip", c.IP(),
			"user_agent", c.Get("User-Agent"),
		}

		// Add trace fields if available
		if traceFields := observability.WithTraceFields(ctx); traceFields != nil {
			for _, field := range traceFields {
				logFields = append(logFields, field.Key, field.String)
			}
		}

		if err != nil {
			logFields = append(logFields, "error", err.Error())
			logger.WithTrace(ctx).Error("HTTP request failed", logFields...)
		} else {
			logger.WithTrace(ctx).Info("HTTP request completed", logFields...)
		}

		return err
	}
}

// RateLimit middleware implements rate limiting using Redis
func RateLimit(redis *database.Redis, cfg *config.RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.UserContext()

		// Use IP address as the key for rate limiting
		key := "rate_limit:" + c.IP()

		// Check rate limit
		allowed, err := redis.CheckRateLimit(ctx, key, cfg.RequestsPerMinute, time.Minute)
		if err != nil {
			// If Redis is down, allow the request but log the error
			// In production, you might want to handle this differently
			return c.Next()
		}

		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Rate limit exceeded",
				"message": "Too many requests, please try again later",
			})
		}

		return c.Next()
	}
}

// ErrorHandler middleware handles errors with proper tracing
func ErrorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			// Add error to the current span
			ctx := c.UserContext()
			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.RecordError(err)
				span.SetAttributes(
					attribute.String("error.type", "http_error"),
					attribute.Bool("error.handled", true),
				)
			}

			// Return appropriate error response
			if e, ok := err.(*fiber.Error); ok {
				return c.Status(e.Code).JSON(fiber.Map{
					"error": e.Message,
				})
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal server error",
			})
		}
		return nil
	}
}

// HealthCheck middleware for dependency health monitoring
func HealthCheck(mongodb *database.MongoDB, redis *database.Redis) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Path() != "/v1/health" {
			return c.Next()
		}

		ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
		defer cancel()

		health := make(map[string]interface{})
		health["status"] = "healthy"
		health["timestamp"] = time.Now().UTC()

		// Check MongoDB
		if err := mongodb.IsConnected(ctx); err != nil {
			health["mongodb"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			health["status"] = "degraded"
		} else {
			health["mongodb"] = map[string]interface{}{
				"status": "healthy",
			}
		}

		// Check Redis
		if err := redis.IsConnected(ctx); err != nil {
			health["redis"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			health["status"] = "degraded"
		} else {
			health["redis"] = map[string]interface{}{
				"status": "healthy",
			}
		}

		// Set appropriate status code
		if health["status"] == "unhealthy" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(health)
		} else if health["status"] == "degraded" {
			return c.Status(fiber.StatusPartialContent).JSON(health)
		}

		return c.JSON(health)
	}
}
