use crate::AiError;
use ollama_rs::generation::completion::request::GenerationRequest;
use ollama_rs::Ollama;

/// Check if Ollama is running locally
pub async fn is_ollama_running() -> bool {
    reqwest::get("http://localhost:11434/api/tags")
        .await
        .map(|r| r.status().is_success())
        .unwrap_or(false)
}

/// Generate a response using Ollama
pub async fn generate(prompt: &str) -> Result<String, AiError> {
    let ollama = Ollama::default();
    let res = ollama
        .generate(GenerationRequest::new(
            "llama3.2:latest".to_string(),
            prompt,
        ))
        .await
        .map_err(|e| AiError::ApiError {
            status: 500,
            message: e.to_string(),
        })?;

    Ok(res.response)
}

/// List available Ollama models
pub async fn list_models() -> Result<Vec<String>, AiError> {
    let ollama = Ollama::default();
    let models = ollama
        .list_local_models()
        .await
        .map_err(|e| AiError::ApiError {
            status: 500,
            message: e.to_string(),
        })?;

    Ok(models.into_iter().map(|m| m.name).collect())
}
