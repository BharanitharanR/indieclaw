mod capability;
mod chat;
use capability::manager::ToolManager;
use capability::read_file::ReadFileTool;
use capability::list_directory::ListDirectoryTool;
use capability::search_text::SearchTextTool;
use capability::run_command::RunCommandTool;
use capability::tool::ToolMetadata;
use serde_json::Value;
use crate::chat::models::*;
use chat::models::{ChatRequest, ChatResponse};
use axum::{
    body::Body,
    extract::State,
    http::{HeaderMap, StatusCode},
    response::{Json, Response},
    routing::{get, post},
    Router,
};
use futures_util::StreamExt;
use reqwest::Client;
use std::sync::Arc;
use tokio::sync::Semaphore;
use tower_http::cors::CorsLayer;

struct AppState {
    http_client: Client,
    queue_permit: Arc<Semaphore>,
    tool_manager: Arc<ToolManager>,
}

#[tokio::main]
async fn main() {
    let mut tool_manager = ToolManager::new();
    tool_manager.register(ReadFileTool);
    tool_manager.register(ListDirectoryTool);
    tool_manager.register(SearchTextTool);
    tool_manager.register(RunCommandTool);

    let state = Arc::new(AppState {
        http_client: Client::new(),
        queue_permit: Arc::new(Semaphore::new(1)),
        tool_manager: Arc::new(tool_manager),
    });

    let app = Router::new()
        .route("/api/v1/chat", post(proxy_chat))
        .route("/api/v1/generate", post(proxy_generate))
        .route("/tools", get(list_tools))
        .route("/tool/execute", post(execute_tool))
        .layer(CorsLayer::permissive())
        .with_state(state);

    let listener = tokio::net::TcpListener::bind("127.0.0.1:8080")
        .await
        .unwrap();

    println!("Rust Gateway running on http://127.0.0.1:8080");

    axum::serve(listener, app).await.unwrap();
}

async fn proxy_chat(
    State(state): State<Arc<AppState>>,
    _headers: HeaderMap,
    body: Body,
) -> Result<Response, StatusCode> {
    let bytes = axum::body::to_bytes(body, usize::MAX)
        .await
        .map_err(|_| StatusCode::BAD_REQUEST)?;

    let mut request: ChatRequest = serde_json::from_slice(&bytes)
        .map_err(|_| StatusCode::BAD_REQUEST)?;

    request.tools = Some(state.tool_manager.ollama_tools());

    let response = ask_ollama(&state, &request).await.map_err(|e| {
        eprintln!("{}", e);
        StatusCode::BAD_GATEWAY
    })?;

    if !response.message.tool_calls.is_empty() {
        request.messages.push(response.message.clone());

        for tool_call in &response.message.tool_calls {
            let tool_name = &tool_call.function.name;

            println!("Executing tool: {}", tool_name);

            let result = state
                .tool_manager
                .execute(tool_name, tool_call.function.arguments.clone())
                .map_err(|e| {
                    eprintln!("{}", e);
                    StatusCode::INTERNAL_SERVER_ERROR
                })?;

            println!("Tool Result:\n{}", result);

            request.messages.push(ChatMessage {
                role: "tool".into(),
                content: result.to_string(),
                images: None,
                thinking: None,
                tool_calls: vec![],
                tool_call_id: Some(tool_call.id.clone()),
                name: Some(tool_call.function.name.clone()),
            });
        }

        let final_response = ask_ollama(&state, &request)
            .await
            .map_err(|_| StatusCode::BAD_GATEWAY)?;

        return Ok(Response::builder()
            .status(StatusCode::OK)
            .header("content-type", "application/json")
            .body(Body::from(serde_json::to_vec(&final_response).unwrap()))
            .unwrap());
    } else {
        return Ok(Response::builder()
            .status(StatusCode::OK)
            .header("content-type", "application/json")
            .body(Body::from(serde_json::to_vec(&response).unwrap()))
            .unwrap());
    }
}

async fn proxy_generate(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    body: Body,
) -> Result<Response, StatusCode> {
    proxy(
        state,
        headers,
        body,
        "http://127.0.0.1:11434/api/generate",
    )
    .await
}

async fn list_tools(
    State(state): State<Arc<AppState>>,
) -> Json<Vec<ToolMetadata>> {
    Json(state.tool_manager.metadata())
}

async fn execute_tool(
    State(state): State<Arc<AppState>>,
    Json(request): Json<Value>,
) -> Result<Json<Value>, StatusCode> {
    let tool_name = request["tool"].as_str().ok_or(StatusCode::BAD_REQUEST)?;
    let args = request["arguments"].clone();

    let result = state
        .tool_manager
        .execute(tool_name, args)
        .map_err(|_| StatusCode::BAD_REQUEST)?;

    Ok(Json(result))
}

async fn ask_ollama(
    state: &Arc<AppState>,
    request: &ChatRequest,
) -> anyhow::Result<ChatResponse> {
    let response = state
        .http_client
        .post("http://127.0.0.1:11434/api/chat")
        .json(request)
        .send()
        .await?;

    if !response.status().is_success() {
        anyhow::bail!("Ollama returned {}", response.status());
    }

    let chat = response.json::<ChatResponse>().await?;
    Ok(chat)
}

async fn proxy(
    state: Arc<AppState>,
    headers: HeaderMap,
    body: Body,
    url: &'static str,
) -> Result<Response, StatusCode> {
    let _permit = state
        .queue_permit
        .acquire()
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    let mut request = state.http_client.post(url);

    if let Some(content_type) = headers.get("content-type") {
        if let Ok(value) = content_type.to_str() {
            request = request.header("content-type", value);
        }
    }

    let reqwest_body = reqwest::Body::wrap_stream(body.into_data_stream());

    let response = request
        .body(reqwest_body)
        .send()
        .await
        .map_err(|e| {
            eprintln!("Gateway error: {}", e);
            StatusCode::BAD_GATEWAY
        })?;

    let status = StatusCode::from_u16(response.status().as_u16())
        .unwrap_or(StatusCode::INTERNAL_SERVER_ERROR);

    let stream = response
        .bytes_stream()
        .map(|item| item.map_err(std::io::Error::other));

    Ok(Response::builder()
        .status(status)
        .header("content-type", "application/json")
        .body(Body::from_stream(stream))
        .unwrap())
}
