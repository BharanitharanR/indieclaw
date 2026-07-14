use std::collections::HashMap;

use anyhow::{anyhow, Result};
use serde_json::{json, Value};

use super::tool::{Tool, ToolMetadata};

pub struct ToolManager {
    tools: HashMap<String, Box<dyn Tool>>,
}

impl ToolManager {
    pub fn new() -> Self {
        Self {
            tools: HashMap::new(),
        }
    }

    pub fn register<T: Tool + 'static>(&mut self, tool: T) {
        self.tools
            .insert(tool.metadata().name.to_string(), Box::new(tool));
    }

    pub fn has_tool(&self, tool_name: &str) -> bool {
        self.tools.contains_key(tool_name)
    }

    pub fn execute(
        &self,
        tool_name: &str,
        input: Value,
    ) -> Result<Value> {
        let tool = self
            .tools
            .get(tool_name)
            .ok_or_else(|| anyhow!("Unknown tool '{}'", tool_name))?;

        tool.execute(input)
    }

    pub fn ollama_tools(&self) -> Value {
        let tools: Vec<Value> = self
            .tools
            .values()
            .map(|tool| {
                let metadata = tool.metadata();

                json!({
                    "type": "function",
                    "function": {
                        "name": metadata.name,
                        "description": metadata.description,
                        "parameters": metadata.parameters
                    }
                })
            })
            .collect();

        Value::Array(tools)
    }

    pub fn metadata(&self) -> Vec<ToolMetadata> {
        self.tools
            .values()
            .map(|tool| tool.metadata())
            .collect()
    }
}