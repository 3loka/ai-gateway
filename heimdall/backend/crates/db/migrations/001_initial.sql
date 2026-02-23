-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── Enums ──────────────────────────────────────────────────────────────────

CREATE TYPE connection_mode AS ENUM (
    'METADATA_ONLY',
    'FULL_SYNC'
);

CREATE TYPE source_status AS ENUM (
    'HEALTHY',
    'WARNING',
    'ERROR',
    'CRAWLING'
);

CREATE TYPE change_type AS ENUM (
    'COLUMN_ADDED',
    'COLUMN_REMOVED',
    'TYPE_CHANGED',
    'TABLE_RENAMED',
    'VOLUME_ANOMALY',
    'NEW_TABLE',
    'TABLE_REMOVED'
);

CREATE TYPE change_severity AS ENUM (
    'INFO',
    'WARNING',
    'CRITICAL'
);

CREATE TYPE extraction_decision AS ENUM (
    'APPROVE',
    'DENY',
    'PARTIAL',
    'DEFER'
);

-- ─── Core Tables ─────────────────────────────────────────────────────────────

CREATE TABLE source_systems (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    source_type         TEXT NOT NULL,
    connection_mode     connection_mode NOT NULL DEFAULT 'METADATA_ONLY',
    status              source_status NOT NULL DEFAULT 'HEALTHY',
    table_count         INT NOT NULL DEFAULT 0,
    column_count        INT NOT NULL DEFAULT 0,
    pii_column_count    INT NOT NULL DEFAULT 0,
    fivetran_connector_id TEXT,
    last_crawled_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE data_assets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id     UUID NOT NULL REFERENCES source_systems(id) ON DELETE CASCADE,
    schema_name   TEXT NOT NULL,
    table_name    TEXT NOT NULL,
    row_count     BIGINT,
    last_modified TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, schema_name, table_name)
);

CREATE TABLE column_metadata (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id        UUID NOT NULL REFERENCES data_assets(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    data_type       TEXT NOT NULL,
    is_pii          BOOLEAN,
    pii_type        TEXT,            -- EMAIL, SSN, PHONE, NAME, ADDRESS, DOB, FINANCIAL
    pii_confidence  REAL,            -- 0.0 - 1.0
    null_pct        REAL,
    distinct_count  BIGINT,
    ai_description  TEXT,
    UNIQUE (asset_id, name)
);

CREATE TABLE schema_snapshots (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id      UUID NOT NULL REFERENCES data_assets(id) ON DELETE CASCADE,
    schema_json   JSONB NOT NULL,
    captured_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE schema_change_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id            UUID NOT NULL REFERENCES data_assets(id) ON DELETE CASCADE,
    change_type         change_type NOT NULL,
    severity            change_severity NOT NULL,
    description         TEXT NOT NULL,
    blast_radius        JSONB,
    ai_analysis         TEXT,
    blocked_extraction  BOOLEAN NOT NULL DEFAULT FALSE,
    resolved            BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_by         TEXT,
    detected_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE relationships (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    src_column_id     UUID NOT NULL REFERENCES column_metadata(id) ON DELETE CASCADE,
    tgt_column_id     UUID NOT NULL REFERENCES column_metadata(id) ON DELETE CASCADE,
    rel_type          TEXT NOT NULL,  -- FOREIGN_KEY, INFERRED, SEMANTIC
    confidence_score  REAL NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL UNIQUE,
    yaml_definition     TEXT NOT NULL,
    applies_to_source   TEXT,        -- NULL = applies to all sources
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_by          TEXT NOT NULL DEFAULT 'system',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type  TEXT NOT NULL,
    source_id   UUID REFERENCES source_systems(id),
    asset_id    UUID REFERENCES data_assets(id),
    column_id   UUID REFERENCES column_metadata(id),
    decision    extraction_decision,
    reason      TEXT,
    policy_id   UUID REFERENCES policies(id),
    actor       TEXT NOT NULL DEFAULT 'system',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Indexes ─────────────────────────────────────────────────────────────────

CREATE INDEX idx_data_assets_source       ON data_assets(source_id);
CREATE INDEX idx_column_metadata_asset    ON column_metadata(asset_id);
CREATE INDEX idx_column_metadata_pii      ON column_metadata(is_pii) WHERE is_pii = TRUE;
CREATE INDEX idx_snapshots_asset          ON schema_snapshots(asset_id, captured_at DESC);
CREATE INDEX idx_changes_detected_at      ON schema_change_events(detected_at DESC);
CREATE INDEX idx_changes_severity         ON schema_change_events(severity);
CREATE INDEX idx_changes_blocked          ON schema_change_events(blocked_extraction) WHERE blocked_extraction = TRUE;
CREATE INDEX idx_audit_created_at         ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_source             ON audit_logs(source_id);
