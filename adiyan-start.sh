#!/bin/bash

###############################################################################
# ADIYAN STARTUP SCRIPT
# Launches the complete Adiyan coaching platform suite
#
# Services:
# - Nginx (gateway)
# - Qdrant (vector database)
# - Orchestrator (AI coaching engine)
# - User Context Service (user knowledge management)
# - WhatsApp Adapter (messaging)
# - Discord Adapter (messaging)
# - Person Config Node (persona management)
###############################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ADIYAN_HOME="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOGS_DIR="$ADIYAN_HOME/logs"
PIDS_FILE="$LOGS_DIR/adiyan.pids"
TEMP_DIR="${TMPDIR:-/tmp}/adiyan"

# Service ports
NGINX_PORT=80
RABBITMQ_PORT=5672
RABBITMQ_MGMT_PORT=15672
QDRANT_PORT=6333
QDRANT_HTTP_PORT=6334
ORCHESTRATOR_PORT=9000
UCS_PORT=8001
WHATSAPP_PORT=3000
DISCORD_PORT=3001
PERSON_CONFIG_PORT=8080

# Service configuration
PERSONA_NAME="${PERSONA_NAME:-executive_coach}"
TEXT_MODEL="${TEXT_MODEL:-qwen2:7b}"
VISION_MODEL="${VISION_MODEL:-llava:7b}"
EMBEDDING_MODEL="${EMBEDDING_MODEL:-nomic-embed-text}"

###############################################################################
# UTILITY FUNCTIONS
###############################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

check_port_available() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 1
    fi
    return 0
}

ensure_directory() {
    mkdir -p "$1"
}

wait_for_service() {
    local port=$1
    local name=$2
    local max_attempts=30
    local attempt=0

    log_info "Waiting for $name to be ready (port $port)..."
    while [ $attempt -lt $max_attempts ]; do
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            log_success "$name is ready"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done

    log_error "$name did not start within timeout"
    return 1
}

save_pid() {
    local pid=$1
    local service=$2
    echo "$service:$pid" >> "$PIDS_FILE"
}

cleanup() {
    log_warning "Shutting down Adiyan suite..."

    if [ -f "$PIDS_FILE" ]; then
        while IFS=: read -r service pid; do
            if kill -0 "$pid" 2>/dev/null; then
                log_info "Stopping $service (PID: $pid)..."
                kill -TERM "$pid" 2>/dev/null || true
                sleep 1
                kill -9 "$pid" 2>/dev/null || true
            fi
        done < "$PIDS_FILE"
        rm -f "$PIDS_FILE"
    fi

    log_success "Adiyan suite stopped"
    exit 0
}

###############################################################################
# PRE-FLIGHT CHECKS
###############################################################################

preflight_checks() {
    log_info "Running pre-flight checks..."

    # Check if binaries exist
    local binaries=("orchestrator" "user-context-service")
    for binary in "${binaries[@]}"; do
        if [ ! -f "$ADIYAN_HOME/gateway-service/bin/$binary" ]; then
            log_error "Binary not found: $ADIYAN_HOME/gateway-service/bin/$binary"
            log_warning "Run: cd gateway-service && go build -o ./bin/$binary ./cmd/$binary/main.go"
            exit 1
        fi
    done

    # Check if required services are available
    command -v nginx >/dev/null 2>&1 || { log_error "nginx not found. Install with: brew install nginx"; exit 1; }
    
    log_success "Pre-flight checks passed"
}

###############################################################################
# SERVICE STARTUP FUNCTIONS
###############################################################################

start_nginx() {
    log_info "Checking Nginx gateway..."

    # Check if already running
    if ! check_port_available $NGINX_PORT; then
        if wait_for_service $NGINX_PORT "Nginx"; then
            log_success "Nginx already running (port $NGINX_PORT)"
            return 0
        fi
    fi

    # Use existing nginx config
    local nginx_conf="$ADIYAN_HOME/nginx/nginx.conf"
    if [ ! -f "$nginx_conf" ]; then
        log_warning "Nginx config not found, skipping"
        return 0
    fi

    # Port 80 requires sudo
    log_info "Starting Nginx (port $NGINX_PORT requires sudo)..."
    sudo nginx -c "$nginx_conf" > "$LOGS_DIR/nginx.log" 2>&1 &
    local nginx_pid=$!
    save_pid $nginx_pid "nginx"

    sleep 1
    if wait_for_service $NGINX_PORT "Nginx"; then
        log_success "Nginx started (PID: $nginx_pid)"
        return 0
    else
        log_warning "Nginx failed to start (may require manual: sudo nginx -c $nginx_conf)"
        return 0
    fi
}

