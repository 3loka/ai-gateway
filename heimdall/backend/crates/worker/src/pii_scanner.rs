/// PII Scanner — periodically finds unclassified columns and runs Claude batch classification.
use std::time::Duration;

use sqlx::PgPool;

use heimdall_core::ai::ClaudeClient;

pub async fn run(pool: PgPool, claude: ClaudeClient, interval: Duration) -> anyhow::Result<()> {
    tracing::info!("PII scanner started (interval: {:?})", interval);
    loop {
        if let Err(e) = scan_unclassified(&pool, &claude).await {
            tracing::error!("PII scan error: {e}");
        }
        tokio::time::sleep(interval).await;
    }
}

async fn scan_unclassified(pool: &PgPool, claude: &ClaudeClient) -> anyhow::Result<()> {
    let columns = sqlx::query_as!(
        heimdall_core::models::ColumnMetadata,
        r#"SELECT id, asset_id, name, data_type, is_pii, pii_type,
                  pii_confidence, null_pct, distinct_count, ai_description
           FROM column_metadata
           WHERE is_pii IS NULL
           LIMIT 100"#
    )
    .fetch_all(pool)
    .await?;

    if columns.is_empty() {
        return Ok(());
    }

    tracing::info!(count = columns.len(), "Classifying unclassified columns");

    for chunk in columns.chunks(20) {
        match claude.classify_pii_batch(chunk).await {
            Ok(results) => {
                for (col, result) in chunk.iter().zip(results.iter()) {
                    sqlx::query!(
                        "UPDATE column_metadata
                         SET is_pii = $1, pii_type = $2, pii_confidence = $3
                         WHERE id = $4",
                        result.is_pii,
                        result.pii_type.as_deref(),
                        result.confidence,
                        col.id,
                    )
                    .execute(pool)
                    .await?;
                }

                // Recompute pii_column_count for all affected sources
                sqlx::query!(
                    r#"UPDATE source_systems ss
                       SET pii_column_count = (
                           SELECT COUNT(*) FROM column_metadata cm
                           JOIN data_assets da ON cm.asset_id = da.id
                           WHERE da.source_id = ss.id AND cm.is_pii = TRUE
                       )"#
                )
                .execute(pool)
                .await?;
            }
            Err(e) => tracing::error!("Batch PII classification error: {e}"),
        }

        // Brief pause between batches to respect API rate limits
        tokio::time::sleep(Duration::from_millis(500)).await;
    }

    Ok(())
}
