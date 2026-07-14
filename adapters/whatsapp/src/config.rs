use anyhow::{Context, Result};
use serde::Deserialize;
use std::{fs, path::Path};

#[derive(Debug, Deserialize, Clone)]
pub struct BotConfig {
    pub allowed_jid: String,
    pub trigger_prefix: String,
    pub default_image_prompt: String,
    pub empty_prompt_reply: String,
    pub error_reply: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct GatewayConfig {
    pub base_url: String,
    pub chat_endpoint: String,
    pub request_timeout_seconds: u64,
}

#[derive(Debug, Deserialize, Clone)]
pub struct AppConfig {
    pub bot: BotConfig,
    pub gateway: GatewayConfig,
}

impl AppConfig {
    pub fn load<P: AsRef<Path>>(path: P) -> Result<Self> {
        let content = fs::read_to_string(path)
            .context("Failed to read config file")?;
        let config: AppConfig = toml::from_str(&content)
            .context("Failed to parse config file")?;
        Ok(config)
    }
}