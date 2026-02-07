use crate::AiError;
use futures_util::StreamExt;
use reqwest::header::{HeaderMap, HeaderValue, CONTENT_TYPE};
use reqwest_eventsource::{Event, EventSource};
use serde::{Deserialize, Serialize};

const API_URL: &str = "https://api.anthropic.com/v1/messages";
const API_VERSION: &str = "2023-06-01";

#[derive(Serialize)]
pub struct MessageRequest {
    pub model: String,
    pub max_tokens: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub system: Option<String>,
    pub messages: Vec<Message>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub stream: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tools: Option<Vec<Tool>>,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct Message {
    pub role: String,
    pub content: serde_json::Value,
}

#[derive(Deserialize, Debug)]
pub struct MessageResponse {
    pub id: String,
    pub content: Vec<ContentBlock>,
    pub stop_reason: Option<String>,
    pub usage: Usage,
}

#[derive(Deserialize, Serialize, Debug, Clone)]
#[serde(tag = "type")]
pub enum ContentBlock {
    #[serde(rename = "text")]
    Text { text: String },
    #[serde(rename = "tool_use")]
    ToolUse {
        id: String,
        name: String,
        input: serde_json::Value,
    },
    #[serde(rename = "tool_result")]
    ToolResult {
        tool_use_id: String,
        content: String,
    },
}

#[derive(Deserialize, Debug)]
pub struct Usage {
    pub input_tokens: u32,
    pub output_tokens: u32,
}

#[derive(Serialize, Clone, Debug)]
pub struct Tool {
    pub name: String,
    pub description: String,
    pub input_schema: serde_json::Value,
}

pub struct ClaudeClient {
    client: reqwest::Client,
    api_key: String,
    model: String,
    system_prompt: Option<String>,
}

impl ClaudeClient {
    pub fn new(api_key: String) -> Self {
        Self {
            client: reqwest::Client::new(),
            api_key,
            model: "claude-sonnet-4-5-20250929".into(),
            system_prompt: Some(
                "You are an AI assistant integrated into a desktop shell. \
                 Help users launch apps, manage files, answer questions, and control their \
                 system. Be concise and actionable."
                    .into(),
            ),
        }
    }

    pub fn with_model(mut self, model: &str) -> Self {
        self.model = model.to_string();
        self
    }

    pub fn with_system_prompt(mut self, prompt: &str) -> Self {
        self.system_prompt = Some(prompt.to_string());
        self
    }

    fn headers(&self) -> HeaderMap {
        let mut h = HeaderMap::new();
        h.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));
        h.insert(
            "x-api-key",
            HeaderValue::from_str(&self.api_key).unwrap(),
        );
        h.insert(
            "anthropic-version",
            HeaderValue::from_static(API_VERSION),
        );
        h
    }

    /// Non-streaming call — returns complete response
    pub async fn send(&self, messages: Vec<Message>) -> Result<MessageResponse, AiError> {
        let request = MessageRequest {
            model: self.model.clone(),
            max_tokens: 4096,
            system: self.system_prompt.clone(),
            messages,
            stream: None,
            tools: None,
        };

        let resp = self
            .client
            .post(API_URL)
            .headers(self.headers())
            .json(&request)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            if status == 429 {
                return Err(AiError::RateLimited);
            }
            let msg = resp.text().await.unwrap_or_default();
            return Err(AiError::ApiError {
                status,
                message: msg,
            });
        }

        Ok(resp.json().await?)
    }

    /// Non-streaming call with tools
    pub async fn send_with_tools(
        &self,
        messages: Vec<Message>,
        tools: Vec<Tool>,
    ) -> Result<MessageResponse, AiError> {
        let request = MessageRequest {
            model: self.model.clone(),
            max_tokens: 4096,
            system: self.system_prompt.clone(),
            messages,
            stream: None,
            tools: Some(tools),
        };

        let resp = self
            .client
            .post(API_URL)
            .headers(self.headers())
            .json(&request)
            .send()
            .await?;

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            if status == 429 {
                return Err(AiError::RateLimited);
            }
            let msg = resp.text().await.unwrap_or_default();
            return Err(AiError::ApiError {
                status,
                message: msg,
            });
        }

        Ok(resp.json().await?)
    }

    /// Streaming call — yields text chunks via callback
    pub async fn stream(
        &self,
        prompt: &str,
        mut on_chunk: impl FnMut(String),
    ) -> Result<(), AiError> {
        let body = serde_json::json!({
            "model": self.model,
            "max_tokens": 4096,
            "stream": true,
            "system": self.system_prompt,
            "messages": [{"role": "user", "content": prompt}]
        });

        let request = self
            .client
            .post(API_URL)
            .headers(self.headers())
            .json(&body);

        let mut es = EventSource::new(request).map_err(|e| AiError::ApiError {
            status: 0,
            message: e.to_string(),
        })?;

        while let Some(event) = es.next().await {
            match event {
                Ok(Event::Message(msg)) => match msg.event.as_str() {
                    "content_block_delta" => {
                        if let Ok(data) = serde_json::from_str::<serde_json::Value>(&msg.data) {
                            if let Some(text) = data["delta"]["text"].as_str() {
                                on_chunk(text.to_string());
                            }
                        }
                    }
                    "message_stop" => {
                        es.close();
                        break;
                    }
                    _ => {}
                },
                Err(_) => {
                    es.close();
                    break;
                }
                _ => {}
            }
        }

        Ok(())
    }
}

/// Define tools Claude can call to control the desktop
pub fn desktop_tools() -> Vec<Tool> {
    vec![
        Tool {
            name: "launch_app".into(),
            description: "Launch an installed application by name".into(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {
                    "app_name": { "type": "string", "description": "Application name" }
                },
                "required": ["app_name"]
            }),
        },
        Tool {
            name: "system_command".into(),
            description: "Execute a system action (volume, brightness, wifi toggle)".into(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {
                    "action": { "type": "string", "enum": [
                        "volume_up", "volume_down", "mute",
                        "brightness_up", "brightness_down",
                        "wifi_toggle", "bluetooth_toggle"
                    ]}
                },
                "required": ["action"]
            }),
        },
    ]
}

/// Handle tool use responses from Claude
/// The caller (shell crate) should provide a handler function
/// since the AI crate doesn't depend on the system crate.
pub async fn handle_tool_call(
    name: &str,
    input: &serde_json::Value,
    handler: impl Fn(&str, &serde_json::Value) -> std::pin::Pin<Box<dyn std::future::Future<Output = String> + Send>> + Send,
) -> String {
    handler(name, input).await
}
