use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::Sse,
    routing::{get, post},
    Json, Router,
};
use futures::Stream;
use serde::Serialize;
use std::{convert::Infallible, time::Duration};
use uuid::Uuid;

use crate::state::SharedState;
use heimdall_core::models::{BlastRadius, SchemaChangeEvent};

pub fn router() -> Router<SharedState> {
    Router::new()
        .route("/",              get(list_changes))
        .route("/stream",        get(stream_changes))
        .route("/:id",           get(get_change))
        .route("/:id/blast-radius", get(get_blast_radius))
        .route("/:id/resolve",   post(resolve_change))
}

// GET /api/changes  — recent changes, most critical first
async fn list_changes(
    State(state): State<SharedState>,
) -> Result<Json<Vec<SchemaChangeEvent>>, StatusCode> {
    let events = sqlx::query_as!(
        SchemaChangeEvent,
        r#"SELECT id, asset_id,
                  change_type AS "change_type: _",
                  severity AS "severity: _",
                  description, blast_radius, ai_analysis,
                  blocked_extraction, resolved, resolved_by, detected_at
           FROM schema_change_events
           ORDER BY
             CASE severity WHEN 'CRITICAL' THEN 0 WHEN 'WARNING' THEN 1 ELSE 2 END,
             detected_at DESC
           LIMIT 100"#
    )
    .fetch_all(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    Ok(Json(events))
}

// GET /api/changes/:id
async fn get_change(
    State(state): State<SharedState>,
    Path(id): Path<Uuid>,
) -> Result<Json<SchemaChangeEvent>, StatusCode> {
    let event = sqlx::query_as!(
        SchemaChangeEvent,
        r#"SELECT id, asset_id,
                  change_type AS "change_type: _",
                  severity AS "severity: _",
                  description, blast_radius, ai_analysis,
                  blocked_extraction, resolved, resolved_by, detected_at
           FROM schema_change_events WHERE id = $1"#,
        id
    )
    .fetch_optional(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?
    .ok_or(StatusCode::NOT_FOUND)?;

    Ok(Json(event))
}

// GET /api/changes/:id/blast-radius
async fn get_blast_radius(
    State(state): State<SharedState>,
    Path(id): Path<Uuid>,
) -> Result<Json<BlastRadius>, StatusCode> {
    let event = sqlx::query!(
        "SELECT id, blast_radius, ai_analysis FROM schema_change_events WHERE id = $1",
        id
    )
    .fetch_optional(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?
    .ok_or(StatusCode::NOT_FOUND)?;

    let blast_radius = event.blast_radius.unwrap_or_default();

    let result = BlastRadius {
        change_id: event.id,
        affected_models: blast_radius
            .get("models")
            .and_then(|v| serde_json::from_value(v.clone()).ok())
            .unwrap_or_default(),
        affected_metrics: blast_radius
            .get("metrics")
            .and_then(|v| serde_json::from_value(v.clone()).ok())
            .unwrap_or_default(),
        affected_dashboards: blast_radius
            .get("dashboards")
            .and_then(|v| serde_json::from_value(v.clone()).ok())
            .unwrap_or_default(),
        ai_analysis: event.ai_analysis.unwrap_or_default(),
        ai_remediation: blast_radius
            .get("remediation")
            .and_then(|v| v.as_str())
            .unwrap_or("See AI analysis above.")
            .to_string(),
    };

    Ok(Json(result))
}

// POST /api/changes/:id/resolve
async fn resolve_change(
    State(state): State<SharedState>,
    Path(id): Path<Uuid>,
    Json(body): Json<serde_json::Value>,
) -> Result<StatusCode, StatusCode> {
    let actor = body
        .get("actor")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown");

    sqlx::query!(
        "UPDATE schema_change_events
         SET resolved = TRUE, resolved_by = $1, blocked_extraction = FALSE
         WHERE id = $2",
        actor,
        id
    )
    .execute(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    // Audit log
    sqlx::query!(
        "INSERT INTO audit_logs (event_type, actor)
         VALUES ('CHANGE_RESOLVED', $1)",
        actor
    )
    .execute(&state.db)
    .await
    .ok();

    Ok(StatusCode::NO_CONTENT)
}

// GET /api/changes/stream  — Server-Sent Events for live change feed
async fn stream_changes(
    State(state): State<SharedState>,
) -> Sse<impl Stream<Item = Result<axum::response::sse::Event, Infallible>>> {
    let stream = async_stream::stream! {
        let mut last_detected: Option<chrono::DateTime<chrono::Utc>> = None;

        loop {
            let result = if let Some(ts) = last_detected {
                sqlx::query_as!(
                    SchemaChangeEvent,
                    r#"SELECT id, asset_id,
                              change_type AS "change_type: _",
                              severity AS "severity: _",
                              description, blast_radius, ai_analysis,
                              blocked_extraction, resolved, resolved_by, detected_at
                       FROM schema_change_events
                       WHERE detected_at > $1
                       ORDER BY detected_at ASC"#,
                    ts
                )
                .fetch_all(&state.db)
                .await
            } else {
                // First poll — send the 10 most recent events to hydrate the UI
                sqlx::query_as!(
                    SchemaChangeEvent,
                    r#"SELECT id, asset_id,
                              change_type AS "change_type: _",
                              severity AS "severity: _",
                              description, blast_radius, ai_analysis,
                              blocked_extraction, resolved, resolved_by, detected_at
                       FROM schema_change_events
                       ORDER BY detected_at DESC
                       LIMIT 10"#
                )
                .fetch_all(&state.db)
                .await
            };

            match result {
                Ok(events) => {
                    for event in &events {
                        if let Ok(data) = serde_json::to_string(event) {
                            yield Ok(axum::response::sse::Event::default()
                                .event("change")
                                .data(data));
                        }
                        last_detected = Some(event.detected_at);
                    }
                }
                Err(e) => tracing::error!("SSE query error: {e}"),
            }

            tokio::time::sleep(Duration::from_secs(2)).await;
        }
    };

    Sse::new(stream).keep_alive(
        axum::response::sse::KeepAlive::new()
            .interval(Duration::from_secs(15))
            .text("ping"),
    )
}
