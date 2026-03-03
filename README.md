# Product Catalog Service

A product catalog microservice built with Go, gRPC, and Google Cloud Spanner, following Domain-Driven Design and Clean Architecture principles.

## Architecture

The service follows a layered architecture:

- **Domain Layer** (`internal/app/product/domain/`) — Pure business logic with no infrastructure dependencies. Contains the Product aggregate, Money/Discount value objects, domain events, and domain errors.
- **Application Layer** (`internal/app/product/usecases/`, `queries/`) — Use cases (commands) and query handlers. Commands follow the Golden Mutation Pattern: Load → Domain → Build Plan → Apply.
- **Infrastructure Layer** (`internal/app/product/repo/`, `internal/transport/`) — Spanner repository, gRPC transport, and database models.

### Key Patterns

- **Golden Mutation Pattern** — Repositories return mutations (never apply them). Use cases build a plan of mutations and apply it atomically via CommitPlan.
- **Transactional Outbox** — Domain events are stored in an outbox table within the same atomic transaction as aggregate changes, ensuring reliable event delivery.
- **CQRS** — Commands go through the domain aggregate; queries bypass it for optimized reads.
- **Change Tracking** — The ChangeTracker records which fields were modified, enabling targeted (partial) updates.

## Prerequisites

- Go 1.26+
- Docker (for Spanner emulator)
- `protoc` with `protoc-gen-go` and `protoc-gen-go-grpc` (optional, for proto regeneration)

## Configuration

Copy `.env.dist` to `.env` and adjust as needed. All settings can also be passed as CLI flags.

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `GRPC_ADDR` | `--grpc-addr` | `:50051` | gRPC listen address |
| `HEALTH_ADDR` | `--health-addr` | `:50052` | Health check gRPC listen address |
| `SPANNER_DATABASE` | `--spanner-database` | — | Spanner database path (required) |
| `SPANNER_EMULATOR_HOST` | — | — | Spanner emulator host (set for local dev) |
| `OTEL_SERVICE_NAME` | `--otel-service-name` | `product-catalog-service` | OpenTelemetry service name |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `--otel-exporter-otlp-endpoint` | — | OTLP exporter endpoint |
| `METRICS_ADDR` | `--metrics-addr` | `:9090` | Prometheus metrics listen address |

## Quick Start

```bash
# Start Spanner emulator
docker-compose up -d

# Set emulator environment
export SPANNER_EMULATOR_HOST=localhost:9010

# Run migrations
make migrate

# Run the gRPC server
make run

# Run E2E tests
make test-e2e
```

## Project Structure

```
cmd/server/             — Service entry point
internal/
  app/product/
    domain/             — Pure domain: Product, Money, Discount, events, errors
    usecases/           — Command handlers (create, update, activate, deactivate, discount)
    queries/            — Query handlers (get, list)
    contracts/          — Repository and read-model interfaces
    repo/               — Spanner implementation
  models/               — Database model structs and field constants
  transport/grpc/       — gRPC handlers and proto mappings
  services/             — DI container
  pkg/                  — Clock abstraction, typed CommitPlan wrapper
proto/                  — Protobuf definitions
migrations/             — Spanner DDL
tests/e2e/              — End-to-end tests
```

## Design Decisions

1. **`*big.Rat` for money** — Avoids floating-point precision issues. Stored as numerator/denominator in Spanner.
2. **CommitPlan library** — Decouples transaction management from business logic. The Spanner driver wraps `client.Apply()`.
3. **No context in domain** — The domain layer is pure Go with zero infrastructure imports, making it trivially testable.
4. **Outbox events as intents** — Domain events are simple structs that capture "what happened." The repository serializes them into the outbox table.
5. **Single repo as read model** — For simplicity, the repository also implements the read-model interface. In production, these could be split for scaling.

## API

The gRPC API exposes:

| RPC | Description |
|-----|-------------|
| `CreateProduct` | Create a new product (draft status) |
| `UpdateProduct` | Update product details |
| `ActivateProduct` | Transition product to active |
| `DeactivateProduct` | Transition product to inactive |
| `ApplyDiscount` | Apply a percentage discount with validity period |
| `RemoveDiscount` | Remove the current discount |
| `GetProduct` | Get product by ID with effective price |
| `ListProducts` | List products with pagination and category filter |