start_rabbitmq() {
    log_info "Checking RabbitMQ message broker..."

    # Check if already running
    if ! check_port_available $RABBITMQ_PORT; then
        if wait_for_service $RABBITMQ_PORT "RabbitMQ"; then
            log_success "RabbitMQ already running (port $RABBITMQ_PORT)"
            return 0
        fi
    fi

    # Try to start via brew services (macOS)
    if command -v brew >/dev/null 2>&1; then
        log_info "Starting RabbitMQ via brew..."
        brew services start rabbitmq > "$LOGS_DIR/rabbitmq.log" 2>&1

        if wait_for_service $RABBITMQ_PORT "RabbitMQ"; then
            log_success "RabbitMQ started (port $RABBITMQ_PORT)"
            return 0
        fi
    fi

    # Fallback: provide manual instructions
    log_error "❌ RabbitMQ is not running and could not be started automatically"
    echo ""
    echo -e "${YELLOW}To start RabbitMQ manually:${NC}"
    echo "  brew services start rabbitmq"
    echo ""
    return 1
}

start_qdrant() {
    log_info "Checking Qdrant vector database..."

    # Check if already running
    if ! check_port_available $QDRANT_HTTP_PORT; then
        if wait_for_service $QDRANT_HTTP_PORT "Qdrant"; then
            log_success "Qdrant already running (port $QDRANT_HTTP_PORT)"
            return 0
        fi
    fi

    # Use local qdrant executable
    local qdrant_bin="$ADIYAN_HOME/gateway-service/bin/qdrant"

    if [ ! -f "$qdrant_bin" ]; then
        log_error "❌ Qdrant executable not found at $qdrant_bin"
        return 1
    fi

    # Check for config file, use if exists
    local qdrant_config="$ADIYAN_HOME/gateway-service/bin/qdrant-config.yaml"
    if [ -f "$qdrant_config" ]; then
        log_info "Starting Qdrant (with config)..."
        "$qdrant_bin" --config-path "$qdrant_config" > "$LOGS_DIR/qdrant.log" 2>&1 &
    else
        log_info "Starting Qdrant (default config)..."
        "$qdrant_bin" > "$LOGS_DIR/qdrant.log" 2>&1 &
    fi

    local qdrant_pid=$!
    save_pid $qdrant_pid "qdrant"

    if wait_for_service $QDRANT_HTTP_PORT "Qdrant"; then
        log_success "Qdrant started (PID: $qdrant_pid)"
        return 0
    else
        log_error "Qdrant failed to start"
        return 1
    fi
}

start_orchestrator() {
    log_info "Starting Orchestrator (AI coaching engine)..."

    # Check if port is available
    if ! check_port_available $ORCHESTRATOR_PORT; then
        log_error "Port $ORCHESTRATOR_PORT already in use"
        return 1
    fi

    cd "$ADIYAN_HOME/gateway-service"

    export PERSONA_NAME="$PERSONA_NAME"
    export TEXT_MODEL="$TEXT_MODEL"
    export VISION_MODEL="$VISION_MODEL"
    export EMBEDDING_MODEL="$EMBEDDING_MODEL"
    export LOG_PATH="$LOGS_DIR/orchestrator.log"

    ./bin/orchestrator > "$LOGS_DIR/orchestrator.log" 2>&1 &
    local orch_pid=$!
    save_pid $orch_pid "orchestrator"

    if wait_for_service $ORCHESTRATOR_PORT "Orchestrator"; then
        log_success "Orchestrator started (PID: $orch_pid)"
        return 0
    else
        log_error "Orchestrator failed to start"
        return 1
    fi
}

start_user_context_service() {
    log_info "Starting User Context Service..."

    # Check if port is available
    if ! check_port_available $UCS_PORT; then
        log_error "Port $UCS_PORT already in use"
        return 1
    fi

    cd "$ADIYAN_HOME/gateway-service"

    export USER_DATA_DIR="$HOME/.adiyan/users"
    export UCS_PORT=":$UCS_PORT"
    export LOG_LEVEL="info"

    ./bin/user-context-service > "$LOGS_DIR/ucs.log" 2>&1 &
    local ucs_pid=$!
    save_pid $ucs_pid "user-context-service"

    # Use just the port number for checking
    if wait_for_service $UCS_PORT "User Context Service"; then
        log_success "User Context Service started (PID: $ucs_pid)"
        cd "$ADIYAN_HOME"
        return 0
    else
        log_error "User Context Service failed to start"
        cd "$ADIYAN_HOME"
        return 1
    fi
}

