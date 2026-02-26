.PHONY: build run dev tidy

build:
	go build -o bin/ai-gateway ./cmd

run: build
	./bin/ai-gateway

dev:
	go run ./cmd

tidy:
	go mod tidy

# Quick smoke test — requires the gateway to be running.
test-generate:
	curl -s -X POST http://localhost:8080/v1/generate \
	  -H "Content-Type: application/json" \
	  -d '{"prompt":"Say hello in one sentence.","profile":"local"}' | jq .

health:
	curl -s http://localhost:8080/v1/health | jq .

profiles:
	curl -s http://localhost:8080/v1/profiles | jq .
