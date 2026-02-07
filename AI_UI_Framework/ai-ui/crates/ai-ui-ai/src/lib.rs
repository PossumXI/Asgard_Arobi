pub mod claude;
pub mod ollama;
pub mod mcp;
pub mod streaming;

use thiserror::Error;

#[derive(Debug, Error)]
pub enum AiError {
    #[error("API key not configured")]
    NoApiKey,

    #[error("Rate limited — please wait")]
    RateLimited,

    #[error("Network error: {0}")]
    Network(#[from] reqwest::Error),

    #[error("API error: {status} — {message}")]
    ApiError { status: u16, message: String },

    #[error("No AI backend available")]
    NoBackend,
}

/// Unified AI backend — Claude first, Ollama fallback
pub async fn generate_response(
    prompt: &str,
    claude_key: Option<&str>,
) -> Result<String, AiError> {
    // Try Claude first
    if let Some(key) = claude_key {
        let client = claude::ClaudeClient::new(key.to_string());
        match client
            .send(vec![claude::Message {
                role: "user".into(),
                content: serde_json::Value::String(prompt.into()),
            }])
            .await
        {
            Ok(resp) => {
                for block in &resp.content {
                    if let claude::ContentBlock::Text { text } = block {
                        return Ok(text.clone());
                    }
                }
            }
            Err(e) => tracing::warn!("Claude failed: {}, trying Ollama", e),
        }
    }

    // Fallback to Ollama
    if ollama::is_ollama_running().await {
        return ollama::generate(prompt).await;
    }

    Err(AiError::NoBackend)
}

/// Load API key from environment, OS keyring, or config file
pub fn load_api_key() -> Option<String> {
    // Priority: 1) env var, 2) OS keyring, 3) config file
    if let Ok(key) = std::env::var("ANTHROPIC_API_KEY") {
        return Some(key);
    }

    if let Ok(entry) = keyring::Entry::new("ai-ui", "anthropic-api-key") {
        if let Ok(key) = entry.get_password() {
            return Some(key);
        }
    }

    // Fall back to ~/.config/ai-ui/config.toml
    let config_dir = dirs::config_dir()?.join("ai-ui");
    let config: toml::Value =
        toml::from_str(&std::fs::read_to_string(config_dir.join("config.toml")).ok()?).ok()?;
    config["api"]["anthropic_key"].as_str().map(String::from)
}

/// Store API key in OS keyring
pub fn store_api_key(key: &str) -> Result<(), keyring::Error> {
    let entry = keyring::Entry::new("ai-ui", "anthropic-api-key")?;
    entry.set_password(key)?;
    Ok(())
}
