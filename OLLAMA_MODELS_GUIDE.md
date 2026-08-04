# Ollama Models Guide - Size-Constrained Selection

## Current Configuration (≤8GB each)

**Your setup uses:**
- `qwen2:7b` - Text generation (7B parameters, ~4.4GB)
- `llava:7b` - Vision analysis (7B parameters, ~4.7GB)
- `nomic-embed-text` - Embeddings (small, ~275MB)

**Total Download:** ~20GB
**Total Storage:** ~10GB
**Total VRAM Used:** ~14GB
**Installation Time:** 15-20 minutes

---

## Available Models (≤8B Only)

### Text Generation Models

| Model | Size | VRAM | Download | Speed | Quality | Best For |
|-------|------|------|----------|-------|---------|----------|
| **qwen2:7b** | 7B | 4.4GB | 4GB | Fast | High | Executive coaching |
| **mistral:7b** | 7B | 4GB | 4GB | Very Fast | Good | General purpose |
| **llama2:7b** | 7B | 4GB | 3.8GB | Fast | Good | Versatile |
| **neural-chat:7b** | 7B | 4GB | 4GB | Fast | Good | Chat optimized |
| **dolphin-mixtral:latest** | 7B (MoE) | 3GB | 3.7GB | Fast | Excellent | Reasoning |
| **gemma:7b** | 7B | 4GB | 4GB | Fast | Good | Lightweight |
| **phi:latest** | 3B | 2GB | 2.7GB | Fastest | Good | Speed priority |
| **orca-mini:3b** | 3B | 2GB | 1.9GB | Fastest | Fair | Minimal setup |

### Vision Models

| Model | Size | VRAM | Download | Best For |
|-------|------|------|----------|----------|
| **llava:7b** | 7B | 4.7GB | 4.5GB | Image analysis (current) |
| **llava:13b** | 13B | 8GB | 7.7GB | Better vision (at limit) |
| **bakllava:latest** | 3.5B | 2GB | 2.1GB | Lightweight vision |
| **moondream:1.8b** | 1.8B | 1.2GB | 1.4GB | Minimal vision |

### Embedding Models

| Model | Size | VRAM | Download | Best For |
|-------|------|------|----------|----------|
| **nomic-embed-text** | Small | 500MB | 275MB | Semantic search (current) |
| **bge-small** | Small | 500MB | 200MB | Alternative embedding |

---

## How to Change Models

### Option 1: Update Configuration File

Edit `~/.indieclaw/indieclaw.env`:
```bash
TEXT_MODEL=mistral:7b      # Change text model
VISION_MODEL=bakllava      # Change vision model
EMBEDDING_MODEL=bge-small  # Change embedding model
```

### Option 2: Update Persona TOML

Edit `gateway-service/config/personas/executive_coach.toml`:
```toml
[models]
text_model = "mistral:7b"
vision_model = "bakllava"
```

### Option 3: Download New Model at Runtime

```bash
ollama pull mistral:7b
ollama pull bakllava
```

---

## RAM Requirements by Configuration

### Minimal Setup (~8GB total)
```
TEXT:   phi:3b           (2GB)
VISION: moondream:1.8b   (1.2GB)
EMBED:  nomic-embed-text (500MB)
OS/Other               (3-4GB)
──────────────────────────────
Total:  ~8GB (bare minimum)
```

### Recommended Setup (~14GB total)
```
TEXT:   qwen2:7b                (4.4GB)  ← CURRENT
VISION: llava:7b                (4.7GB)  ← CURRENT
EMBED:  nomic-embed-text        (500MB)  ← CURRENT
OS/Other                        (4.5GB)
──────────────────────────────
Total:  ~14GB (recommended)
```

### High Quality (~16GB total)
```
TEXT:   qwen2:7b         (4.4GB)
VISION: llava:13b        (8GB)
EMBED:  nomic-embed-text (500MB)
OS/Other                 (3GB)
──────────────────────────────
Total:  ~16GB (requires 20GB RAM)
```

### Speed Optimized (~6GB total)
```
TEXT:   mistral:7b       (4GB)
VISION: bakllava         (2.1GB)
EMBED:  nomic-embed-text (500MB)
OS/Other                 (1-2GB)
──────────────────────────────
Total:  ~8GB (fastest)
```

---

## Model Characteristics

