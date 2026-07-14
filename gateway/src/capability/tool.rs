use anyhow::Result;
use serde::{Deserialize, Serialize};
use serde_json::Value;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolMetadata {
    pub name: &'static str,
    pub description: &'static str,
    pub parameters: Value,
}

pub trait Tool: Send + Sync {
    fn metadata(&self) -> ToolMetadata;

    fn execute(&self, input: Value) -> Result<Value>;
}