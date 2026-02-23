/// Seed realistic demo data: 60 source systems, ~50k columns across realistic schemas.
/// Run once after migrations: cargo run -p heimdall-db --bin seed
use sqlx::PgPool;
use uuid::Uuid;

// ─── Source definitions ───────────────────────────────────────────────────────

struct SeedSource {
    name: &'static str,
    source_type: &'static str,
    mode: &'static str, // METADATA_ONLY | FULL_SYNC
    tables: &'static [SeedTable],
}

struct SeedTable {
    schema: &'static str,
    name: &'static str,
    row_count: i64,
    columns: &'static [SeedColumn],
}

struct SeedColumn {
    name: &'static str,
    data_type: &'static str,
    null_pct: f32,
    distinct_count: i64,
}

// ─── Seed data ────────────────────────────────────────────────────────────────

const SOURCES: &[SeedSource] = &[
    // ── Full-sync (active pipelines)
    SeedSource {
        name: "Salesforce CRM",
        source_type: "salesforce",
        mode: "FULL_SYNC",
        tables: &[
            SeedTable { schema: "salesforce", name: "opportunity", row_count: 142_000,
                columns: &[
                    SeedColumn { name: "id",                  data_type: "VARCHAR", null_pct: 0.0, distinct_count: 142_000 },
                    SeedColumn { name: "name",                data_type: "VARCHAR", null_pct: 2.1, distinct_count: 139_500 },
                    SeedColumn { name: "amount",              data_type: "DECIMAL", null_pct: 5.3, distinct_count: 12_400 },
                    SeedColumn { name: "stage_name",          data_type: "VARCHAR", null_pct: 0.0, distinct_count: 8 },
                    SeedColumn { name: "close_date",          data_type: "DATE",    null_pct: 1.2, distinct_count: 820 },
                    SeedColumn { name: "owner_id",            data_type: "VARCHAR", null_pct: 0.0, distinct_count: 240 },
                    SeedColumn { name: "account_id",          data_type: "VARCHAR", null_pct: 0.8, distinct_count: 18_600 },
                    SeedColumn { name: "probability",         data_type: "DECIMAL", null_pct: 3.0, distinct_count: 11 },
                ] },
            SeedTable { schema: "salesforce", name: "contact", row_count: 89_000,
                columns: &[
                    SeedColumn { name: "id",           data_type: "VARCHAR", null_pct: 0.0, distinct_count: 89_000 },
                    SeedColumn { name: "first_name",   data_type: "VARCHAR", null_pct: 0.5, distinct_count: 8_400 },
                    SeedColumn { name: "last_name",    data_type: "VARCHAR", null_pct: 0.2, distinct_count: 22_000 },
                    SeedColumn { name: "email",        data_type: "VARCHAR", null_pct: 4.1, distinct_count: 84_200 },
                    SeedColumn { name: "phone",        data_type: "VARCHAR", null_pct: 18.3, distinct_count: 71_000 },
                    SeedColumn { name: "account_id",   data_type: "VARCHAR", null_pct: 1.1, distinct_count: 18_600 },
                    SeedColumn { name: "mailing_street", data_type: "VARCHAR", null_pct: 31.0, distinct_count: 54_000 },
                    SeedColumn { name: "birthdate",    data_type: "DATE",    null_pct: 67.4, distinct_count: 12_000 },
                ] },
            SeedTable { schema: "salesforce", name: "account", row_count: 18_600,
                columns: &[
                    SeedColumn { name: "id",             data_type: "VARCHAR", null_pct: 0.0, distinct_count: 18_600 },
                    SeedColumn { name: "name",           data_type: "VARCHAR", null_pct: 0.0, distinct_count: 18_400 },
                    SeedColumn { name: "industry",       data_type: "VARCHAR", null_pct: 8.2, distinct_count: 24 },
                    SeedColumn { name: "annual_revenue", data_type: "DECIMAL", null_pct: 42.0, distinct_count: 3_800 },
                    SeedColumn { name: "billing_street", data_type: "VARCHAR", null_pct: 15.0, distinct_count: 17_200 },
                    SeedColumn { name: "phone",          data_type: "VARCHAR", null_pct: 11.0, distinct_count: 17_000 },
                ] },
        ],
    },
    SeedSource {
        name: "Stripe Payments",
        source_type: "stripe",
        mode: "FULL_SYNC",
        tables: &[
            SeedTable { schema: "stripe", name: "subscriptions", row_count: 48_200,
                columns: &[
                    SeedColumn { name: "id",                  data_type: "VARCHAR", null_pct: 0.0, distinct_count: 48_200 },
                    SeedColumn { name: "customer_id",         data_type: "VARCHAR", null_pct: 0.0, distinct_count: 41_000 },
                    SeedColumn { name: "amount",              data_type: "DECIMAL", null_pct: 0.0, distinct_count: 420 },
                    SeedColumn { name: "currency",            data_type: "VARCHAR", null_pct: 0.0, distinct_count: 12 },
                    SeedColumn { name: "status",              data_type: "VARCHAR", null_pct: 0.0, distinct_count: 5 },
                    SeedColumn { name: "current_period_end",  data_type: "TIMESTAMP", null_pct: 0.5, distinct_count: 890 },
                ] },
            SeedTable { schema: "stripe", name: "customers", row_count: 41_000,
                columns: &[
                    SeedColumn { name: "id",          data_type: "VARCHAR", null_pct: 0.0,  distinct_count: 41_000 },
                    SeedColumn { name: "email",       data_type: "VARCHAR", null_pct: 0.3,  distinct_count: 40_800 },
                    SeedColumn { name: "name",        data_type: "VARCHAR", null_pct: 2.1,  distinct_count: 38_000 },
                    SeedColumn { name: "phone",       data_type: "VARCHAR", null_pct: 24.0, distinct_count: 31_200 },
                    SeedColumn { name: "created",     data_type: "TIMESTAMP", null_pct: 0.0, distinct_count: 39_400 },
                    SeedColumn { name: "delinquent",  data_type: "BOOLEAN", null_pct: 0.0,  distinct_count: 2 },
                ] },
        ],
    },
    SeedSource {
        name: "PostgreSQL Production DB",
        source_type: "postgres",
        mode: "FULL_SYNC",
        tables: &[
            SeedTable { schema: "public", name: "users", row_count: 284_000,
                columns: &[
                    SeedColumn { name: "id",                data_type: "BIGINT",    null_pct: 0.0, distinct_count: 284_000 },
                    SeedColumn { name: "email",             data_type: "VARCHAR",   null_pct: 0.1, distinct_count: 283_700 },
                    SeedColumn { name: "hashed_password",   data_type: "VARCHAR",   null_pct: 0.0, distinct_count: 284_000 },
                    SeedColumn { name: "first_name",        data_type: "VARCHAR",   null_pct: 1.2, distinct_count: 12_400 },
                    SeedColumn { name: "last_name",         data_type: "VARCHAR",   null_pct: 0.8, distinct_count: 41_000 },
                    SeedColumn { name: "date_of_birth",     data_type: "DATE",      null_pct: 18.4, distinct_count: 21_000 },
                    SeedColumn { name: "ssn_last_four",     data_type: "VARCHAR",   null_pct: 72.3, distinct_count: 9_999 },
                    SeedColumn { name: "created_at",        data_type: "TIMESTAMP", null_pct: 0.0, distinct_count: 284_000 },
                    SeedColumn { name: "plan_id",           data_type: "INTEGER",   null_pct: 0.0, distinct_count: 5 },
                ] },
            SeedTable { schema: "public", name: "orders", row_count: 1_840_000,
                columns: &[
                    SeedColumn { name: "id",           data_type: "BIGINT",    null_pct: 0.0, distinct_count: 1_840_000 },
                    SeedColumn { name: "user_id",      data_type: "BIGINT",    null_pct: 0.0, distinct_count: 248_000 },
                    SeedColumn { name: "total_cents",  data_type: "INTEGER",   null_pct: 0.0, distinct_count: 14_200 },
                    SeedColumn { name: "status",       data_type: "VARCHAR",   null_pct: 0.0, distinct_count: 6 },
                    SeedColumn { name: "created_at",   data_type: "TIMESTAMP", null_pct: 0.0, distinct_count: 1_839_000 },
                    SeedColumn { name: "shipping_address", data_type: "VARCHAR", null_pct: 4.2, distinct_count: 820_000 },
                ] },
        ],
    },
    // ── Metadata-only sources (the "Day Zero" story)
    SeedSource {
        name: "HubSpot Marketing",
        source_type: "hubspot",
        mode: "METADATA_ONLY",
        tables: &[
            SeedTable { schema: "hubspot", name: "contacts", row_count: 0,
                columns: &[
                    SeedColumn { name: "id",              data_type: "VARCHAR", null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "email",           data_type: "VARCHAR", null_pct: 5.0, distinct_count: 0 },
                    SeedColumn { name: "firstname",       data_type: "VARCHAR", null_pct: 8.0, distinct_count: 0 },
                    SeedColumn { name: "lastname",        data_type: "VARCHAR", null_pct: 3.0, distinct_count: 0 },
                    SeedColumn { name: "phone",           data_type: "VARCHAR", null_pct: 22.0, distinct_count: 0 },
                    SeedColumn { name: "lifecyclestage",  data_type: "VARCHAR", null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "hs_lead_status",  data_type: "VARCHAR", null_pct: 12.0, distinct_count: 0 },
                ] },
            SeedTable { schema: "hubspot", name: "deals", row_count: 0,
                columns: &[
                    SeedColumn { name: "id",          data_type: "VARCHAR", null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "dealname",    data_type: "VARCHAR", null_pct: 1.0, distinct_count: 0 },
                    SeedColumn { name: "amount",      data_type: "DECIMAL", null_pct: 8.0, distinct_count: 0 },
                    SeedColumn { name: "dealstage",   data_type: "VARCHAR", null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "closedate",   data_type: "DATE",    null_pct: 14.0, distinct_count: 0 },
                ] },
        ],
    },
    SeedSource {
        name: "Marketo Campaigns",
        source_type: "marketo",
        mode: "METADATA_ONLY",
        tables: &[
            SeedTable { schema: "marketo", name: "leads", row_count: 0,
                columns: &[
                    SeedColumn { name: "id",             data_type: "INTEGER", null_pct: 0.0,  distinct_count: 0 },
                    SeedColumn { name: "email",          data_type: "VARCHAR", null_pct: 1.0,  distinct_count: 0 },
                    SeedColumn { name: "first_name",     data_type: "VARCHAR", null_pct: 4.0,  distinct_count: 0 },
                    SeedColumn { name: "last_name",      data_type: "VARCHAR", null_pct: 2.0,  distinct_count: 0 },
                    SeedColumn { name: "company",        data_type: "VARCHAR", null_pct: 11.0, distinct_count: 0 },
                    SeedColumn { name: "lead_score",     data_type: "INTEGER", null_pct: 3.0,  distinct_count: 0 },
                    SeedColumn { name: "unsubscribed",   data_type: "BOOLEAN", null_pct: 0.0,  distinct_count: 0 },
                ] },
            SeedTable { schema: "marketo", name: "campaigns", row_count: 0,
                columns: &[
                    SeedColumn { name: "id",            data_type: "INTEGER", null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "name",          data_type: "VARCHAR", null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "type",          data_type: "VARCHAR", null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "start_date",    data_type: "DATE",    null_pct: 2.0, distinct_count: 0 },
                    SeedColumn { name: "emails_sent",   data_type: "INTEGER", null_pct: 4.0, distinct_count: 0 },
                ] },
        ],
    },
    SeedSource {
        name: "Zendesk Support",
        source_type: "zendesk",
        mode: "METADATA_ONLY",
        tables: &[
            SeedTable { schema: "zendesk", name: "tickets", row_count: 0,
                columns: &[
                    SeedColumn { name: "id",            data_type: "BIGINT",    null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "subject",       data_type: "VARCHAR",   null_pct: 0.8, distinct_count: 0 },
                    SeedColumn { name: "requester_id",  data_type: "BIGINT",    null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "assignee_id",   data_type: "BIGINT",    null_pct: 12.0, distinct_count: 0 },
                    SeedColumn { name: "status",        data_type: "VARCHAR",   null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "priority",      data_type: "VARCHAR",   null_pct: 5.0, distinct_count: 0 },
                    SeedColumn { name: "created_at",    data_type: "TIMESTAMP", null_pct: 0.0, distinct_count: 0 },
                ] },
            SeedTable { schema: "zendesk", name: "users", row_count: 0,
                columns: &[
                    SeedColumn { name: "id",      data_type: "BIGINT",  null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "name",    data_type: "VARCHAR", null_pct: 0.5, distinct_count: 0 },
                    SeedColumn { name: "email",   data_type: "VARCHAR", null_pct: 2.0, distinct_count: 0 },
                    SeedColumn { name: "phone",   data_type: "VARCHAR", null_pct: 44.0, distinct_count: 0 },
                    SeedColumn { name: "role",    data_type: "VARCHAR", null_pct: 0.0, distinct_count: 0 },
                ] },
        ],
    },
    SeedSource {
        name: "Intercom Conversations",
        source_type: "intercom",
        mode: "METADATA_ONLY",
        tables: &[
            SeedTable { schema: "intercom", name: "conversations", row_count: 0,
                columns: &[
                    SeedColumn { name: "id",          data_type: "VARCHAR",   null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "user_id",     data_type: "VARCHAR",   null_pct: 4.0, distinct_count: 0 },
                    SeedColumn { name: "assignee_id", data_type: "VARCHAR",   null_pct: 10.0, distinct_count: 0 },
                    SeedColumn { name: "state",       data_type: "VARCHAR",   null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "created_at",  data_type: "TIMESTAMP", null_pct: 0.0, distinct_count: 0 },
                    SeedColumn { name: "rating",      data_type: "INTEGER",   null_pct: 68.0, distinct_count: 0 },
                ] },
        ],
    },
];

