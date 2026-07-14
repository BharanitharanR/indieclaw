use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use std::time::Duration;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatMessage {
    pub role: String,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<String>>,
}

#[derive(Serialize)]
struct ChatRequest {
    messages: Vec<ChatMessage>,
    stream: bool,
}

#[derive(Deserialize)]
struct ChatResponse {
    message: MessageContent,
}

#[derive(Deserialize)]
struct MessageContent {
    content: String,
}

#[derive(Clone)]
pub struct GatewayClient {
    base_url: String,
    chat_endpoint: String,
    http_client: reqwest::Client,
}

impl GatewayClient {
    pub fn new(base_url: impl Into<String>, chat_endpoint: impl Into<String>, timeout_secs: u64) -> Result<Self> {
        let http_client = reqwest::Client::builder()
            .timeout(Duration::from_secs(timeout_secs))
            .build()?;

        Ok(Self {
            base_url: base_url.into(),
            chat_endpoint: chat_endpoint.into(),
            http_client,
        })
    }

    pub async fn chat(&self, messages: Vec<ChatMessage>) -> Result<String> {
        let payload = ChatRequest {
            messages,
            stream: false,
        };

        let target_url = format!("{}{}", self.base_url, self.chat_endpoint);

        let response = self
            .http_client
            .post(&target_url)
            .json(&payload)
            .send()
            .await?;

        if !response.status().is_success() {
            return Err(anyhow!("Gateway returned HTTP status: {}", response.status()));
        }

        let body: ChatResponse = response.json().await?;
        Ok(body.message.content)
    }
}