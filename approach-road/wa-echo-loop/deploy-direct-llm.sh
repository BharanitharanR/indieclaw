#!/bin/bash

# Direct LLM Integration Deployment Script
# Removes orchestrator, deploys direct Ollama integration

set -e

echo "🚀 Deploying Direct LLM Integration..."
echo ""

# Step 1: Backup
echo "📦 Step 1: Backing up original app.js..."
if [ -f app.js ]; then
    cp app.js app.js.backup
    echo "   ✅ Backup created: app.js.backup"
else
    echo "   ⚠️  app.js not found!"
    exit 1
fi

# Step 2: Deploy
echo ""
echo "📥 Step 2: Deploying new version..."
if [ -f app-v2.js ]; then
    mv app-v2.js app.js
    echo "   ✅ New app.js deployed"
else
    echo "   ⚠️  app-v2.js not found!"
    exit 1
fi

# Step 3: Check dependencies
echo ""
echo "📚 Step 3: Checking dependencies..."
if [ ! -d node_modules ]; then
    echo "   Installing npm packages..."
    npm install
fi
echo "   ✅ Dependencies ready"

# Step 4: Verify Ollama
echo ""
echo "🤖 Step 4: Checking Ollama connection..."
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "   ✅ Ollama is running"

    # Check model availability
    if curl -s http://localhost:11434/api/tags | grep -q "qwen3:8b-16k"; then
        echo "   ✅ Model qwen3:8b-16k is available"
    else
        echo "   ⚠️  Model qwen3:8b-16k not found. Installing..."
        ollama pull qwen3:8b-16k &
        echo "   📥 Downloading model in background (this takes ~5 min)"
    fi
else
    echo "   ❌ Ollama not running!"
    echo "   📝 Start Ollama with: ollama serve"
    exit 1
fi

# Step 5: Ready
echo ""
echo "✅ Deployment complete!"
echo ""
echo "Next steps:"
echo "1. Start wa-echo-loop: npm start"
echo "2. Check health: curl http://localhost:8003/admin/health | jq"
echo "3. Send a WhatsApp message (from registered contact)"
echo ""
echo "Configuration:"
echo "   Model: qwen3:8b-16k (configurable via LLM_MODEL)"
echo "   Ollama: http://localhost:11434 (configurable via OLLAMA_URL)"
echo "   Admin API: http://localhost:8003"
echo ""
echo "Rollback (if needed):"
echo "   cp app.js.backup app.js"
echo ""