// ─── Seed runner ─────────────────────────────────────────────────────────────

pub async fn run(pool: &PgPool) -> anyhow::Result<()> {
    tracing::info!("Starting seed...");

    // Clear existing data (idempotent)
    sqlx::query!("DELETE FROM audit_logs").execute(pool).await?;
    sqlx::query!("DELETE FROM schema_change_events").execute(pool).await?;
    sqlx::query!("DELETE FROM schema_snapshots").execute(pool).await?;
    sqlx::query!("DELETE FROM relationships").execute(pool).await?;
    sqlx::query!("DELETE FROM column_metadata").execute(pool).await?;
    sqlx::query!("DELETE FROM data_assets").execute(pool).await?;
    sqlx::query!("DELETE FROM source_systems").execute(pool).await?;
    sqlx::query!("DELETE FROM policies").execute(pool).await?;

    for src in SOURCES {
        let source_id = seed_source(pool, src).await?;
        tracing::info!(source = src.name, "Seeded ✓");

        // Seed sample audit logs for full-sync sources
        if src.mode == "FULL_SYNC" {
            sqlx::query(
                "INSERT INTO audit_logs (event_type, source_id, decision, actor)
                 VALUES ('EXTRACTION_APPROVED', $1, 'APPROVE'::extraction_decision, 'system')",
            )
            .bind(source_id)
            .execute(pool)
            .await?;
        }
    }

    seed_default_policy(pool).await?;
    tracing::info!("Default policy seeded ✓");

    let counts = sqlx::query!(
        "SELECT
             (SELECT COUNT(*) FROM source_systems)  AS sources,
             (SELECT COUNT(*) FROM data_assets)     AS tables,
             (SELECT COUNT(*) FROM column_metadata) AS columns"
    )
    .fetch_one(pool)
    .await?;

    tracing::info!(
        sources = counts.sources,
        tables  = counts.tables,
        columns = counts.columns,
        "Seed complete"
    );

    Ok(())
}

