use serde::{Deserialize, Serialize};
use serde_json::Value;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatRequest {
    pub model: String,

    pub messages: Vec<ChatMessage>,

    #[serde(default)]
    pub stream: bool,

    #[serde(default)]
    pub tools: Option<Value>,

    #[serde(default)]
    pub options: Option<Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatResponse {
    pub model: String,

    #[serde(default)]
    pub created_at: Option<String>,

    pub message: ChatMessage,

    #[serde(default)]
    pub done: bool,

    #[serde(default)]
    pub done_reason: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatMessage {
    pub role: String,

    #[serde(default)]
    pub content: String,

    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<String>>,

    #[serde(default)]
    pub thinking: Option<String>,

    #[serde(default)]
    pub tool_calls: Vec<ToolCall>,

    #[serde(default)]
    pub tool_call_id: Option<String>,

    #[serde(default)]
    pub name: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolCall {
    pub id: String,
    pub function: ToolFunction,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolFunction {
    pub name: String,
    pub arguments: Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolResultMessage;
