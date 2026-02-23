use thiserror::Error;

#[derive(Debug, Error)]
pub enum HeimdallError {
    #[error("Database error: {0}")]
    Database(#[from] sqlx::Error),

    #[error("Claude API error: {0}")]
    ClaudeApi(String),

    #[error("Fivetran API error: {0}")]
    FivetranApi(String),

    #[error("Policy violation: {0}")]
    PolicyViolation(String),

    #[error("Not found: {0}")]
    NotFound(String),

    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error(transparent)]
    Other(#[from] anyhow::Error),
}

pub type Result<T> = std::result::Result<T, HeimdallError>;
