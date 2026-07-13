use std::sync::Arc;
use anyhow::{anyhow, Result};
use axum::{
    body::{to_bytes, Body},
    extract::State,
    http::{HeaderMap, StatusCode},
    response::Response,
};
use serde_json::{json, Value};
use crate::AppState;
use super::models::{
    ChatMessage,
    ChatRequest,
    ChatResponse,
    ToolCall,
    ToolFunction,
    ToolResultMessage,
};

const OLLAMA_CHAT_URL: &str = "http://127.0.0.1:11434/api/chat";

fn build_tool_message(
    tool_name: &str,
    result: Value,
) -> ChatMessage {
    ChatMessage {
        role: "tool".to_string(),
        content: result.to_string(),
        images: None,
        thinking: None,
        tool_calls: None,
        tool_name: Some(tool_name.to_string()),
    }
}

fn build_assistant_tool_call(
    tool_calls: Vec<ToolCall>,
) -> ChatMessage {
    ChatMessage {
        role: "assistant".to_string(),
        content: String::new(),
        images: None,
        thinking: None,
        tool_calls: Some(tool_calls),
        tool_name: None,
    }
}

fn inject_tools(
    mut request: Value,
    state: &Arc<AppState>,
) -> Value {
    request["tools"] = state.tool_manager.ollama_tools();
    request
}

fn parse_request(bytes: &[u8]) -> Result<Value> {
    let value: Value = serde_json::from_slice(bytes)?;
    Ok(value)
}

fn serialize_request(request: &Value) -> Result<String> {
    Ok(serde_json::to_string(request)?)
}

async fn call_ollama(
    state: &Arc<AppState>,
    request: &Value,
) -> Result<ChatResponse> {
    let response = state
        .http_client
        .post(OLLAMA_CHAT_URL)
        .header("Content-Type", "application/json")
        .json(request)
        .send()
        .await?;

    if !response.status().is_success() {
        return Err(anyhow!("Ollama returned {}", response.status()));
    }

    let chat_response = response.json::<ChatResponse>().await?;
    Ok(chat_response)
}

fn extract_tool_calls(
    response: &ChatResponse,
) -> Vec<ToolCall> {
    response
        .message
        .tool_calls
        .clone()
        .unwrap_or_default()
}

async fn execute_tool(
    state: &Arc<AppState>,
    tool_call: &ToolCall,
) -> Result<ChatMessage> {
    let tool_name = &tool_call.function.name;

    if !state.tool_manager.has_tool(tool_name) {
        return Err(anyhow!("Unknown tool '{}'", tool_name));
    }

    let result = state
        .tool_manager
        .execute(
            tool_name,
            tool_call.function.arguments.clone(),
        )?;

    Ok(build_tool_message(tool_name, result))
}

fn append_tool_messages(
    request: &mut ChatRequest,
    assistant: ChatMessage,
    tool: ChatMessage,
) {
    request.messages.push(assistant);
    request.messages.push(tool);
}

fn build_request_json(
    request: &ChatRequest,
    state: &Arc<AppState>,
) -> Result<Value> {
    let mut value = serde_json::to_value(request)?;
    value["tools"] = state.tool_manager.ollama_tools();
    Ok(value)
}

async fn execute_tool_calls(
    state: &Arc<AppState>,
    request: &mut ChatRequest,
    response: &ChatResponse,
) -> Result<bool> {
    // Safely extract tool loops based on Option serialization types
    let calls = match &response.message.tool_calls {
        Some(c) if !c.is_empty() => c,
        _ => return Ok(false),
    };

    // Preserve assistant tool call
    request.messages.push(response.message.clone());

    // Execute every tool call
    for tool_call in calls {
        let tool_name = &tool_call.function.name;
        println!("Executing tool: {}", tool_name);

        if !state.tool_manager.has_tool(tool_name) {
            request.messages.push(ChatMessage {
                role: "tool".to_string(),
                content: format!("Tool '{}' not found.", tool_name),
                images: None,
                thinking: None,
                tool_calls: None,
                tool_name: Some(tool_name.clone()),
            });
            continue;
        }

        let result = state
            .tool_manager
            .execute(
                tool_name,
                tool_call.function.arguments.clone(),
            )?;

        request.messages.push(ChatMessage {
            role: "tool".to_string(),
            content: result.to_string(),
            images: None,
            thinking: None,
            tool_calls: None,
            tool_name: Some(tool_name.clone()),
        });
    }

    Ok(true)
}

async fn ask_ollama(
    state: &Arc<AppState>,
    request: &ChatRequest,
) -> Result<ChatResponse> {
    let mut json = serde_json::to_value(request)?;
    json["tools"] = state.tool_manager.ollama_tools();

    let response = state
        .http_client
        .post(OLLAMA_CHAT_URL)
        .json(&json)
        .send()
        .await?;

    if !response.status().is_success() {
        return Err(anyhow!("Ollama returned {}", response.status()));
    }

    Ok(response.json::<ChatResponse>().await?)
}
