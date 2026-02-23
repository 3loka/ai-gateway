use axum::Router;
use sqlx::PgPool;
use std::sync::Arc;
use tower_http::cors::CorsLayer;
use tracing_subscriber::EnvFilter;

mod routes;
mod state;

use state::AppState;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();

    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .init();

    let db_url = std::env::var("DATABASE_URL")
        .expect("DATABASE_URL must be set");
    let anthropic_key = std::env::var("ANTHROPIC_API_KEY")
        .expect("ANTHROPIC_API_KEY must be set");

    tracing::info!("Connecting to database...");
    let pool = PgPool::connect(&db_url).await?;
    sqlx::migrate!("../db/migrations").run(&pool).await?;
    tracing::info!("Migrations applied ✓");

    let state = Arc::new(AppState::new(pool, anthropic_key));

    let app = Router::new()
        .nest("/api/dashboard", routes::dashboard::router())
        .nest("/api/sources",   routes::catalog::router())
        .nest("/api/changes",   routes::changes::router())
        .nest("/api/policies",  routes::policies::router())
        .nest("/api/audit",     routes::audit::router())
        .layer(CorsLayer::permissive())
        .with_state(state);

    let addr = "0.0.0.0:8080";
    tracing::info!("Heimdall API listening on {addr}");
    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;
    Ok(())
}
