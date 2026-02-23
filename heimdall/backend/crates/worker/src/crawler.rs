/// Schema Crawler — polls Fivetran schema reload API for each connected source
/// and detects structural changes by diffing consecutive snapshots.
use std::time::Duration;

use sqlx::PgPool;
use uuid::Uuid;

pub async fn run(pool: PgPool, interval: Duration) -> anyhow::Result<()> {
    tracing::info!("Schema crawler started (interval: {:?})", interval);
    loop {
        if let Err(e) = crawl_all(&pool).await {
            tracing::error!("Crawl cycle error: {e}");
        }
        tokio::time::sleep(interval).await;
    }
}

async fn crawl_all(pool: &PgPool) -> anyhow::Result<()> {
    // Only crawl sources that have a Fivetran connector ID configured
    let sources = sqlx::query!(
        "SELECT id, name, fivetran_connector_id
         FROM source_systems
         WHERE fivetran_connector_id IS NOT NULL
           AND status != 'ERROR'"
    )
    .fetch_all(pool)
    .await?;

    tracing::debug!(count = sources.len(), "Crawling sources with Fivetran connectors");

    for source in sources {
        if let Some(connector_id) = source.fivetran_connector_id {
            if let Err(e) = crawl_source(pool, source.id, &connector_id, &source.name).await {
                tracing::warn!(source = %source.name, "Crawl failed: {e}");
                sqlx::query!(
                    "UPDATE source_systems SET status = 'ERROR' WHERE id = $1",
                    source.id
                )
                .execute(pool)
                .await
                .ok();
            }
        }
    }

    Ok(())
}

async fn crawl_source(
    pool: &PgPool,
    source_id: Uuid,
    connector_id: &str,
    source_name: &str,
) -> anyhow::Result<()> {
    let api_key = std::env::var("FIVETRAN_API_KEY").unwrap_or_default();
    let api_secret = std::env::var("FIVETRAN_API_SECRET").unwrap_or_default();

    // Mark as crawling
    sqlx::query!(
        "UPDATE source_systems SET status = 'CRAWLING' WHERE id = $1",
        source_id
    )
    .execute(pool)
    .await?;

    // Call Fivetran schema reload API
    let client = reqwest::Client::new();
    let url = format!("https://api.fivetran.com/v1/connectors/{connector_id}/schemas");
    let resp = client
        .get(&url)
        .basic_auth(&api_key, Some(&api_secret))
        .send()
        .await?;

    if !resp.status().is_success() {
        anyhow::bail!("Fivetran API returned {}", resp.status());
    }

    let schema_json: serde_json::Value = resp.json().await?;

    // Snapshot and diff
    process_schema_snapshot(pool, source_id, source_name, &schema_json).await?;

    // Mark healthy
    sqlx::query!(
        "UPDATE source_systems
         SET status = 'HEALTHY', last_crawled_at = NOW()
         WHERE id = $1",
        source_id
    )
    .execute(pool)
    .await?;

    tracing::debug!(source = %source_name, "Crawl complete");
    Ok(())
}

async fn process_schema_snapshot(
    pool: &PgPool,
    source_id: Uuid,
    source_name: &str,
    schema_json: &serde_json::Value,
) -> anyhow::Result<()> {
    // Parse tables from Fivetran response shape: { data: { schemas: { <schema>: { tables: {...} } } } }
    let schemas = schema_json
        .pointer("/data/schemas")
        .and_then(|v| v.as_object())
        .cloned()
        .unwrap_or_default();

    for (schema_name, schema_val) in &schemas {
        let tables = schema_val
            .get("tables")
            .and_then(|v| v.as_object())
            .cloned()
            .unwrap_or_default();

        for (table_name, table_val) in &tables {
            // Upsert data_asset
            let asset = sqlx::query!(
                r#"INSERT INTO data_assets (source_id, schema_name, table_name)
                   VALUES ($1, $2, $3)
                   ON CONFLICT (source_id, schema_name, table_name) DO UPDATE
                     SET last_modified = NOW()
                   RETURNING id"#,
                source_id,
                schema_name,
                table_name,
            )
            .fetch_one(pool)
            .await?;

            // Snapshot current schema
            sqlx::query!(
                "INSERT INTO schema_snapshots (asset_id, schema_json)
                 VALUES ($1, $2)",
                asset.id,
                table_val
            )
            .execute(pool)
            .await?;

            // Upsert columns
            if let Some(columns) = table_val.get("columns").and_then(|v| v.as_object()) {
                for (col_name, col_val) in columns {
                    let data_type = col_val
                        .get("data_type")
                        .and_then(|v| v.as_str())
                        .unwrap_or("UNKNOWN");

                    sqlx::query!(
                        r#"INSERT INTO column_metadata (asset_id, name, data_type)
                           VALUES ($1, $2, $3)
                           ON CONFLICT (asset_id, name) DO UPDATE
                             SET data_type = EXCLUDED.data_type"#,
                        asset.id,
                        col_name,
                        data_type,
                    )
                    .execute(pool)
                    .await?;
                }
            }
        }
    }

    // Update counts on source
    sqlx::query!(
        r#"UPDATE source_systems
           SET table_count  = (SELECT COUNT(*) FROM data_assets WHERE source_id = $1),
               column_count = (SELECT COUNT(*) FROM column_metadata cm
                               JOIN data_assets da ON cm.asset_id = da.id
                               WHERE da.source_id = $1)
           WHERE id = $1"#,
        source_id
    )
    .execute(pool)
    .await?;

    tracing::debug!(source = %source_name, "Schema snapshot stored");
    Ok(())
}
