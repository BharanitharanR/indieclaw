#!/bin/bash
# Check RabbitMQ response queue using rabbitmq-admin

RESPONSE=$(curl -s -u indieclaw:secretpass \
  "http://localhost:15672/api/queues/%2F/orchestrator.responses" \
  -H "Accept: application/json")

echo "📊 Response Queue Status:"
echo "$RESPONSE" | jq '{
  name: .name,
  messages: .messages,
  messages_ready: .messages_ready,
  messages_unacked: .messages_unacked,
  durable: .durable
}' 2>/dev/null || echo "$RESPONSE"
