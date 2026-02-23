#!/usr/bin/env bash
# Heimdall demo reset — run before each leadership demo
# Usage: ./demo-reset.sh [--api-key sk-ant-...]

set -e

PG="postgres://heimdall@localhost:5432/heimdall"
API_LOG=/tmp/heimdall-api.log
WORKER_LOG=/tmp/heimdall-worker.log
FRONTEND_LOG=/tmp/heimdall-frontend.log
BACKEND=/Users/3loka/projects/heimdall/backend
FRONTEND=/Users/3loka/projects/heimdall/frontend

# ── Optional API key override ──────────────────────────────────────────────
if [[ "$1" == "--api-key" && -n "$2" ]]; then
  sed -i '' "s|ANTHROPIC_API_KEY=.*|ANTHROPIC_API_KEY=$2|" "$BACKEND/.env"
  echo "✓ API key updated"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Heimdall Demo Reset"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ── Stop running services ──────────────────────────────────────────────────
echo "→ Stopping services..."
pkill -f "heimdall-api" 2>/dev/null || true
pkill -f "heimdall-worker" 2>/dev/null || true
sleep 1

# ── Reset database ─────────────────────────────────────────────────────────
echo "→ Resetting database..."
/opt/homebrew/opt/postgresql@16/bin/psql "$PG" -q <<SQL
  TRUNCATE schema_change_events CASCADE;
  TRUNCATE audit_logs CASCADE;
  UPDATE column_metadata SET pii_type = NULL, pii_confidence = NULL, pii_classified_at = NULL;
  UPDATE source_systems SET status = 'ACTIVE';
SQL
echo "  ✓ Change events cleared"
echo "  ✓ Audit logs cleared"
echo "  ✓ PII classifications reset"

# ── Re-seed sources ────────────────────────────────────────────────────────
echo "→ Re-seeding sources..."
cd "$BACKEND"
PATH="/Users/3loka/.cargo/bin:/opt/homebrew/opt/postgresql@16/bin:/usr/bin:$PATH" \
  cargo run --bin seed -q 2>/dev/null || \
  PATH="/Users/3loka/.cargo/bin:/opt/homebrew/opt/postgresql@16/bin:/usr/bin:$PATH" \
  cargo run --bin seed 2>&1 | tail -5
echo "  ✓ Seed data loaded"

# ── Restart API ────────────────────────────────────────────────────────────
echo "→ Starting API server..."
PATH="/Users/3loka/.cargo/bin:/opt/homebrew/opt/postgresql@16/bin:/usr/bin:$PATH" \
  nohup cargo run --bin heimdall-api > "$API_LOG" 2>&1 &
API_PID=$!
sleep 3

# Check API is up
if curl -sf http://localhost:8080/api/dashboard/stats > /dev/null 2>&1; then
  echo "  ✓ API running (PID $API_PID)"
else
  echo "  ✗ API failed to start — check $API_LOG"
  tail -20 "$API_LOG"
  exit 1
fi

# ── Restart Worker ─────────────────────────────────────────────────────────
echo "→ Starting worker..."
PATH="/Users/3loka/.cargo/bin:/opt/homebrew/opt/postgresql@16/bin:/usr/bin:$PATH" \
  nohup cargo run --bin heimdall-worker > "$WORKER_LOG" 2>&1 &
WORKER_PID=$!
sleep 1
echo "  ✓ Worker running (PID $WORKER_PID) — demo events fire every 30s"

# ── Frontend ───────────────────────────────────────────────────────────────
if ! pgrep -f "next-server" > /dev/null 2>&1; then
  echo "→ Starting frontend..."
  cd "$FRONTEND"
  nohup /opt/homebrew/bin/node node_modules/.bin/next start \
    > "$FRONTEND_LOG" 2>&1 &
  sleep 3
  echo "  ✓ Frontend running"
else
  echo "→ Frontend already running ✓"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Demo is ready!"
echo ""
echo "  Control Center  →  http://localhost:3000"
echo "  Source Catalog  →  http://localhost:3000/catalog"
echo "  Change Detection→  http://localhost:3000/changes"
echo "  Policy Engine   →  http://localhost:3000/policies"
echo "  Compliance Audit→  http://localhost:3000/audit"
echo ""
echo "  API health      →  http://localhost:8080/api/dashboard/stats"
echo "  Logs: $API_LOG"
echo "        $WORKER_LOG"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
