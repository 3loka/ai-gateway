# ─── Builder ──────────────────────────────────────────────────────────────────
FROM rust:1.82-slim AS builder

RUN apt-get update && apt-get install -y pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .

# Build both binaries in release mode
RUN cargo build --release -p heimdall-api -p heimdall-worker

# ─── API image ─────────────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS api

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/heimdall-api /usr/local/bin/heimdall-api
EXPOSE 8080
CMD ["heimdall-api"]

# ─── Worker image ──────────────────────────────────────────────────────────────
FROM debian:bookworm-slim AS worker

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/heimdall-worker /usr/local/bin/heimdall-worker
CMD ["heimdall-worker"]
