mod config;

use anyhow::Result;
use base64::engine::general_purpose::STANDARD as BASE64;
use base64::Engine;
use config::AppConfig;
use gateway_sdk::{ChatMessage, GatewayClient};
use qrcode::render::unicode;
use qrcode::QrCode;
use std::sync::Arc;

// Direct imports from whatsapp-rust
use whatsapp_rust::bot::Bot;
use whatsapp_rust::client::Client;
use whatsapp_rust::store::SqliteStore;
use whatsapp_rust::TokioRuntime;
use whatsapp_rust_tokio_transport::TokioWebSocketTransportFactory;
use whatsapp_rust_ureq_http_client::UreqHttpClient;

// Correct type origins
use wacore::types::events::Event;
use wacore_binary::jid::Jid;
use waproto::whatsapp as wa;

#[tokio::main]
async fn main() -> Result<()> {
    let config_path = std::env::var("CONFIG_PATH").unwrap_or_else(|_| "config.toml".to_string());
    let app_config = Arc::new(AppConfig::load(&config_path)?);

    let ai_client = Arc::new(GatewayClient::new(
        &app_config.gateway.base_url,
        &app_config.gateway.chat_endpoint,
        app_config.gateway.request_timeout_seconds,
    )?);

    let backend = Arc::new(SqliteStore::new(":memory:").await?);

    let config_clone = Arc::clone(&app_config);
    let ai_clone = Arc::clone(&ai_client);

    let mut bot = Bot::builder()
        .with_backend(backend)
        .with_transport_factory(TokioWebSocketTransportFactory::new())
        .with_http_client(UreqHttpClient::new())
        .with_runtime(TokioRuntime)
        .on_event(move |event, client| {
            let cfg = Arc::clone(&config_clone);
            let ai = Arc::clone(&ai_clone);
            let cli = client.clone();

            async move {
                // Dereference Arc<Event> to match on Event variants
                match *event {
                    Event::PairingQrCode { ref code, .. } => {
                        println!("Scan the QR code below:");
                        if let Ok(qr) = QrCode::new(code) {
                            let image = qr
                                .render::<unicode::Dense1x2>()
                                .dark_color(unicode::Dense1x2::Light)
                                .light_color(unicode::Dense1x2::Dark)
                                .build();
                            println!("{}", image);
                        } else {
                            println!("QR Code string: {}", code);
                        }
                    }
                    Event::Message(ref msg, _info) => {
                        let msg_clone = msg.clone();
                        tokio::spawn(async move {
                            if let Err(err) = handle_message(cli, msg_clone, cfg, ai).await {
                                eprintln!("Error processing message: {:?}", err);
                            }
                        });
                    }
                    _ => {}
                }
            }
        })
        .build()
        .await?;

    println!("WhatsApp bot started. Awaiting connection...");
    bot.run().await?;

    Ok(())
}

async fn handle_message(
    client: Arc<Client>,
    msg: wa::Message,
    config: Arc<AppConfig>,
    ai: Arc<GatewayClient>,
) -> Result<()> {
    // Extract recipient/sender context safely
    let sender_jid_str = msg.conversation.clone().unwrap_or_default();

    // Check prefix and trigger conditions
    let raw_text = msg.conversation.as_deref().unwrap_or_default();
    let trimmed = raw_text.trim();

    if !trimmed.starts_with(&config.bot.trigger_prefix) {
        return Ok(());
    }

    println!("Processing message...");

    let prompt_text = trimmed[config.bot.trigger_prefix.len()..].trim().to_string();
    let target_jid: Jid = config.bot.allowed_jid.parse()?;

    if prompt_text.is_empty() {
        send_text_msg(&client, &target_jid, &config.bot.empty_prompt_reply).await?;
        return Ok(());
    }

    let user_msg = ChatMessage {
        role: "user".to_string(),
        content: prompt_text,
        images: None,
    };

    match ai.chat(vec![user_msg]).await {
        Ok(reply) => {
            send_text_msg(&client, &target_jid, &reply).await?;
        }
        Err(err) => {
            eprintln!("AI Gateway Request Error: {:?}", err);
            send_text_msg(&client, &target_jid, &config.bot.error_reply).await?;
        }
    }

    Ok(())
}

async fn send_text_msg(client: &Client, recipient: &Jid, text: &str) -> Result<()> {
    let message = wa::Message {
        conversation: Some(text.to_string()),
        ..Default::default()
    };
    client.send_message(recipient.clone(), message).await?;
    Ok(())
}