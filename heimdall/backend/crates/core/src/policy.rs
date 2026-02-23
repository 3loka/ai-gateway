use serde::{Deserialize, Serialize};

use crate::models::{ColumnMetadata, ExtractionDecision, PolicyDecision};

/// Shape of a Heimdall policy YAML file.
///
/// Example:
/// ```yaml
/// name: default-pii-policy
/// pii_gate:
///   blocked_types: [SSN, FINANCIAL]
///   requires_approval: [EMAIL, PHONE]
///   mask_types: [NAME, ADDRESS]
/// cost_guardrail:
///   max_monthly_mar: 10_000_000
/// freshness_sla:
///   warn_after_hours: 6
///   error_after_hours: 24
/// ```
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyDefinition {
    pub name: String,
    pub pii_gate: Option<PiiGatePolicy>,
    pub cost_guardrail: Option<CostGuardrailPolicy>,
    pub freshness_sla: Option<FreshnessSlaPolicy>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PiiGatePolicy {
    /// These PII types are completely blocked — DENY
    #[serde(default)]
    pub blocked_types: Vec<String>,
    /// These PII types require manual approval before extraction
    #[serde(default)]
    pub requires_approval: Vec<String>,
    /// These PII types are extracted with masking — PARTIAL
    #[serde(default)]
    pub mask_types: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CostGuardrailPolicy {
    pub max_monthly_mar: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FreshnessSlaPolicy {
    pub warn_after_hours: Option<i64>,
    pub error_after_hours: Option<i64>,
}

/// Evaluate a policy against the columns of a table being requested for extraction.
pub fn evaluate(policy: &PolicyDefinition, columns: &[ColumnMetadata]) -> PolicyDecision {
    let mut blocked_columns: Vec<String> = Vec::new();
    let mut masked_columns: Vec<String> = Vec::new();

    if let Some(pii_gate) = &policy.pii_gate {
        for col in columns {
            if let (Some(true), Some(pii_type)) = (col.is_pii, &col.pii_type) {
                if pii_gate.blocked_types.contains(pii_type) {
                    blocked_columns.push(col.name.clone());
                } else if pii_gate.mask_types.contains(pii_type) {
                    masked_columns.push(col.name.clone());
                }
            }
        }
    }

    if !blocked_columns.is_empty() {
        return PolicyDecision {
            decision: ExtractionDecision::Deny,
            reason: format!(
                "Blocked PII columns detected: {}. Define a masking policy or exclude these columns.",
                blocked_columns.join(", ")
            ),
            policy_name: Some(policy.name.clone()),
            blocked_columns,
        };
    }

    if !masked_columns.is_empty() {
        return PolicyDecision {
            decision: ExtractionDecision::Partial,
            reason: format!(
                "Extracting with masking applied to: {}",
                masked_columns.join(", ")
            ),
            policy_name: Some(policy.name.clone()),
            blocked_columns: masked_columns,
        };
    }

    PolicyDecision {
        decision: ExtractionDecision::Approve,
        reason: "All policy checks passed.".to_string(),
        policy_name: Some(policy.name.clone()),
        blocked_columns: vec![],
    }
}

pub fn parse_yaml(yaml: &str) -> Result<PolicyDefinition, serde_yaml::Error> {
    serde_yaml::from_str(yaml)
}
