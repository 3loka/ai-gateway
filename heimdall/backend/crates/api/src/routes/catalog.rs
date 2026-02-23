use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    routing::{get, post},
    Json, Router,
};
use serde::Deserialize;
use uuid::Uuid;

use crate::state::SharedState;
use heimdall_core::models::{
    ColumnMetadata, ConnectSourceRequest, DataAsset, SourceSystem,
};

pub fn router() -> Router<SharedState> {
    Router::new()
        .route("/",                                      get(list_sources).post(connect_source))
        .route("/:source_id",                            get(get_source))
        .route("/:source_id/assets",                     get(list_assets))
        .route("/:source_id/assets/:asset_id/columns",   get(list_columns))
        .route("/:source_id/pii-scan",                   post(trigger_pii_scan))
}

// GET /api/sources
async fn list_sources(
    State(state): State<SharedState>,
) -> Result<Json<Vec<SourceSystem>>, StatusCode> {
    let sources = sqlx::query_as!(
        SourceSystem,
        r#"SELECT id, name, source_type,
                  connection_mode AS "connection_mode: _",
                  status AS "status: _",
                  table_count, column_count, pii_column_count,
                  last_crawled_at, created_at
           FROM source_systems
           ORDER BY created_at DESC"#
    )
    .fetch_all(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    Ok(Json(sources))
}

// GET /api/sources/:id
async fn get_source(
    State(state): State<SharedState>,
    Path(source_id): Path<Uuid>,
) -> Result<Json<SourceSystem>, StatusCode> {
    let source = sqlx::query_as!(
        SourceSystem,
        r#"SELECT id, name, source_type,
                  connection_mode AS "connection_mode: _",
                  status AS "status: _",
                  table_count, column_count, pii_column_count,
                  last_crawled_at, created_at
           FROM source_systems WHERE id = $1"#,
        source_id
    )
    .fetch_optional(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?
    .ok_or(StatusCode::NOT_FOUND)?;

    Ok(Json(source))
}

// POST /api/sources
async fn connect_source(
    State(state): State<SharedState>,
    Json(req): Json<ConnectSourceRequest>,
) -> Result<Json<SourceSystem>, StatusCode> {
    let source = sqlx::query_as!(
        SourceSystem,
        r#"INSERT INTO source_systems (name, source_type, connection_mode)
           VALUES ($1, $2, $3)
           RETURNING id, name, source_type,
                     connection_mode AS "connection_mode: _",
                     status AS "status: _",
                     table_count, column_count, pii_column_count,
                     last_crawled_at, created_at"#,
        req.name,
        req.source_type,
        req.connection_mode as _,
    )
    .fetch_one(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    Ok(Json(source))
}

// GET /api/sources/:id/assets
#[derive(Deserialize)]
struct AssetQuery {
    schema: Option<String>,
}

async fn list_assets(
    State(state): State<SharedState>,
    Path(source_id): Path<Uuid>,
    Query(q): Query<AssetQuery>,
) -> Result<Json<Vec<DataAsset>>, StatusCode> {
    let assets = sqlx::query_as!(
        DataAsset,
        r#"SELECT id, source_id, schema_name, table_name, row_count, last_modified, created_at
           FROM data_assets
           WHERE source_id = $1
             AND ($2::text IS NULL OR schema_name = $2)
           ORDER BY schema_name, table_name"#,
        source_id,
        q.schema,
    )
    .fetch_all(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    Ok(Json(assets))
}

// GET /api/sources/:id/assets/:asset_id/columns
async fn list_columns(
    State(state): State<SharedState>,
    Path((_source_id, asset_id)): Path<(Uuid, Uuid)>,
) -> Result<Json<Vec<ColumnMetadata>>, StatusCode> {
    let columns = sqlx::query_as!(
        ColumnMetadata,
        r#"SELECT id, asset_id, name, data_type, is_pii, pii_type,
                  pii_confidence, null_pct, distinct_count, ai_description
           FROM column_metadata
           WHERE asset_id = $1
           ORDER BY name"#,
        asset_id
    )
    .fetch_all(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    Ok(Json(columns))
}

// POST /api/sources/:id/pii-scan
// Enqueues background PII classification for all unclassified columns in the source.
async fn trigger_pii_scan(
    State(state): State<SharedState>,
    Path(source_id): Path<Uuid>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let columns = sqlx::query_as!(
        ColumnMetadata,
        r#"SELECT cm.id, cm.asset_id, cm.name, cm.data_type, cm.is_pii, cm.pii_type,
                  cm.pii_confidence, cm.null_pct, cm.distinct_count, cm.ai_description
           FROM column_metadata cm
           JOIN data_assets da ON cm.asset_id = da.id
           WHERE da.source_id = $1 AND cm.is_pii IS NULL
           LIMIT 500"#,
        source_id
    )
    .fetch_all(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    let total = columns.len();
    tracing::info!(source_id = %source_id, columns = total, "PII scan triggered");

    // Spawn background task — respond immediately, scan runs async
    tokio::spawn({
        let state = state.clone();
        async move {
            // Process in batches of 20
            for chunk in columns.chunks(20) {
                match state.claude.classify_pii_batch(chunk).await {
                    Ok(results) => {
                        for (col, result) in chunk.iter().zip(results.iter()) {
                            let _ = sqlx::query!(
                                "UPDATE column_metadata
                                 SET is_pii = $1, pii_type = $2, pii_confidence = $3
                                 WHERE id = $4",
                                result.is_pii,
                                result.pii_type.as_deref(),
                                result.confidence,
                                col.id,
                            )
                            .execute(&state.db)
                            .await;
                        }

                        // Update pii_column_count on the source
                        let _ = sqlx::query!(
                            r#"UPDATE source_systems ss
                               SET pii_column_count = (
                                   SELECT COUNT(*) FROM column_metadata cm
                                   JOIN data_assets da ON cm.asset_id = da.id
                                   WHERE da.source_id = ss.id AND cm.is_pii = TRUE
                               )
                               WHERE ss.id = $1"#,
                            source_id
                        )
                        .execute(&state.db)
                        .await;
                    }
                    Err(e) => tracing::error!("PII batch error: {e}"),
                }
            }
            tracing::info!(source_id = %source_id, "PII scan complete");
        }
    });

    Ok(Json(serde_json::json!({
        "status": "scanning",
        "columns_queued": total
    })))
}
