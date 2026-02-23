use sqlx::PgPool;
use std::sync::Arc;

use heimdall_core::ai::ClaudeClient;

pub type SharedState = Arc<AppState>;

pub struct AppState {
    pub db: PgPool,
    pub claude: ClaudeClient,
}

impl AppState {
    pub fn new(db: PgPool, anthropic_api_key: String) -> Self {
        Self {
            db,
            claude: ClaudeClient::new(anthropic_api_key),
        }
    }
}
