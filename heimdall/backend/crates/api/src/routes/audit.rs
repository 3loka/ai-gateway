use axum::{
    extract::{Query, State},
    http::StatusCode,
    routing::get,
    Json, Router,
};
use serde::Deserialize;
use uuid::Uuid;

use crate::state::SharedState;
use heimdall_core::models::AuditLog;

pub fn router() -> Router<SharedState> {
    Router::new()
        .route("/",          get(list_audit_logs))
        .route("/pii-report", get(pii_report))
}

#[derive(Deserialize)]
struct AuditQuery {
    source_id: Option<Uuid>,
    event_type: Option<String>,
    actor: Option<String>,
    limit: Option<i64>,
}

// GET /api/audit
async fn list_audit_logs(
    State(state): State<SharedState>,
    Query(q): Query<AuditQuery>,
) -> Result<Json<Vec<AuditLog>>, StatusCode> {
    let limit = q.limit.unwrap_or(100).min(500);

    let logs = sqlx::query_as!(
        AuditLog,
        r#"SELECT id, event_type, source_id, asset_id, column_id,
                  decision AS "decision: _", reason, policy_id, actor, created_at
           FROM audit_logs
           WHERE ($1::uuid IS NULL OR source_id = $1)
             AND ($2::text IS NULL OR event_type = $2)
             AND ($3::text IS NULL OR actor = $3)
           ORDER BY created_at DESC
           LIMIT $4"#,
        q.source_id,
        q.event_type,
        q.actor,
        limit,
    )
    .fetch_all(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    Ok(Json(logs))
}

// GET /api/audit/pii-report
// Returns the SOC2-ready PII classification report — the "15 minutes vs 7 days" demo moment.
async fn pii_report(
    State(state): State<SharedState>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    // PII columns with their source and asset context
    let rows = sqlx::query!(
        r#"SELECT
               ss.name  AS source_name,
               ss.source_type,
               da.schema_name,
               da.table_name,
               cm.name  AS column_name,
               cm.data_type,
               cm.pii_type,
               cm.pii_confidence,
               ss.connection_mode::text AS connection_mode
           FROM column_metadata cm
           JOIN data_assets da     ON cm.asset_id  = da.id
           JOIN source_systems ss  ON da.source_id = ss.id
           WHERE cm.is_pii = TRUE
           ORDER BY ss.name, da.table_name, cm.name"#
    )
    .fetch_all(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    // Summary counts
    let summary = sqlx::query!(
        r#"SELECT
               COUNT(*) FILTER (WHERE cm.is_pii = TRUE)                              AS total_pii,
               COUNT(*) FILTER (WHERE cm.is_pii = TRUE AND ss.connection_mode = 'METADATA_ONLY') AS pii_metadata_only,
               COUNT(*) FILTER (WHERE cm.is_pii = TRUE AND ss.connection_mode = 'FULL_SYNC')     AS pii_extracted,
               COUNT(DISTINCT ss.id) FILTER (WHERE cm.is_pii = TRUE)                AS sources_with_pii
           FROM column_metadata cm
           JOIN data_assets da    ON cm.asset_id  = da.id
           JOIN source_systems ss ON da.source_id = ss.id"#
    )
    .fetch_one(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    let columns: Vec<serde_json::Value> = rows
        .iter()
        .map(|r| {
            serde_json::json!({
                "source":       r.source_name,
                "source_type":  r.source_type,
                "schema":       r.schema_name,
                "table":        r.table_name,
                "column":       r.column_name,
                "data_type":    r.data_type,
                "pii_type":     r.pii_type,
                "confidence":   r.pii_confidence,
                "mode":         r.connection_mode,
            })
        })
        .collect();

    Ok(Json(serde_json::json!({
        "generated_at": chrono::Utc::now(),
        "summary": {
            "total_pii_columns":    summary.total_pii,
            "pii_metadata_only":    summary.pii_metadata_only,
            "pii_being_extracted":  summary.pii_extracted,
            "sources_with_pii":     summary.sources_with_pii,
        },
        "columns": columns
    })))
}