start_whatsapp_adapter() {
    log_info "Starting WhatsApp Adapter..."

    if [ ! -d "$ADIYAN_HOME/approach-road/wa-echo-loop" ]; then
        log_warning "WhatsApp adapter not found. Skipping..."
        return 0
    fi

    cd "$ADIYAN_HOME/approach-road/wa-echo-loop"

    # Check for dependencies
    if [ ! -f "package.json" ]; then
        log_warning "WhatsApp adapter package.json not found. Skipping..."
        return 0
    fi

    if [ ! -f "app.js" ]; then
        log_warning "WhatsApp adapter app.js not found. Skipping..."
        return 0
    fi

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        log_info "Installing WhatsApp adapter dependencies..."
        npm install >> "$LOGS_DIR/whatsapp.log" 2>&1
    fi

    export PERSONA_NAME="$PERSONA_NAME"
    node app.js > "$LOGS_DIR/whatsapp.log" 2>&1 &
    local wa_pid=$!
    save_pid $wa_pid "whatsapp-adapter"

    sleep 2
    log_success "WhatsApp adapter started (PID: $wa_pid)"
    cd "$ADIYAN_HOME"
}

start_discord_adapter() {
    log_info "Starting Discord Adapter..."

    if [ ! -d "$ADIYAN_HOME/approach-road/discord" ]; then
        log_warning "Discord adapter not found. Skipping..."
        return 0
    fi

    cd "$ADIYAN_HOME/approach-road/discord"

    # Check for dependencies
    if [ ! -f "package.json" ]; then
        log_warning "Discord adapter package.json not found. Skipping..."
        return 0
    fi

    if [ ! -f "index.js" ] && [ ! -f "app.js" ] && [ ! -f "bot.js" ]; then
        log_warning "Discord adapter entry file not found. Skipping..."
        return 0
    fi

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        log_info "Installing Discord adapter dependencies..."
        npm install >> "$LOGS_DIR/discord.log" 2>&1
    fi

    # Find entry file
    local entry_file="index.js"
    [ -f "app.js" ] && entry_file="app.js"
    [ -f "bot.js" ] && entry_file="bot.js"

    export PERSONA_NAME="$PERSONA_NAME"
    node "$entry_file" > "$LOGS_DIR/discord.log" 2>&1 &
    local discord_pid=$!
    save_pid $discord_pid "discord-adapter"

    sleep 2
    log_success "Discord adapter started (PID: $discord_pid)"
    cd "$ADIYAN_HOME"
}

start_person_config() {
    log_info "Starting Person Config Node..."

    # Check multiple possible directories
    local person_dir=""
    [ -d "$ADIYAN_HOME/approach-road/person-node" ] && person_dir="$ADIYAN_HOME/approach-road/person-node"
    [ -d "$ADIYAN_HOME/approach-road/person" ] && person_dir="$ADIYAN_HOME/approach-road/person"

    if [ -z "$person_dir" ]; then
        log_warning "Person config node not found. Skipping..."
        return 0
    fi

    cd "$person_dir"

    # Check for dependencies
    if [ ! -f "package.json" ]; then
        log_warning "Person config package.json not found. Skipping..."
        return 0
    fi

    if [ ! -f "index.js" ] && [ ! -f "app.js" ] && [ ! -f "server.js" ]; then
        log_warning "Person config entry file not found. Skipping..."
        return 0
    fi

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        log_info "Installing Person config dependencies..."
        npm install >> "$LOGS_DIR/person-config.log" 2>&1
    fi

    # Find entry file
    local entry_file="index.js"
    [ -f "app.js" ] && entry_file="app.js"
    [ -f "server.js" ] && entry_file="server.js"

    export PERSON_CONFIG_PORT=$PERSON_CONFIG_PORT
    node "$entry_file" > "$LOGS_DIR/person-config.log" 2>&1 &
    local person_pid=$!
    save_pid $person_pid "person-config"

    sleep 2
    log_success "Person config started (PID: $person_pid)"
    cd "$ADIYAN_HOME"
}

###############################################################################
# NGINX CONFIGURATION GENERATOR
###############################################################################

