package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Kafka     KafkaConfig     `mapstructure:"kafka"`
	External  ExternalConfig  `mapstructure:"external"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type ServerConfig struct {
	Port                    string        `mapstructure:"port"`
	Environment             string        `mapstructure:"environment"`
	LogLevel                string        `mapstructure:"log_level"`
	GracefulShutdownTimeout time.Duration `mapstructure:"graceful_shutdown_timeout"`
}

type DatabaseConfig struct {
	MongoURI string `mapstructure:"mongo_uri"`
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topics  Topics   `mapstructure:"topics"`
}

type Topics struct {
	Orders   string `mapstructure:"orders"`
	Payments string `mapstructure:"payments"`
	Rewards  string `mapstructure:"rewards"`
	Users    string `mapstructure:"users"`
}

type ExternalConfig struct {
	MTNPay MTNPayConfig `mapstructure:"mtn_pay"`
	MADAPI MADAPIConfig `mapstructure:"madapi"`
	SOA    SOAConfig    `mapstructure:"soa"`
}

type MTNPayConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Secret  string `mapstructure:"secret"`
}

type MADAPIConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
}

type SOAConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
}

type TelemetryConfig struct {
	ServiceName                  string `mapstructure:"service_name"`
	ServiceVersion               string `mapstructure:"service_version"`
	JaegerEndpoint               string `mapstructure:"jaeger_endpoint"`
	PrometheusPort               int    `mapstructure:"prometheus_port"`
	AzureMonitorConnectionString string `mapstructure:"azure_monitor_connection_string"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
	BurstSize         int `mapstructure:"burst_size"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set environment variable mappings
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Warning: Config file not found, using environment variables and defaults: %v\n", err)
	}

	// Bind environment variables
	bindEnvVars()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("server.port", "3000")
	viper.SetDefault("server.environment", "development")
	viper.SetDefault("server.log_level", "info")
	viper.SetDefault("server.graceful_shutdown_timeout", "30s")

	viper.SetDefault("database.mongo_uri", "mongodb://localhost:27017/otel_demo")
	viper.SetDefault("redis.url", "redis://localhost:6379/0")

	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topics.orders", "orders")
	viper.SetDefault("kafka.topics.payments", "payments")
	viper.SetDefault("kafka.topics.rewards", "rewards")
	viper.SetDefault("kafka.topics.users", "users")

	viper.SetDefault("telemetry.service_name", "otel-fiber-demo")
	viper.SetDefault("telemetry.service_version", "1.0.0")
	viper.SetDefault("telemetry.jaeger_endpoint", "http://localhost:14268/api/traces")
	viper.SetDefault("telemetry.prometheus_port", 8080)

	viper.SetDefault("rate_limit.requests_per_minute", 100)
	viper.SetDefault("rate_limit.burst_size", 10)
}

func bindEnvVars() {
	// Server
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.environment", "ENVIRONMENT")
	viper.BindEnv("server.log_level", "LOG_LEVEL")
	viper.BindEnv("server.graceful_shutdown_timeout", "GRACEFUL_SHUTDOWN_TIMEOUT")

	// Database
	viper.BindEnv("database.mongo_uri", "MONGODB_URI")
	viper.BindEnv("redis.url", "REDIS_URL")

	// Kafka
	viper.BindEnv("kafka.brokers", "KAFKA_BROKERS")
	viper.BindEnv("kafka.topics.orders", "KAFKA_TOPIC_ORDERS")
	viper.BindEnv("kafka.topics.payments", "KAFKA_TOPIC_PAYMENTS")
	viper.BindEnv("kafka.topics.rewards", "KAFKA_TOPIC_REWARDS")
	viper.BindEnv("kafka.topics.users", "KAFKA_TOPIC_USERS")

	// External APIs
	viper.BindEnv("external.mtn_pay.base_url", "MTN_PAY_BASE_URL")
	viper.BindEnv("external.mtn_pay.api_key", "MTN_PAY_API_KEY")
	viper.BindEnv("external.mtn_pay.secret", "MTN_PAY_SECRET")
	viper.BindEnv("external.madapi.base_url", "MADAPI_BASE_URL")
	viper.BindEnv("external.madapi.api_key", "MADAPI_API_KEY")
	viper.BindEnv("external.soa.base_url", "SOA_BASE_URL")
	viper.BindEnv("external.soa.api_key", "SOA_API_KEY")

	// Telemetry
	viper.BindEnv("telemetry.service_name", "OTEL_SERVICE_NAME")
	viper.BindEnv("telemetry.service_version", "OTEL_SERVICE_VERSION")
	viper.BindEnv("telemetry.jaeger_endpoint", "OTEL_EXPORTER_JAEGER_ENDPOINT")
	viper.BindEnv("telemetry.prometheus_port", "OTEL_EXPORTER_PROMETHEUS_PORT")
	viper.BindEnv("telemetry.azure_monitor_connection_string", "AZURE_MONITOR_CONNECTION_STRING")

	// Rate Limiting
	viper.BindEnv("rate_limit.requests_per_minute", "RATE_LIMIT_REQUESTS_PER_MINUTE")
	viper.BindEnv("rate_limit.burst_size", "RATE_LIMIT_BURST_SIZE")
}
