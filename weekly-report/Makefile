.PHONY: build run dry-run clean test

GATEWAY_URL ?= http://localhost:8090

# Build the binary
build:
	go build -o bin/dbt-weekly-report ./cmd/

# Run with default config — uses URL from config.yaml (direct Ollama by default)
run: build
	./bin/dbt-weekly-report

# Fetch data only (no LLM needed)
dry-run: build
	./bin/dbt-weekly-report --dry-run

# Local Ollama directly (no Docker NAT — avoids EOF on long inference)
run-local: build
	./bin/dbt-weekly-report --model llama3.1 --url http://localhost:11434

# Cloud models via ai-gateway-platform (fast, no local GPU needed)
run-fast: build
	./bin/dbt-weekly-report --model claude-haiku-4-5-20251001 --url $(GATEWAY_URL)

run-accurate: build
	./bin/dbt-weekly-report --model claude-sonnet-4-20250514 --url $(GATEWAY_URL)

# Time range shortcuts
run-weekly: build
	./bin/dbt-weekly-report --range weekly

run-biweekly: build
	./bin/dbt-weekly-report --range biweekly

run-monthly: build
	./bin/dbt-weekly-report --range monthly

run-quarterly: build
	./bin/dbt-weekly-report --range quarterly

# Clean build artifacts
clean:
	rm -rf bin/ reports/

# Download dependencies
deps:
	go mod tidy
	go mod download

# Run tests
test:
	go test ./...

# Lint
lint:
	golangci-lint run ./...
