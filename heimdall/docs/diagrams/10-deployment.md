# Deployment Diagram

```mermaid
graph TB
    subgraph Client["🖥️ Browser"]
        NEXT["Next.js SPA\nlocalhost:3000"]
    end

    subgraph Compose["🐳 Docker Compose (local) / Kubernetes (prod)"]
        subgraph RustService["Rust Service"]
            AXUM["Axum API\n:8080\nREST + SSE"]
            WORKER["Background Worker\nCrawler · PII Scanner\nEvent Generator"]
        end

        subgraph DataLayer["Data Layer"]
            PG[("PostgreSQL\n:5432\nheimdall DB")]
        end
    end

    subgraph ExtAPIs["☁️ External APIs"]
        FTV_API["Fivetran REST API\napi.fivetran.com\nschema reload · connector metadata"]
        CLAUDE_API["Claude API\napi.anthropic.com\nclaude-sonnet-4-6"]
    end

    NEXT -->|"REST /api/*\nSSE /api/changes/stream"| AXUM
    AXUM <-->|"sqlx async queries"| PG
    WORKER <-->|"sqlx async queries"| PG
    AXUM -->|"schema reload\nconnector metadata"| FTV_API
    WORKER -->|"periodic crawl\nevery 2 min"| FTV_API
    WORKER -->|"classify PII\nanalyze changes"| CLAUDE_API
```

## Environment Variables

```env
# Database
DATABASE_URL=postgres://heimdall:heimdall@localhost:5432/heimdall

# External APIs
ANTHROPIC_API_KEY=sk-ant-...
FIVETRAN_API_KEY=...
FIVETRAN_API_SECRET=...

# App config
RUST_LOG=info
CRAWL_INTERVAL_SECS=120
DEMO_MODE=true   # enables fake event generator for leadership demo
```

## docker-compose.yml

```yaml
version: "3.9"
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_USER: heimdall
      POSTGRES_PASSWORD: heimdall
      POSTGRES_DB: heimdall
    ports: ["5432:5432"]
    volumes: ["pgdata:/var/lib/postgresql/data"]

  api:
    build: ./backend
    ports: ["8080:8080"]
    environment:
      DATABASE_URL: postgres://heimdall:heimdall@db:5432/heimdall
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      FIVETRAN_API_KEY: ${FIVETRAN_API_KEY}
      DEMO_MODE: "true"
    depends_on: [db]

  frontend:
    build: ./frontend
    ports: ["3000:3000"]
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:8080

volumes:
  pgdata:
```
