use anyhow::Result;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use tracing::{debug, instrument};

use crate::models::ColumnMetadata;

// ─── Claude API types ─────────────────────────────────────────────────────────

#[derive(Serialize)]
struct ClaudeRequest {
    model: String,
    max_tokens: u32,
    messages: Vec<ClaudeMessage>,
}

#[derive(Serialize)]
struct ClaudeMessage {
    role: String,
    content: String,
}

#[derive(Deserialize)]
struct ClaudeResponse {
    content: Vec<ClaudeContent>,
}

#[derive(Deserialize)]
struct ClaudeContent {
    text: String,
}

// ─── Output types ─────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PiiClassification {
    pub is_pii: bool,
    pub pii_type: Option<String>, // EMAIL | SSN | PHONE | NAME | ADDRESS | DOB | FINANCIAL
    pub confidence: f32,          // 0.0 - 1.0
    pub reasoning: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ChangeAnalysis {
    pub severity: String,          // INFO | WARNING | CRITICAL
    pub reasoning: String,
    pub remediation: String,
    pub blast_radius_summary: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct EntityResolution {
    pub are_same_entity: bool,
    pub confidence: f32,
    pub reasoning: String,
    pub suggested_join_type: Option<String>, // LEFT | INNER | etc
}

// ─── Client ───────────────────────────────────────────────────────────────────

#[derive(Clone)]
pub struct ClaudeClient {
    client: Client,
    api_key: String,
    model: String,
}

impl ClaudeClient {
    pub fn new(api_key: String) -> Self {
        Self {
            client: Client::new(),
            api_key,
            model: "claude-sonnet-4-6".to_string(),
        }
    }

    /// Classify a single column for PII.
    #[instrument(skip(self, col), fields(column = %col.name))]
    pub async fn classify_pii(&self, col: &ColumnMetadata) -> Result<PiiClassification> {
        let prompt = format!(
            r#"You are a data governance AI. Classify this database column for PII.

Column name: {name}
Data type: {dtype}
Null percentage: {null_pct:.1}%
Distinct values: {distinct}

Respond ONLY with valid JSON, no explanation outside the JSON:
{{
  "is_pii": true/false,
  "pii_type": "EMAIL" | "SSN" | "PHONE" | "NAME" | "ADDRESS" | "DOB" | "FINANCIAL" | null,
  "confidence": 0.0-1.0,
  "reasoning": "one sentence"
}}"#,
            name = col.name,
            dtype = col.data_type,
            null_pct = col.null_pct.unwrap_or(0.0),
            distinct = col.distinct_count.unwrap_or(0),
        );

        let raw = self.call(&prompt, 256).await?;
        debug!(raw = %raw, "PII classification response");
        let result = serde_json::from_str::<PiiClassification>(&raw)
            .map_err(|e| anyhow::anyhow!("Failed to parse PII response: {e}\nRaw: {raw}"))?;
        Ok(result)
    }

    /// Classify a batch of columns — more efficient for seed/scan jobs.
    #[instrument(skip(self, cols), fields(count = cols.len()))]
    pub async fn classify_pii_batch(
        &self,
        cols: &[ColumnMetadata],
    ) -> Result<Vec<PiiClassification>> {
        let items: Vec<String> = cols
            .iter()
            .enumerate()
            .map(|(i, c)| {
                format!(
                    r#"{{"index": {i}, "name": "{}", "type": "{}", "null_pct": {:.1}, "distinct": {}}}"#,
                    c.name,
                    c.data_type,
                    c.null_pct.unwrap_or(0.0),
                    c.distinct_count.unwrap_or(0),
                )
            })
            .collect();

        let prompt = format!(
            r#"You are a data governance AI. Classify each column for PII.

Columns:
[{}]

Respond ONLY with a JSON array in the same order, no explanation outside JSON:
[
  {{"index": 0, "is_pii": true/false, "pii_type": "EMAIL"|"SSN"|"PHONE"|"NAME"|"ADDRESS"|"DOB"|"FINANCIAL"|null, "confidence": 0.0-1.0, "reasoning": "one sentence"}},
  ...
]"#,
            items.join(",\n")
        );

        let max_tokens = 100 * cols.len() as u32;
        let raw = self.call(&prompt, max_tokens.min(4096)).await?;
        debug!(raw = %raw, "Batch PII classification response");

        #[derive(Deserialize)]
        struct BatchItem {
            index: usize,
            is_pii: bool,
            pii_type: Option<String>,
            confidence: f32,
            reasoning: String,
        }

        let batch: Vec<BatchItem> = serde_json::from_str(&raw)
            .map_err(|e| anyhow::anyhow!("Failed to parse batch PII response: {e}\nRaw: {raw}"))?;

        let mut results = vec![
            PiiClassification {
                is_pii: false,
                pii_type: None,
                confidence: 0.0,
                reasoning: String::new(),
            };
            cols.len()
        ];

        for item in batch {
            if item.index < results.len() {
                results[item.index] = PiiClassification {
                    is_pii: item.is_pii,
                    pii_type: item.pii_type,
                    confidence: item.confidence,
                    reasoning: item.reasoning,
                };
            }
        }

        Ok(results)
    }

    /// Analyze a schema change and return severity + remediation.
    #[instrument(skip(self))]
    pub async fn analyze_schema_change(
        &self,
        change_description: &str,
        table_name: &str,
        affected_models: &[String],
        affected_metrics: &[String],
    ) -> Result<ChangeAnalysis> {
        let prompt = format!(
            r#"You are a data pipeline reliability AI. Analyze this schema change.

Table: {table}
Change: {change}
Affected dbt models: {models}
Affected MetricFlow metrics: {metrics}

Respond ONLY with valid JSON:
{{
  "severity": "INFO" | "WARNING" | "CRITICAL",
  "reasoning": "why this severity level (1-2 sentences)",
  "remediation": "what the data team should do (1-2 sentences)",
  "blast_radius_summary": "plain english summary of downstream impact (1 sentence)"
}}"#,
            table = table_name,
            change = change_description,
            models = affected_models.join(", "),
            metrics = affected_metrics.join(", "),
        );

        let raw = self.call(&prompt, 512).await?;
        debug!(raw = %raw, "Change analysis response");
        let result = serde_json::from_str::<ChangeAnalysis>(&raw)
            .map_err(|e| anyhow::anyhow!("Failed to parse change analysis: {e}\nRaw: {raw}"))?;
        Ok(result)
    }

    /// Resolve whether two columns from different sources represent the same entity.
    #[instrument(skip(self))]
    pub async fn resolve_entity(
        &self,
        src_table: &str,
        src_column: &str,
        src_type: &str,
        src_distinct: i64,
        tgt_table: &str,
        tgt_column: &str,
        tgt_type: &str,
        tgt_distinct: i64,
    ) -> Result<EntityResolution> {
        let prompt = format!(
            r#"You are a data modeling AI. Determine if two columns from different source systems represent the same entity.

Column A: {src_table}.{src_col} (type: {src_type}, ~{src_distinct} distinct values)
Column B: {tgt_table}.{tgt_col} (type: {tgt_type}, ~{tgt_distinct} distinct values)

Respond ONLY with valid JSON:
{{
  "are_same_entity": true/false,
  "confidence": 0.0-1.0,
  "reasoning": "one sentence",
  "suggested_join_type": "LEFT" | "INNER" | null
}}"#,
            src_table = src_table,
            src_col = src_column,
            src_type = src_type,
            src_distinct = src_distinct,
            tgt_table = tgt_table,
            tgt_col = tgt_column,
            tgt_type = tgt_type,
            tgt_distinct = tgt_distinct,
        );

        let raw = self.call(&prompt, 256).await?;
        let result = serde_json::from_str::<EntityResolution>(&raw)
            .map_err(|e| anyhow::anyhow!("Failed to parse entity resolution: {e}\nRaw: {raw}"))?;
        Ok(result)
    }

    // ─── Internal ─────────────────────────────────────────────────────────────

    async fn call(&self, prompt: &str, max_tokens: u32) -> Result<String> {
        let request = ClaudeRequest {
            model: self.model.clone(),
            max_tokens,
            messages: vec![ClaudeMessage {
                role: "user".to_string(),
                content: prompt.to_string(),
            }],
        };

        let response = self
            .client
            .post("https://api.anthropic.com/v1/messages")
            .header("x-api-key", &self.api_key)
            .header("anthropic-version", "2023-06-01")
            .header("content-type", "application/json")
            .json(&request)
            .send()
            .await
            .map_err(|e| anyhow::anyhow!("HTTP error calling Claude: {e}"))?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            return Err(anyhow::anyhow!("Claude API returned {status}: {body}"));
        }

        let resp = response
            .json::<ClaudeResponse>()
            .await
            .map_err(|e| anyhow::anyhow!("Failed to deserialize Claude response: {e}"))?;

        resp.content
            .into_iter()
            .next()
            .map(|c| c.text)
            .ok_or_else(|| anyhow::anyhow!("Empty content from Claude"))
    }
}