generate_nginx_config() {
    local config_file="$ADIYAN_HOME/nginx.conf"

    cat > "$config_file" << 'EOF'
worker_processes auto;

events {
    worker_connections 1024;
}

http {
    # Upstream services
    upstream orchestrator {
        server localhost:9000;
    }

    upstream user_context_service {
        server localhost:8001;
    }

    upstream whatsapp_adapter {
        server localhost:3000;
    }

    upstream discord_adapter {
        server localhost:3001;
    }

    upstream person_config {
        server localhost:8080;
    }

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;

    server {
        listen 80 default_server;
        server_name _;

        # Logging
        access_log logs/access.log;
        error_log logs/error.log;

        # Root
        location / {
            return 301 /api/v1/health;
        }

        # Health check
        location /health {
            access_log off;
            return 200 '{"status": "ok"}\n';
            add_header Content-Type application/json;
        }

        # Orchestrator API
        location /orchestrator/ {
            limit_req zone=api_limit burst=20;
            proxy_pass http://orchestrator/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_read_timeout 600s;
        }

        # User Context Service
        location /context/ {
            limit_req zone=api_limit burst=20;
            proxy_pass http://user_context_service/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }

        # WhatsApp Adapter
        location /whatsapp/ {
            proxy_pass http://whatsapp_adapter/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }

        # Discord Adapter
        location /discord/ {
            proxy_pass http://discord_adapter/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }

        # Person Config
        location /person/ {
            proxy_pass http://person_config/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }
    }
}
EOF

    log_success "Generated Nginx config: $config_file"
}

###############################################################################
# STATUS DISPLAY
###############################################################################

show_status() {
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}✨ ADIYAN SUITE STARTED SUCCESSFULLY ✨${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${BLUE}Services running:${NC}"
    echo -e "  ${GREEN}✓${NC} Nginx Gateway              : http://localhost:80"
    echo -e "  ${GREEN}✓${NC} RabbitMQ Broker            : localhost:5672"
    echo -e "  ${GREEN}✓${NC} RabbitMQ Management       : http://localhost:15672"
    echo -e "  ${GREEN}✓${NC} Qdrant Vector DB          : http://localhost:6334"
    echo -e "  ${GREEN}✓${NC} Orchestrator              : http://localhost:9000 (gRPC)"
    echo -e "  ${GREEN}✓${NC} User Context Service      : http://localhost:8001"
    echo ""
    echo -e "${BLUE}Optional services:${NC}"
    echo -e "  • WhatsApp Adapter            : http://localhost:3000"
    echo -e "  • Discord Adapter             : http://localhost:3001"
    echo -e "  • Person Config               : http://localhost:8080"
    echo ""
    echo -e "${BLUE}Configuration:${NC}"
    echo -e "  Persona                       : $PERSONA_NAME"
    echo -e "  Text Model                    : $TEXT_MODEL"
    echo -e "  Vision Model                  : $VISION_MODEL"
    echo -e "  User Data Dir                 : $HOME/.adiyan/users"
    echo ""
    echo -e "${BLUE}Logs:${NC}"
    echo -e "  Log directory                 : $LOGS_DIR"
    echo ""
    echo -e "${BLUE}Commands:${NC}"
    echo -e "  View logs                     : tail -f $LOGS_DIR/orchestrator.log"
    echo -e "  Health check                  : curl http://localhost/health"
    echo -e "  Stop services                 : Press Ctrl+C"
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
    echo ""
}

###############################################################################
# MAIN STARTUP SEQUENCE
###############################################################################

main() {
    log_info "Starting Adiyan Suite v1.0"
    log_info "Home: $ADIYAN_HOME"
    echo ""

    # Setup trap for graceful shutdown
    trap cleanup SIGINT SIGTERM

    # Create logs directory
    ensure_directory "$LOGS_DIR"
    ensure_directory "$TEMP_DIR"
    rm -f "$PIDS_FILE"

    # Run pre-flight checks
    preflight_checks
    echo ""

    # Start services in order
    log_info "Starting Adiyan services..."
    echo ""

    # Database services (required)
    start_rabbitmq || log_warning "RabbitMQ not started (see instructions above)"
    start_qdrant || log_warning "Qdrant not started (see instructions above)"

    # Gateway (optional)
    start_nginx || log_warning "Nginx failed to start"

    # Core services (required)
    start_orchestrator || { log_error "Failed to start Orchestrator"; exit 1; }
    start_user_context_service || { log_error "Failed to start User Context Service"; exit 1; }

    # Adapters (optional but included)
    log_info "Starting adapters..."
    start_whatsapp_adapter || log_warning "WhatsApp adapter not started"
    start_discord_adapter || log_warning "Discord adapter not started"
    start_person_config || log_warning "Person config not started"

    echo ""
    show_status

    # Keep the script running
    log_info "Adiyan suite is running. Press Ctrl+C to stop."

    # Wait indefinitely
    while true; do
        sleep 1
    done
}

# Run main function
main "$@"
