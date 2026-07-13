use anyhow::{Context, Result};
use serde_json::{json, Value};
use std::fs;

use super::tool::{Tool, ToolMetadata};

pub struct ReadFileTool;

impl Tool for ReadFileTool {
    fn metadata(&self) -> ToolMetadata {
        ToolMetadata {
            name: "read_file",
            description: "Reads a UTF-8 text file from disk.",
            parameters: json!({
                "type": "object",
                "properties": {
                    "path": {
                        "type": "string",
                        "description": "Path of the file to read"
                    }
                },
                "required": ["path"]
            }),
        }
    }

    fn execute(&self, input: Value) -> Result<Value> {
        let path = input["path"]
            .as_str()
            .context("Missing 'path'")?;

        let content = fs::read_to_string(path)?;

        Ok(json!({
            "path": path,
            "content": content
        }))
    }
}