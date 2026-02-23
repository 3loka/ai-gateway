use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{get, post},
    Json, Router,
};
use uuid::Uuid;

use crate::state::SharedState;
use heimdall_core::{
    models::{Policy, PolicyDecision},
    policy as policy_engine,
};

pub fn router() -> Router<SharedState> {
    Router::new()
        .route("/",              get(list_policies).post(create_policy))
        .route("/:id",           get(get_policy))
        .route("/evaluate",      post(evaluate_policy))
}

// GET /api/policies
async fn list_policies(
    State(state): State<SharedState>,
) -> Result<Json<Vec<Policy>>, StatusCode> {
    let policies = sqlx::query_as!(
        Policy,
        "SELECT id, name, yaml_definition, applies_to_source,
                is_active, created_by, created_at
         FROM policies
         ORDER BY created_at DESC"
    )
    .fetch_all(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    Ok(Json(policies))
}

// GET /api/policies/:id
async fn get_policy(
    State(state): State<SharedState>,
    Path(id): Path<Uuid>,
) -> Result<Json<Policy>, StatusCode> {
    let policy = sqlx::query_as!(
        Policy,
        "SELECT id, name, yaml_definition, applies_to_source,
                is_active, created_by, created_at
         FROM policies WHERE id = $1",
        id
    )
    .fetch_optional(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?
    .ok_or(StatusCode::NOT_FOUND)?;

    Ok(Json(policy))
}

// POST /api/policies
async fn create_policy(
    State(state): State<SharedState>,
    Json(body): Json<serde_json::Value>,
) -> Result<Json<Policy>, StatusCode> {
    let name = body.get("name").and_then(|v| v.as_str()).unwrap_or("unnamed");
    let yaml = body
        .get("yaml_definition")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let created_by = body.get("created_by").and_then(|v| v.as_str()).unwrap_or("system");
    let applies_to = body.get("applies_to_source").and_then(|v| v.as_str());

    // Validate YAML parses correctly before storing
    policy_engine::parse_yaml(yaml).map_err(|e| {
        tracing::warn!("Invalid policy YAML: {e}");
        StatusCode::BAD_REQUEST
    })?;

    let policy = sqlx::query_as!(
        Policy,
        "INSERT INTO policies (name, yaml_definition, applies_to_source, created_by)
         VALUES ($1, $2, $3, $4)
         RETURNING id, name, yaml_definition, applies_to_source,
                   is_active, created_by, created_at",
        name,
        yaml,
        applies_to,
        created_by,
    )
    .fetch_one(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    Ok(Json(policy))
}

// POST /api/policies/evaluate
// Body: { "asset_id": "<uuid>", "policy_yaml": "..." }
async fn evaluate_policy(
    State(state): State<SharedState>,
    Json(body): Json<serde_json::Value>,
) -> Result<Json<PolicyDecision>, StatusCode> {
    let asset_id: Uuid = body
        .get("asset_id")
        .and_then(|v| v.as_str())
        .and_then(|s| s.parse().ok())
        .ok_or(StatusCode::BAD_REQUEST)?;

    let yaml = body
        .get("policy_yaml")
        .and_then(|v| v.as_str())
        .ok_or(StatusCode::BAD_REQUEST)?;

    let policy_def = policy_engine::parse_yaml(yaml).map_err(|e| {
        tracing::warn!("Invalid policy YAML: {e}");
        StatusCode::BAD_REQUEST
    })?;

    let columns = sqlx::query_as!(
        heimdall_core::models::ColumnMetadata,
        r#"SELECT id, asset_id, name, data_type, is_pii, pii_type,
                  pii_confidence, null_pct, distinct_count, ai_description
           FROM column_metadata WHERE asset_id = $1"#,
        asset_id
    )
    .fetch_all(&state.db)
    .await
    .map_err(|e| { tracing::error!("{e}"); StatusCode::INTERNAL_SERVER_ERROR })?;

    let decision = policy_engine::evaluate(&policy_def, &columns);
    Ok(Json(decision))
}
