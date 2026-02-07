use crate::claude::ClaudeClient;
use crate::AiError;

/// Collect a complete streaming response into a single string
pub async fn stream_response(api_key: &str, prompt: &str) -> Result<String, AiError> {
    let client = ClaudeClient::new(api_key.to_string());
    let mut full_response = String::new();

    client
        .stream(prompt, |chunk| {
            full_response.push_str(&chunk);
        })
        .await?;

    Ok(full_response)
}
