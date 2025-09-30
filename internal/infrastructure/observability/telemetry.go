package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/webbies/otel-fiber-demo/internal/infrastructure/config"
)

type TelemetryManager struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter
	config         *config.TelemetryConfig
}

func NewTelemetryManager(cfg *config.TelemetryConfig) (*TelemetryManager, error) {
	tm := &TelemetryManager{
		config: cfg,
	}

	if err := tm.setupResource(); err != nil {
		return nil, fmt.Errorf("failed to setup resource: %w", err)
	}

	if err := tm.setupTracing(); err != nil {
		return nil, fmt.Errorf("failed to setup tracing: %w", err)
	}

	if err := tm.setupMetrics(); err != nil {
		return nil, fmt.Errorf("failed to setup metrics: %w", err)
	}

	tm.setupPropagation()

	// Initialize tracer and meter
	tm.tracer = otel.Tracer("github.com/webbies/otel-fiber-demo")
	tm.meter = otel.Meter("github.com/webbies/otel-fiber-demo")

	return tm, nil
}

func (tm *TelemetryManager) setupResource() error {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(tm.config.ServiceName),
			semconv.ServiceVersionKey.String(tm.config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String("development"),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

func (tm *TelemetryManager) setupTracing() error {
	var exporters []sdktrace.SpanExporter

	// OTLP HTTP Exporter (modern replacement for Jaeger)
	if tm.config.JaegerEndpoint != "" {
		otlpExporter, err := otlptracehttp.New(
			context.Background(),
			otlptracehttp.WithEndpoint(tm.config.JaegerEndpoint),
			otlptracehttp.WithInsecure(), // For development
		)
		if err != nil {
			return fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
		exporters = append(exporters, otlpExporter)
	}

	if len(exporters) == 0 {
		// Fallback to console exporter for development
		consoleExporter, err := newConsoleExporter()
		if err != nil {
			return fmt.Errorf("failed to create console exporter: %w", err)
		}
		exporters = append(exporters, consoleExporter)
	}

	// Create span processors
	var processors []sdktrace.SpanProcessor
	for _, exp := range exporters {
		processors = append(processors, sdktrace.NewBatchSpanProcessor(exp))
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(tm.getResource()),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	for _, processor := range processors {
		tp.RegisterSpanProcessor(processor)
	}

	tm.tracerProvider = tp
	otel.SetTracerProvider(tp)

	return nil
}

func (tm *TelemetryManager) setupMetrics() error {
	// Prometheus exporter
	promExporter, err := prometheus.New()
	if err != nil {
		return fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	// Create meter provider
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(tm.getResource()),
		sdkmetric.WithReader(promExporter),
	)

	tm.meterProvider = mp
	otel.SetMeterProvider(mp)

	return nil
}

func (tm *TelemetryManager) setupPropagation() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

func (tm *TelemetryManager) getResource() *resource.Resource {
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(tm.config.ServiceName),
			semconv.ServiceVersionKey.String(tm.config.ServiceVersion),
		),
	)
	return res
}

func (tm *TelemetryManager) Tracer() trace.Tracer {
	return tm.tracer
}

func (tm *TelemetryManager) Meter() metric.Meter {
	return tm.meter
}

func (tm *TelemetryManager) Shutdown(ctx context.Context) error {
	var errs []error

	if tm.tracerProvider != nil {
		if err := tm.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer provider shutdown failed: %w", err))
		}
	}

	if tm.meterProvider != nil {
		if err := tm.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter provider shutdown failed: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("telemetry shutdown errors: %v", errs)
	}

	return nil
}

// Console exporter for development
type consoleExporter struct{}

func newConsoleExporter() (sdktrace.SpanExporter, error) {
	return &consoleExporter{}, nil
}

func (c *consoleExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		fmt.Printf("Span: %s [%s] Duration: %v\n",
			span.Name(),
			span.SpanKind().String(),
			span.EndTime().Sub(span.StartTime()))
	}
	return nil
}

func (c *consoleExporter) Shutdown(ctx context.Context) error {
	return nil
}

// Business Metrics
type BusinessMetrics struct {
	RequestCounter        metric.Int64Counter
	RequestDuration       metric.Float64Histogram
	PaymentSuccessCounter metric.Int64Counter
	PaymentFailureCounter metric.Int64Counter
	OrderCounter          metric.Int64Counter
	UserCreationCounter   metric.Int64Counter
	ExternalAPICounter    metric.Int64Counter
	ExternalAPIDuration   metric.Float64Histogram
}

func NewBusinessMetrics(meter metric.Meter) (*BusinessMetrics, error) {
	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	paymentSuccessCounter, err := meter.Int64Counter(
		"payments_success_total",
		metric.WithDescription("Total successful payments"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	paymentFailureCounter, err := meter.Int64Counter(
		"payments_failure_total",
		metric.WithDescription("Total failed payments"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	orderCounter, err := meter.Int64Counter(
		"orders_total",
		metric.WithDescription("Total number of orders"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	userCreationCounter, err := meter.Int64Counter(
		"users_created_total",
		metric.WithDescription("Total users created"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	externalAPICounter, err := meter.Int64Counter(
		"external_api_calls_total",
		metric.WithDescription("Total external API calls"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	externalAPIDuration, err := meter.Float64Histogram(
		"external_api_duration_seconds",
		metric.WithDescription("External API call duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &BusinessMetrics{
		RequestCounter:        requestCounter,
		RequestDuration:       requestDuration,
		PaymentSuccessCounter: paymentSuccessCounter,
		PaymentFailureCounter: paymentFailureCounter,
		OrderCounter:          orderCounter,
		UserCreationCounter:   userCreationCounter,
		ExternalAPICounter:    externalAPICounter,
		ExternalAPIDuration:   externalAPIDuration,
	}, nil
}
