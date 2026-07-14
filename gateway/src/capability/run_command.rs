use anyhow::{Context, Result};
use serde_json::{json, Value};
use std::process::Command;

use super::tool::{Tool, ToolMetadata};

pub struct RunCommandTool;

impl Tool for RunCommandTool {
    fn metadata(&self) -> ToolMetadata {
        ToolMetadata {
            name: "run_command",
            description: "Runs a whitelisted shell command.",
            parameters: json!({
                "type":"object",
                "properties":{
                    "command":{
                        "type":"string",
                        "description":"Command to execute"
                    }
                },
                "required":["command"]
            }),
        }
    }

    fn execute(&self, input: Value) -> Result<Value> {
        let command = input["command"]
            .as_str()
            .context("Missing command")?;

        let allowed = [
            "cargo build",
            "cargo test",
            "cargo fmt",
            "cargo clippy",
            "git status",
            "git diff",
        ];

        if !allowed.contains(&command) {
            anyhow::bail!("Command not allowed");
        }

        let parts: Vec<&str> = command.split_whitespace().collect();

        let output = Command::new(parts[0])
            .args(&parts[1..])
            .output()?;

        Ok(json!({
            "exit_code": output.status.code(),
            "stdout": String::from_utf8_lossy(&output.stdout),
            "stderr": String::from_utf8_lossy(&output.stderr)
        }))
    }
}