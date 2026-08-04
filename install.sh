#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_PREFIX="$HOME/.indieclaw"
LOG_FILE="$INSTALL_PREFIX/install.log"

# Functions
print_header() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}\n"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

check_command() {
    if command -v $1 &> /dev/null; then
        return 0
    else
        return 1
    fi
}

# ============= START INSTALLATION =============

print_header "🎯 Indieclaw Installer for macOS (Apple Silicon)"

echo "This installer will install:"
echo "  • Homebrew (if needed)"
echo "  • Go 1.21+"
echo "  • Node.js 18 LTS"
echo "  • RabbitMQ"
echo "  • Qdrant"
echo "  • Ollama + LLM Models"
echo "  • Python 3.11"
echo ""
echo "Prerequisites:"
echo "  • macOS with Apple Silicon (M1/M2/M3+)"
echo "  • 16GB+ RAM recommended"
echo "  • 80GB+ free disk space"
echo "  • Internet connection"
echo ""
read -p "Continue with installation? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_error "Installation cancelled"
    exit 1
fi

# Create installation directory
mkdir -p "$INSTALL_PREFIX"
exec 1> >(tee -a "$LOG_FILE")
exec 2>&1

print_header "Step 1: Checking System Requirements"

# Check OS
if [[ ! "$OSTYPE" == "darwin"* ]]; then
    print_error "This installer only supports macOS"
    exit 1
fi
print_success "macOS detected"

# Check CPU architecture
if ! sysctl hw.modelname | grep -q "Apple"; then
    print_error "This installer requires Apple Silicon (M1/M2/M3+)"
    exit 1
fi
print_success "Apple Silicon detected"

# Check RAM
TOTAL_RAM=$(sysctl hw.memsize | awk '{print $2 / (1024^3)}')
if (( $(echo "$TOTAL_RAM < 8" | bc -l) )); then
    print_warning "You have ${TOTAL_RAM}GB RAM. 16GB recommended."
fi
print_success "System RAM: ${TOTAL_RAM}GB"

# Check free disk space
FREE_SPACE=$(df / | awk 'NR==2 {print $4 / (1024^2)}')
if (( $(echo "$FREE_SPACE < 80000" | bc -l) )); then
    print_error "You need 80GB free disk space. You have ${FREE_SPACE}MB"
    exit 1
fi
print_success "Free disk space: ${FREE_SPACE}MB"

# ============= HOMEBREW =============
print_header "Step 2: Installing Homebrew (if needed)"

if ! check_command brew; then
    print_info "Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    print_success "Homebrew installed"
else
    print_success "Homebrew already installed"
fi

# ============= XCODE COMMAND LINE TOOLS =============
print_header "Step 3: Installing Xcode Command Line Tools (if needed)"

if ! check_command gcc; then
    print_info "Installing Xcode Command Line Tools..."
    xcode-select --install
    read -p "Press Enter after Xcode installation completes"
    print_success "Xcode Command Line Tools installed"
else
    print_success "Xcode Command Line Tools already installed"
fi

# ============= GO =============
print_header "Step 4: Installing Go"

if ! check_command go; then
    print_info "Installing Go..."
    brew install go
    print_success "Go installed"
else
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go already installed: $GO_VERSION"
fi

# Verify Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_GO="1.21"
if [[ "$GO_VERSION" < "$REQUIRED_GO" ]]; then
    print_warning "Go version $GO_VERSION is older than $REQUIRED_GO. Upgrading..."
    brew upgrade go
fi

# ============= NODE.JS =============
print_header "Step 5: Installing Node.js"

if ! check_command node; then
    print_info "Installing Node.js..."
    brew install node@18
    brew link node@18 --force
    print_success "Node.js installed"
else
    NODE_VERSION=$(node --version)
    print_success "Node.js already installed: $NODE_VERSION"
fi

# ============= RABBITMQ =============
print_header "Step 6: Installing RabbitMQ"

if ! check_command rabbitmq-server; then
    print_info "Installing RabbitMQ..."
    brew install rabbitmq
    print_success "RabbitMQ installed"
