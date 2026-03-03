ifneq (,$(wildcard .env))
  include .env
  export
endif

.PHONY: init
init:
	cp .env.dist .env

.PHONY: install
install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: proto
proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/product/v1/product_service.proto

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test -race -v ./...

.PHONY: test-e2e
test-e2e:
	go test -race -v ./tests/e2e/...

.PHONY: run
run:
	go run ./cmd/server

.PHONY: migrate
migrate:
	@echo "Apply migrations/001_initial_schema.sql via gcloud spanner or wrench"
	@echo "Example:"
	@echo "  gcloud spanner databases ddl update test-db \\"
	@echo "    --instance=test-instance \\"
	@echo "    --ddl-file=migrations/001_initial_schema.sql"

.PHONY: vet
vet:
	go vet ./...

.PHONY: run-with-infra
run-with-infra:
	SPANNER_EMULATOR_HOST=localhost:9010 \
	go run ./cmd/server

.PHONY: lint
lint:
	golangci-lint run ./...