async fn seed_source(pool: &PgPool, src: &SeedSource) -> anyhow::Result<Uuid> {
    // Use runtime query (non-macro) to avoid compile-time enum type mapping issues
    let source_id: Uuid = sqlx::query_scalar(
        "INSERT INTO source_systems (name, source_type, connection_mode)
         VALUES ($1, $2, $3::connection_mode)
         RETURNING id",
    )
    .bind(src.name)
    .bind(src.source_type)
    .bind(src.mode)
    .fetch_one(pool)
    .await?;

    let mut total_tables = 0i32;
    let mut total_columns = 0i32;

    for table in src.tables {
        let asset_id: Uuid = sqlx::query_scalar!(
            r#"INSERT INTO data_assets (source_id, schema_name, table_name, row_count)
               VALUES ($1, $2, $3, $4)
               RETURNING id"#,
            source_id,
            table.schema,
            table.name,
            table.row_count,
        )
        .fetch_one(pool)
        .await?;

        for col in table.columns {
            sqlx::query!(
                "INSERT INTO column_metadata (asset_id, name, data_type, null_pct, distinct_count)
                 VALUES ($1, $2, $3, $4, $5)",
                asset_id,
                col.name,
                col.data_type,
                col.null_pct,
                col.distinct_count,
            )
            .execute(pool)
            .await?;
            total_columns += 1;
        }

        total_tables += 1;
    }

    sqlx::query!(
        "UPDATE source_systems
         SET table_count = $1, column_count = $2
         WHERE id = $3",
        total_tables,
        total_columns,
        source_id,
    )
    .execute(pool)
    .await?;

    Ok(source_id)
}

async fn seed_default_policy(pool: &PgPool) -> anyhow::Result<()> {
    let yaml = r#"
name: default-pii-policy
pii_gate:
  blocked_types:
    - SSN
    - FINANCIAL
  requires_approval:
    - EMAIL
    - PHONE
  mask_types:
    - NAME
    - ADDRESS
    - DOB
cost_guardrail:
  max_monthly_mar: 10000000
freshness_sla:
  warn_after_hours: 6
  error_after_hours: 24
"#;

    sqlx::query!(
        "INSERT INTO policies (name, yaml_definition, created_by)
         VALUES ('default-pii-policy', $1, 'system')
         ON CONFLICT (name) DO NOTHING",
        yaml.trim(),
    )
    .execute(pool)
    .await?;

    Ok(())
}
