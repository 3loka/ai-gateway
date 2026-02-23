/// Seed binary — run once to populate demo data.
/// Usage: cargo run -p heimdall-db
use sqlx::PgPool;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();
    tracing_subscriber::fmt().init();

    let db_url = std::env::var("DATABASE_URL").expect("DATABASE_URL must be set");
    let pool = PgPool::connect(&db_url).await?;

    heimdall_db::seed::run(&pool).await?;
    Ok(())
}
