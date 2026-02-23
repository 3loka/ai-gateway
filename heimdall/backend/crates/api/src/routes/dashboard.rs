use axum::{extract::State, routing::get, Json, Router};

use crate::state::SharedState;
use heimdall_core::models::DashboardStats;

pub fn router() -> Router<SharedState> {
    Router::new().route("/stats", get(get_stats))
}

async fn get_stats(
    State(state): State<SharedState>,
) -> Result<Json<DashboardStats>, axum::http::StatusCode> {
    let stats = sqlx::query_as!(
        DashboardStats,
        r#"
        SELECT
            (SELECT COUNT(*) FROM source_systems)                                        AS "total_sources!",
            (SELECT COUNT(*) FROM source_systems WHERE connection_mode = 'METADATA_ONLY') AS "metadata_only_sources!",
            (SELECT COUNT(*) FROM source_systems WHERE connection_mode = 'FULL_SYNC')     AS "full_sync_sources!",
            (SELECT COUNT(*) FROM data_assets)                                           AS "total_tables!",
            (SELECT COUNT(*) FROM column_metadata)                                       AS "total_columns!",
            (SELECT COUNT(*) FROM column_metadata WHERE is_pii = TRUE)                   AS "pii_columns!",
            (SELECT COUNT(*) FROM schema_change_events
             WHERE severity = 'CRITICAL' AND detected_at > NOW() - INTERVAL '24 hours') AS "critical_changes_today!",
            (SELECT COUNT(*) FROM schema_change_events
             WHERE blocked_extraction = TRUE AND resolved = FALSE)                       AS "blocked_extractions!"
        "#
    )
    .fetch_one(&state.db)
    .await
    .map_err(|e| {
        tracing::error!("Dashboard stats error: {e}");
        axum::http::StatusCode::INTERNAL_SERVER_ERROR
    })?;

    Ok(Json(stats))
}