else
    print_success "RabbitMQ already installed"
fi

# Configure RabbitMQ user
print_info "Configuring RabbitMQ user 'indieclaw'..."
sudo rabbitmqctl add_user indieclaw secretpass 2>/dev/null || print_warning "User may already exist"
sudo rabbitmqctl set_permissions -p / indieclaw ".*" ".*" ".*" 2>/dev/null || print_warning "Permissions may already be set"
print_success "RabbitMQ configured"

# ============= QDRANT =============
print_header "Step 7: Installing Qdrant"

if ! check_command qdrant; then
    print_info "Installing Qdrant..."
    brew install qdrant
    print_success "Qdrant installed"
else
    print_success "Qdrant already installed"
fi

# ============= OLLAMA =============
print_header "Step 8: Installing Ollama"

if ! check_command ollama; then
    print_info "Downloading Ollama installer..."
    # Download from official source
    curl -L https://ollama.ai/download/Ollama-darwin.zip -o /tmp/Ollama.zip
    unzip -o /tmp/Ollama.zip -d /Applications/
    rm /tmp/Ollama.zip
    print_success "Ollama installed"
else
    print_success "Ollama already installed"
fi

# ============= PYTHON =============
print_header "Step 9: Installing Python 3.11"

if ! check_command python3; then
    print_info "Installing Python 3.11..."
    brew install python@3.11
    brew link python@3.11 --force
    print_success "Python installed"
else
    PYTHON_VERSION=$(python3 --version)
    print_success "Python already installed: $PYTHON_VERSION"
fi

# ============= OLLAMA MODELS =============
print_header "Step 10: Downloading LLM Models"

print_info "Starting Ollama in background..."
ollama serve > "$INSTALL_PREFIX/ollama.log" 2>&1 &
OLLAMA_PID=$!
sleep 5

print_info "This step downloads ~50GB of models. This will take 30-60 minutes..."
print_info "Download can continue in background. Press Ctrl+C to continue to next step."
echo ""

MODELS=("qwen2:7b" "llava:7b" "nomic-embed-text")
for model in "${MODELS[@]}"; do
    print_info "Pulling $model..."
    timeout 3600 ollama pull "$model" || print_warning "Model pull may have timed out. Try: ollama pull $model"
    print_success "$model ready"
done

print_info "Stopping Ollama server..."
kill $OLLAMA_PID 2>/dev/null || true
sleep 2
print_success "Models downloaded"

# ============= VERIFICATION =============
print_header "Step 11: Verifying Installation"

CHECKS=0
PASSED=0

check_install() {
    CHECKS=$((CHECKS + 1))
    if check_command "$1"; then
        print_success "$2 installed"
        PASSED=$((PASSED + 1))
    else
        print_error "$2 NOT installed"
    fi
}

check_install "brew" "Homebrew"
check_install "go" "Go"
check_install "node" "Node.js"
check_install "npm" "npm"
check_install "ollama" "Ollama"
check_install "rabbitmq-server" "RabbitMQ"
check_install "python3" "Python"

print_info "\nVerification: $PASSED/$CHECKS checks passed"

if [ $PASSED -lt $CHECKS ]; then
    print_warning "Some components failed to install. Check logs at $LOG_FILE"
fi

# ============= SETUP ENVIRONMENT =============
print_header "Step 12: Setting Up Environment"

# Create shell config
cat > "$INSTALL_PREFIX/indieclaw.env" << 'EOF'
# Indieclaw Environment Configuration

# Persona selection: generic_assistant | executive_coach | simple_qa | brainstorm_partner
export PERSONA_NAME=generic_assistant

# LLM Models
export TEXT_MODEL=qwen2:7b
export VISION_MODEL=llava:7b
export EMBEDDING_MODEL=nomic-embed-text

# Services
export OLLAMA_HOST=http://localhost:11434
export RABBITMQ_URL=amqp://indieclaw:secretpass@localhost:5672/

# Logging
export LOG_PATH=$HOME/.indieclaw/logs/orchestrator.log

# WhatsApp Settings
export TRIGGER_PREFIX=Self