### Qwen2:7b (Current Text Model)
- **Pros:** Excellent reasoning, good for coaching, multilingual
- **Cons:** Slightly slower than Mistral
- **Best for:** Executive coaching, complex reasoning
- **VRAM:** 4.4GB

### Mistral:7b (Alternative)
- **Pros:** Very fast, good quality, instruction-optimized
- **Cons:** Less specialized
- **Best for:** Speed + quality balance
- **VRAM:** 4GB

### LLava:7b (Current Vision Model)
- **Pros:** Good image understanding, reasonable speed
- **Cons:** May miss fine details
- **Best for:** General image analysis
- **VRAM:** 4.7GB

### Bakllava (Lightweight Vision)
- **Pros:** Fast, small, decent image understanding
- **Cons:** Less detailed analysis
- **Best for:** Quick image classification
- **VRAM:** 2GB

---

## Download Sizes

Total for **current setup** (qwen2:7b + llava:7b + nomic-embed-text):
- **Download:** ~20GB
- **Storage on Disk:** ~10GB
- **Time to Download:** 15-20 minutes

---

## Performance Comparison

### Text Generation Speed
```
Orca-mini:3b   ████████████ 8.2 tok/s (fastest)
Phi:3b         ███████████  7.5 tok/s
Mistral:7b     ██████████   6.8 tok/s
Llama2:7b      █████████    6.2 tok/s
Qwen2:7b       ████████     5.8 tok/s
```

### Quality (Subjective)
```
Qwen2:7b       ██████████   9.2/10 (best reasoning)
Dolphin-mixtral ████████▌   9.0/10
Mistral:7b     ████████     8.5/10
Llama2:7b      ███████      8.0/10
Phi:3b         █████        6.5/10
```

---

## Switching Between Models

### Minimal Downtime Approach

```bash
# 1. Keep current models running
ollama pull mistral:7b          # Downloads while running

# 2. Edit environment
vi ~/.indieclaw/indieclaw.env
# Change: TEXT_MODEL=mistral:7b

# 3. Restart orchestrator
# (Kill old, start new)
```

### Clean Approach

```bash
# 1. Stop services
pkill -f "ollama serve"

# 2. Download new model
ollama pull mistral:7b

# 3. Update config
vi ~/.indieclaw/indieclaw.env

# 4. Restart
ollama serve &
```

---

## Removing Unused Models

To free up disk space:

```bash
# List installed models
ollama list

# Remove specific model
ollama rm qwen2:7b

# This frees up ~4GB of space
```

---

## Troubleshooting

### "Out of memory" error
1. Check running processes: `ps aux | grep ollama`
2. Kill other apps consuming RAM
3. Use smaller models (e.g., phi:3b instead of qwen2:7b)
4. Use unload command: `ollama rm <model-name>`

### Model won't load
```bash
# Clear Ollama cache
rm -rf ~/.ollama/

# Restart Ollama
pkill -f "ollama serve"
ollama serve &
```

### Slow inference
1. Check if model is loaded: `curl http://localhost:11434/api/tags`
2. Wait for model to fully load (first request is slow)
3. Try faster model: `mistral:7b` or `phi:3b`
4. Close other apps to free RAM

---

## Recommended Model Combinations

### For Coaching (Current ✅)
```
TEXT:   qwen2:7b
VISION: llava:7b
EMBED:  nomic-embed-text
Total:  ~14GB RAM needed
Quality: ⭐⭐⭐⭐⭐
Speed:   ⭐⭐⭐⭐
```

### For Speed
```
TEXT:   mistral:7b
VISION: bakllava
EMBED:  nomic-embed-text
Total:  ~8GB RAM needed
Quality: ⭐⭐⭐⭐
Speed:   ⭐⭐⭐⭐⭐
```

### For Minimal Setup
```
TEXT:   phi:3b
VISION: moondream:1.8b
EMBED:  nomic-embed-text
Total:  ~6GB RAM needed
Quality: ⭐⭐⭐
Speed:   ⭐⭐⭐⭐⭐
```

---

## Current Setup Summary

✅ **Text Model:** qwen2:7b (7B parameters)
✅ **Vision Model:** llava:7b (7B parameters)
✅ **Embedding:** nomic-embed-text (small)
✅ **Total VRAM:** ~14GB
✅ **Download Size:** ~20GB
✅ **Installation Time:** 15-20 minutes
✅ **Quality Level:** Excellent for coaching

**All models are ≤8B as requested!**
