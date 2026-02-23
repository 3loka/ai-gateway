use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

// ─── Enums ────────────────────────────────────────────────────────────────────

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "connection_mode", rename_all = "SCREAMING_SNAKE_CASE")]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum ConnectionMode {
    MetadataOnly,
    FullSync,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "source_status", rename_all = "SCREAMING_SNAKE_CASE")]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum SourceStatus {
    Healthy,
    Warning,
    Error,
    Crawling,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "change_type", rename_all = "SCREAMING_SNAKE_CASE")]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum ChangeType {
    ColumnAdded,
    ColumnRemoved,
    TypeChanged,
    TableRenamed,
    VolumeAnomaly,
    NewTable,
    TableRemoved,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "change_severity", rename_all = "SCREAMING_SNAKE_CASE")]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum ChangeSeverity {
    Info,
    Warning,
    Critical,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "extraction_decision", rename_all = "SCREAMING_SNAKE_CASE")]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum ExtractionDecision {
    Approve,
    Deny,
    Partial,
    Defer,
}

// ─── Core Entities ────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct SourceSystem {
    pub id: Uuid,
    pub name: String,
    pub source_type: String,
    pub connection_mode: ConnectionMode,
    pub status: SourceStatus,
    pub table_count: i32,
    pub column_count: i32,
    pub pii_column_count: i32,
    pub last_crawled_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct DataAsset {
    pub id: Uuid,
    pub source_id: Uuid,
    pub schema_name: String,
    pub table_name: String,
    pub row_count: Option<i64>,
    pub last_modified: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct ColumnMetadata {
    pub id: Uuid,
    pub asset_id: Uuid,
    pub name: String,
    pub data_type: String,
    pub is_pii: Option<bool>,
    pub pii_type: Option<String>,
    pub pii_confidence: Option<f32>,
    pub null_pct: Option<f32>,
    pub distinct_count: Option<i64>,
    pub ai_description: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct SchemaChangeEvent {
    pub id: Uuid,
    pub asset_id: Uuid,
    pub change_type: ChangeType,
    pub severity: ChangeSeverity,
    pub description: String,
    pub blast_radius: Option<serde_json::Value>,
    pub ai_analysis: Option<String>,
    pub blocked_extraction: bool,
    pub resolved: bool,
    pub resolved_by: Option<String>,
    pub detected_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct Policy {
    pub id: Uuid,
    pub name: String,
    pub yaml_definition: String,
    pub applies_to_source: Option<String>,
    pub is_active: bool,
    pub created_by: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct AuditLog {
    pub id: Uuid,
    pub event_type: String,
    pub source_id: Option<Uuid>,
    pub asset_id: Option<Uuid>,
    pub column_id: Option<Uuid>,
    pub decision: Option<ExtractionDecision>,
    pub reason: Option<String>,
    pub policy_id: Option<Uuid>,
    pub actor: String,
    pub created_at: DateTime<Utc>,
}

// ─── Request / Response DTOs ──────────────────────────────────────────────────

#[derive(Debug, Deserialize)]
pub struct ConnectSourceRequest {
    pub name: String,
    pub source_type: String,
    pub connection_mode: ConnectionMode,
    /// Optional Fivetran connector ID — if provided, crawl real metadata
    pub fivetran_connector_id: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct DashboardStats {
    pub total_sources: i64,
    pub metadata_only_sources: i64,
    pub full_sync_sources: i64,
    pub total_tables: i64,
    pub total_columns: i64,
    pub pii_columns: i64,
    pub critical_changes_today: i64,
    pub blocked_extractions: i64,
}

#[derive(Debug, Serialize)]
pub struct BlastRadius {
    pub change_id: Uuid,
    pub affected_models: Vec<String>,
    pub affected_metrics: Vec<String>,
    pub affected_dashboards: Vec<String>,
    pub ai_analysis: String,
    pub ai_remediation: String,
}

#[derive(Debug, Serialize)]
pub struct PolicyDecision {
    pub decision: ExtractionDecision,
    pub reason: String,
    pub policy_name: Option<String>,
    pub blocked_columns: Vec<String>,
}
