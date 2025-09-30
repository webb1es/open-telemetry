# OpenTelemetry Go Fiber Demo

A comprehensive demonstration of OpenTelemetry capabilities in a Go application using the Fiber framework with clean architecture principles.

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Make (optional)

### Running with Docker

1. Clone the repository
2. Copy environment variables:
   ```bash
   cp deployments/.env.example deployments/.env
   ```
3. Start the entire stack:
   ```bash
   docker-compose -f deployments/docker-compose.yml up -d
   ```
4. Wait for all services to start (may take 2-3 minutes)

### Running Locally

1. Ensure MongoDB, Redis, and Kafka are running
2. Copy and configure environment variables:
   ```bash
   cp deployments/.env.example .env
   # Edit .env with your local configuration
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Run the application:
   ```bash
   go run cmd/api/main.go
   ```

## 📊 Observability Stack

| Service | URL | Description |
|---------|-----|-------------|
| Application | http://localhost:3000 | Main API server |
| Jaeger UI | http://localhost:16686 | Distributed tracing |
| Prometheus | http://localhost:9090 | Metrics collection |
| Grafana | http://localhost:3001 | Dashboards (admin/admin123) |
| Kafka UI | http://localhost:8090 | Kafka topic management |

## 🔧 API Endpoints

### Core Business APIs
```
GET  /v1/health                     # Health check with dependency status
POST /v1/users/create               # User onboarding with validation
GET  /v1/dashboard                  # Real-time dashboard aggregation
POST /v1/payments                   # Payment processing via MTN-Pay
GET  /v1/payments/:id/status        # Payment status tracking
POST /v1/orders                     # Order creation with inventory
GET  /v1/orders/:id                 # Order details with shipping
POST /v1/rewards                    # Reward processing
GET  /v1/rewards/:userId            # User rewards summary
GET  /v1/catalogue                  # Product catalogue with pricing
GET  /v1/unifiedBalances/:userId    # Multi-source balance aggregation
POST /v1/simulate-error             # Error injection for demos
GET  /v1/metrics                    # Prometheus metrics endpoint
```

### Example Requests

#### Create User
```bash
curl -X POST http://localhost:3000/v1/users/create \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890"
  }'
```

#### Process Payment
```bash
curl -X POST http://localhost:3000/v1/payments \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_id_here",
    "amount": 100.00,
    "currency": "USD",
    "method": "mtn_pay"
  }'
```

## 🏗️ Architecture

### Clean Architecture Layers
- **Entities**: Domain models (`internal/core/entities`)
- **Use Cases**: Business logic (`internal/core/usecases`)
- **Adapters**: HTTP handlers, repositories (`internal/adapters`)
- **Infrastructure**: Database, external APIs (`internal/infrastructure`)

### OpenTelemetry Features
- ✅ Distributed tracing across all layers
- ✅ Custom business metrics
- ✅ Structured logging with trace correlation
- ✅ Automatic instrumentation for HTTP, MongoDB, Redis, Kafka
- ✅ Manual instrumentation for business flows
- ✅ Error tracking and recording
- ✅ Multiple exporters (Jaeger, Prometheus, Azure Monitor)

## 🔗 External Integrations

### Simulated External APIs
- **MTN-Pay**: Payment processing gateway
- **MADAPI**: User validation and pricing
- **SOA**: Inventory and shipping services

### Data Stores
- **MongoDB**: Primary database with automatic tracing
- **Redis**: Caching and rate limiting with instrumentation
- **Kafka**: Event streaming with distributed tracing

## 🎯 Demo Scenarios

### 1. Happy Path Flow
```bash
# Create user → Process payment → Create order → Check rewards
curl -X POST localhost:3000/v1/users/create -d '{"email":"demo@example.com",...}'
```

### 2. Error Scenarios
```bash
# Simulate various failures
curl -X POST localhost:3000/v1/simulate-error -d '{"type":"mtn_pay_timeout"}'
curl -X POST localhost:3000/v1/simulate-error -d '{"type":"database_error"}'
```

### 3. Performance Testing
```bash
# Generate load to see tracing and metrics
for i in {1..100}; do
  curl -X GET localhost:3000/v1/dashboard &
done
```

## 📈 Monitoring

### Key Metrics
- HTTP request rates and latencies
- Payment success/failure rates
- External API performance
- Database query performance
- Cache hit/miss rates
- Kafka message throughput

### Traces
- End-to-end request tracing
- Database query traces
- External API call traces
- Background job traces
- Error propagation

### Logs
- Structured JSON logs
- Trace ID correlation
- Error context
- Business event logging

## ⚙️ Configuration

### Environment Variables
```env
# Database
MONGODB_URI=mongodb://localhost:27017/otel_demo
REDIS_URL=redis://localhost:6379/0

# Kafka
KAFKA_BROKERS=localhost:9092

# External APIs
MTN_PAY_BASE_URL=https://api.mtn.com/pay/v1
MTN_PAY_API_KEY=your_key
MADAPI_BASE_URL=https://madapi.example.com/v1
SOA_BASE_URL=https://soa.example.com/api/v1

# OpenTelemetry
OTEL_SERVICE_NAME=otel-fiber-demo
OTEL_EXPORTER_JAEGER_ENDPOINT=http://localhost:14268/api/traces
AZURE_MONITOR_CONNECTION_STRING=InstrumentationKey=your-key
```

## 🧪 Development

### Running Tests
```bash
go test ./...
```

### Building
```bash
go build -o bin/app cmd/api/main.go
```

### Linting
```bash
golangci-lint run
```

## 🐛 Troubleshooting

### Common Issues

1. **Services not starting**: Check Docker resources and port conflicts
2. **No traces visible**: Verify Jaeger endpoint configuration
3. **Metrics not appearing**: Check Prometheus scraping configuration
4. **Database connection failed**: Ensure MongoDB is running and accessible

### Debug Commands
```bash
# Check service health
curl localhost:3000/v1/health

# View logs
docker-compose -f deployments/docker-compose.yml logs -f otel-fiber-demo

# Check OpenTelemetry Collector
curl localhost:13133/  # Health check
```

## 🎯 Demo Script

1. **Start Services**: `docker-compose up -d`
2. **Check Health**: Visit health endpoint
3. **Create User**: Show tracing across MongoDB + Kafka
4. **Process Payment**: Demonstrate MTN-Pay integration + metrics
5. **View Traces**: Open Jaeger UI to show end-to-end traces
6. **Check Metrics**: View Prometheus metrics
7. **Simulate Errors**: Show error tracing and recovery
8. **Dashboard**: View Grafana dashboards

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with proper tests
4. Submit a pull request

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details.