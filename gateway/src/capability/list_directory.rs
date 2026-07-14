use anyhow::{Context, Result};
use serde_json::{json, Value};
use std::fs;

use super::tool::{Tool, ToolMetadata};

pub struct ListDirectoryTool;

impl Tool for ListDirectoryTool {
    fn metadata(&self) -> ToolMetadata {
        ToolMetadata {
            name: "list_directory",
            description: "Lists files and folders inside a directory.",
            parameters: json!({
                "type":"object",
                "properties":{
                    "path":{
                        "type":"string",
                        "description":"Directory path"
                    }
                },
                "required":["path"]
            }),
        }
    }

    fn execute(&self, input: Value) -> Result<Value> {
        let path = input["path"]
            .as_str()
            .context("Missing 'path'")?;

        let mut entries = Vec::new();

        for entry in fs::read_dir(path)? {
            let entry = entry?;

            entries.push(json!({
                "name": entry.file_name().to_string_lossy(),
                "is_dir": entry.path().is_dir()
            }));
        }

        Ok(json!({
            "path": path,
            "entries": entries
        }))
    }
}