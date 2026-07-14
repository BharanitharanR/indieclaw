use anyhow::{Context, Result};
use serde_json::{json, Value};
use std::fs;
use walkdir::WalkDir;

use super::tool::{Tool, ToolMetadata};

pub struct SearchTextTool;

impl Tool for SearchTextTool {
    fn metadata(&self) -> ToolMetadata {
        ToolMetadata {
            name: "search_text",
            description: "Searches text recursively inside a directory.",
            parameters: json!({
                "type":"object",
                "properties":{
                    "root":{
                        "type":"string",
                        "description":"Root directory"
                    },
                    "query":{
                        "type":"string",
                        "description":"Text to search"
                    }
                },
                "required":["root","query"]
            }),
        }
    }

    fn execute(&self, input: Value) -> Result<Value> {
        let root = input["root"]
            .as_str()
            .context("Missing root")?;

        let query = input["query"]
            .as_str()
            .context("Missing query")?;

        let mut matches = Vec::new();

        for entry in WalkDir::new(root)
            .into_iter()
            .filter_map(Result::ok)
        {
            if !entry.file_type().is_file() {
                continue;
            }

            if let Ok(content) = fs::read_to_string(entry.path()) {
                for (line_no, line) in content.lines().enumerate() {
                    if line.contains(query) {
                        matches.push(json!({
                            "file": entry.path().display().to_string(),
                            "line": line_no + 1,
                            "text": line
                        }));
                    }
                }
            }
        }

        Ok(json!({
            "matches": matches
        }))
    }
}