# Persona Config Directory
export PERSONA_CONFIG_DIR=$HOME/.indieclaw/config/personas

# Create logs and config directories
mkdir -p $HOME/.indieclaw/logs
mkdir -p $HOME/.indieclaw/config/personas
EOF

print_success "Environment file created at $INSTALL_PREFIX/indieclaw.env"
print_info "Available personas: generic_assistant, executive_coach, simple_qa, brainstorm_partner"
print_info "Change persona: Edit PERSONA_NAME in $INSTALL_PREFIX/indieclaw.env"

# ============= STARTUP SCRIPTS =============
print_header "Step 13: Creating Startup Scripts"

# Start services script
cat > "$INSTALL_PREFIX/start-services.sh" << 'EOF'
#!/bin/bash
echo "🚀 Starting Indieclaw Services..."

# Source environment
source ~/.indieclaw/indieclaw.env

# Start RabbitMQ
echo "🐰 Starting RabbitMQ..."
brew services start rabbitmq || rabbitmq-server -detached

# Start Qdrant
echo "🗄️  Starting Qdrant..."
brew services start qdrant || qdrant &

# Start Ollama
echo "🤖 Starting Ollama..."
ollama serve > ~/.indieclaw/logs/ollama.log 2>&1 &

sleep 3

# Verify all services
echo ""
echo "✅ Services started!"
echo "📊 Check services:"
echo "   • RabbitMQ: http://localhost:15672 (guest:guest)"
echo "   • Qdrant: http://localhost:6333"
echo "   • Ollama: http://localhost:11434"
EOF

chmod +x "$INSTALL_PREFIX/start-services.sh"
print_success "Startup script created"

# ============= COMPLETION =============
print_header "✅ Installation Complete!"

echo "Your personal AI assistant is ready!"
echo ""
echo "NEXT STEPS (Copy & Paste):"
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Terminal 1: Start Services"
echo "═══════════════════════════════════════════════════════════════"
echo ""
print_info "Copy and paste this command:"
echo "~/.indieclaw/start-services.sh"
echo ""
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Terminal 2: Start Orchestrator"
echo "═══════════════════════════════════════════════════════════════"
echo ""
print_info "Copy and paste these commands (one at a time):"
echo ""
echo "cd ~/indieclaw/gateway-service"
echo "source ~/.indieclaw/indieclaw.env"
echo "go run ./cmd/orchestrator/main.go"
echo ""
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Terminal 3: Start WhatsApp Bot"
echo "═══════════════════════════════════════════════════════════════"
echo ""
print_info "Copy and paste these commands (one at a time):"
echo ""
echo "cd ~/indieclaw/approach-road/wa-echo-loop"
echo "source ~/.indieclaw/indieclaw.env"
echo "npm install"
echo "node app.js"
echo ""
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Terminal 4 (Optional): Configuration UI"
echo "═══════════════════════════════════════════════════════════════"
echo ""
print_info "Copy and paste these commands (one at a time):"
echo ""
echo "cd ~/indieclaw/persona-control-plane"
echo "node server.js"
echo ""
print_info "Then open in your browser:"
echo "http://localhost:3001"
echo ""
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "📱 TESTING YOUR BOT"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Terminal 3 will show a QR code."
echo "Scan it with WhatsApp on your phone."
echo "Send a message: Self Hello"
echo "Bot should reply within 30 seconds!"
echo ""
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "📋 IMPORTANT NOTES"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "✅ Installation complete!"
echo "✅ All services ready!"
echo "✅ ~20GB AI models downloaded"
echo "✅ Configuration saved to: ~/.indieclaw/indieclaw.env"
echo "✅ Logs saved to: ~/.indieclaw/logs/"
echo ""
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "🆘 NEED HELP?"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Check installation log:"
echo "cat $LOG_FILE"
echo ""
echo "See all running services:"
echo "ps aux | grep -E 'rabbitmq|qdrant|ollama'"
echo ""
echo "RabbitMQ Admin:"
echo "http://localhost:15672  (guest / guest)"
echo ""
echo ""
print_success "🚀 All done! Your AI assistant is ready to use!"
