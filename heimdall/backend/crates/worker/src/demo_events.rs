/// Demo Event Generator — fires realistic schema change events on a timer.
/// Only active when DEMO_MODE=true. Powers the "Monday 10:01am" live demo moment.
use std::time::Duration;

use sqlx::PgPool;

// Scripted demo events that tell the Heimdall story
struct DemoEvent {
    table_hint: &'static str,   // match an asset containing this string
    change_type: &'static str,
    severity: &'static str,
    description: &'static str,
    models: &'static [&'static str],
    metrics: &'static [&'static str],
    dashboards: &'static [&'static str],
    ai_analysis: &'static str,
    block: bool,
}

const DEMO_SCRIPT: &[DemoEvent] = &[
    DemoEvent {
        table_hint: "opportunity",
        change_type: "TYPE_CHANGED",
        severity: "CRITICAL",
        description: "Column 'stage_name' in salesforce.opportunity: picklist value renamed from 'Enterprise' to 'Enterprise Tier'",
        models: &["fct_revenue", "dim_accounts", "fct_pipeline"],
        metrics: &["mrr_metric", "pipeline_by_segment"],
        dashboards: &["Sales VP Dashboard"],
        ai_analysis: "CRITICAL: Picklist rename will cause downstream models to silently drop rows matching 'Enterprise'. The fct_revenue model filters on stage_name — this will show $0 revenue for the Enterprise segment until fixed. Remediation: add a coalesce or case statement to handle both values, then update the source contract.",
        block: true,
    },
    DemoEvent {
        table_hint: "subscription",
        change_type: "TYPE_CHANGED",
        severity: "WARNING",
        description: "Column 'amount' in stripe.subscriptions changed type from INTEGER to DECIMAL — value unit changed from cents to dollars",
        models: &["fct_mrr", "stg_stripe__subscriptions"],
        metrics: &["mrr_metric", "arr_metric"],
        dashboards: &["CFO Revenue Dashboard", "Finance Weekly"],
        ai_analysis: "WARNING: Type widening from INTEGER to DECIMAL is non-breaking in SQL, but the value semantics changed — amount was in cents (e.g. 4990), now in dollars (e.g. 49.90). Your MRR metric divides by 100 — this will make revenue appear 100x smaller. Remediation: remove the /100 normalization in stg_stripe__subscriptions and update the source contract.",
        block: false,
    },
    DemoEvent {
        table_hint: "customer",
        change_type: "COLUMN_REMOVED",
        severity: "CRITICAL",
        description: "Column 'customer_lifetime_value' removed from postgres.customers table",
        models: &["dim_customers", "fct_churn"],
        metrics: &["ltv_metric"],
        dashboards: &["Customer Success Dashboard"],
        ai_analysis: "CRITICAL: Column removal is a breaking change. The dim_customers model references customers.customer_lifetime_value directly — this will cause a compilation error on next dbt build. Remediation: check if column was renamed (look for 'clv' or 'ltv' in the updated schema), update staging model, add not_null test to catch future removals.",
        block: true,
    },
    DemoEvent {
        table_hint: "contact",
        change_type: "COLUMN_ADDED",
        severity: "INFO",
        description: "New column 'preferred_contact_channel' added to hubspot.contacts",
        models: &[],
        metrics: &[],
        dashboards: &[],
        ai_analysis: "INFO: New column available. No existing models are broken. This column may be valuable for marketing segmentation models. Consider adding a source contract to track it, and a not_null or accepted_values test if it becomes a key dimension.",
        block: false,
    },
    DemoEvent {
        table_hint: "order",
        change_type: "VOLUME_ANOMALY",
        severity: "WARNING",
        description: "Row count in shopify.orders dropped 87% overnight (142,000 → 18,400 rows)",
        models: &["fct_orders", "fct_revenue"],
        metrics: &["gmv_metric", "order_count"],
        dashboards: &["E-commerce Operations", "CEO Dashboard"],
        ai_analysis: "WARNING: 87% row count drop is anomalous — this is likely a source system issue (Shopify API pagination bug, connector misconfiguration, or data deletion). Extraction has NOT been blocked (existing data is intact), but downstream models will show dramatically lower numbers until resolved. Recommend investigating Shopify connector logs before next sync.",
        block: false,
    },
];

pub async fn run(pool: PgPool, enabled: bool, interval: Duration) -> anyhow::Result<()> {
    if !enabled {
        tracing::info!("Demo event generator disabled (DEMO_MODE != true)");
        // Sleep forever — keeps the tokio::select! arm alive
        futures::future::pending::<()>().await;
        return Ok(());
    }

    tracing::info!("Demo event generator started (interval: {:?})", interval);

    let mut idx = 0usize;
    loop {
        tokio::time::sleep(interval).await;

        let demo = &DEMO_SCRIPT[idx % DEMO_SCRIPT.len()];
        if let Err(e) = fire_event(&pool, demo).await {
            tracing::warn!("Demo event fire error: {e}");
        }
        idx += 1;
    }
}

async fn fire_event(pool: &PgPool, demo: &DemoEvent) -> anyhow::Result<()> {
    // Find any asset whose table_name contains the hint
    let asset = sqlx::query!(
        "SELECT id FROM data_assets WHERE table_name ILIKE $1 LIMIT 1",
        format!("%{}%", demo.table_hint)
    )
    .fetch_optional(pool)
    .await?;

    let asset_id = match asset {
        Some(a) => a.id,
        None => {
            tracing::debug!(hint = demo.table_hint, "No matching asset for demo event — skipping");
            return Ok(());
        }
    };

    let blast_radius = serde_json::json!({
        "models":      demo.models,
        "metrics":     demo.metrics,
        "dashboards":  demo.dashboards,
        "remediation": extract_remediation(demo.ai_analysis),
    });

    sqlx::query(
        "INSERT INTO schema_change_events
             (asset_id, change_type, severity, description, blast_radius, ai_analysis, blocked_extraction)
         VALUES ($1, $2::change_type, $3::change_severity, $4, $5, $6, $7)",
    )
    .bind(asset_id)
    .bind(demo.change_type)
    .bind(demo.severity)
    .bind(demo.description)
    .bind(blast_radius)
    .bind(demo.ai_analysis)
    .bind(demo.block)
    .execute(pool)
    .await?;

    tracing::info!(
        table = demo.table_hint,
        severity = demo.severity,
        blocked = demo.block,
        "Demo event fired"
    );

    Ok(())
}

fn extract_remediation(analysis: &str) -> &str {
    // Pull the text after "Remediation:" if present
    if let Some(idx) = analysis.find("Remediation:") {
        analysis[idx + "Remediation:".len()..].trim()
    } else {
        "See AI analysis."
    }
}
