use std::time::Duration;

use dotenvy::dotenv;
use sqlx::PgPool;
use tracing_subscriber::EnvFilter;

mod crawler;
mod demo_events;
mod pii_scanner;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    dotenv().ok();
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .init();

    let db_url = std::env::var("DATABASE_URL").expect("DATABASE_URL must be set");
    let anthropic_key = std::env::var("ANTHROPIC_API_KEY").expect("ANTHROPIC_API_KEY must be set");
    let demo_mode = std::env::var("DEMO_MODE").unwrap_or_default() == "true";
    let crawl_interval = std::env::var("CRAWL_INTERVAL_SECS")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(120u64);

    tracing::info!("Connecting to database...");
    let pool = PgPool::connect(&db_url).await?;

    let claude = heimdall_core::ai::ClaudeClient::new(anthropic_key);

    tracing::info!(demo_mode, crawl_interval, "Starting Heimdall worker");

    // Spawn all background loops concurrently
    tokio::select! {
        r = crawler::run(pool.clone(), Duration::from_secs(crawl_interval)) => {
            tracing::error!("Crawler exited: {:?}", r);
        }
        r = pii_scanner::run(pool.clone(), claude.clone(), Duration::from_secs(60)) => {
            tracing::error!("PII scanner exited: {:?}", r);
        }
        r = demo_events::run(pool.clone(), demo_mode, Duration::from_secs(30)) => {
            tracing::error!("Demo event generator exited: {:?}", r);
        }
    }

    Ok(())
}
