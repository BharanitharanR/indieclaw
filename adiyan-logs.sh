#!/bin/bash

###############################################################################
# ADIYAN LOGS SCRIPT
# View logs for Adiyan services
###############################################################################

ADIYAN_HOME="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOGS_DIR="$ADIYAN_HOME/logs"
PIDS_FILE="$LOGS_DIR/adiyan.pids"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

show_help() {
    cat << EOF
Usage: ./adiyan-logs.sh [COMMAND] [SERVICE]

Commands:
  status                Show status of all services
  tail SERVICE          Tail logs for a specific service
  follow SERVICE        Follow logs for a specific service
  clear                 Clear all logs
  help                  Show this help message

Services:
  orchestrator          AI Coaching Engine
  ucs                   User Context Service
  nginx                 Nginx Gateway
  rabbitmq              RabbitMQ Message Broker
  qdrant                Qdrant Vector Database
  whatsapp              WhatsApp Adapter
  discord               Discord Adapter
  person-config         Person Config Node

Examples:
  ./adiyan-logs.sh status
  ./adiyan-logs.sh tail orchestrator
  ./adiyan-logs.sh follow ucs
  ./adiyan-logs.sh clear

EOF
}

show_status() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}ADIYAN SERVICE STATUS${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo ""

    if [ ! -f "$PIDS_FILE" ]; then
        log_error "No services running (no PID file found)"
        return 1
    fi

    local running=0
    local stopped=0

    while IFS=: read -r service pid; do
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "  ${GREEN}✓${NC} $service (PID: $pid)"
            ((running++))
        else
            echo -e "  ${RED}✗${NC} $service (PID: $pid) - NOT RUNNING"
            ((stopped++))
        fi
    done < "$PIDS_FILE"

    echo ""
    echo -e "Running: ${GREEN}$running${NC}  Stopped: ${RED}$stopped${NC}"
    echo ""

    # Check port availability
    echo -e "${BLUE}Port Status:${NC}"
    ports=("80:nginx" "5672:rabbitmq" "15672:rabbitmq-mgmt" "6334:qdrant" "9000:orchestrator" "8001:ucs" "3000:whatsapp" "3001:discord" "8080:person-config")
    for port_service in "${ports[@]}"; do
        IFS=: read -r port service <<< "$port_service"
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo -e "  ${GREEN}✓${NC} Port $port ($service) - LISTENING"
        else
            echo -e "  ${RED}✗${NC} Port $port ($service) - NOT LISTENING"
        fi
    done

    echo ""
}

show_logs() {
    local service=$1
    local log_file

    case "$service" in
        orchestrator)
            log_file="$LOGS_DIR/orchestrator.log"
            ;;
        ucs)
            log_file="$LOGS_DIR/ucs.log"
            ;;
        nginx)
            log_file="$LOGS_DIR/nginx.log"
            ;;
        qdrant)
            log_file="$LOGS_DIR/qdrant.log"
            ;;
        whatsapp)
            log_file="$LOGS_DIR/whatsapp.log"
            ;;
        discord)
            log_file="$LOGS_DIR/discord.log"
            ;;
        person-config)
            log_file="$LOGS_DIR/person-config.log"
            ;;
        *)
            log_error "Unknown service: $service"
            return 1
            ;;
    esac

    if [ ! -f "$log_file" ]; then
        log_error "Log file not found: $log_file"
        return 1
    fi

    tail -n 50 "$log_file"
}

follow_logs() {
    local service=$1
    local log_file

    case "$service" in
        orchestrator)
            log_file="$LOGS_DIR/orchestrator.log"
            ;;
        ucs)
            log_file="$LOGS_DIR/ucs.log"
            ;;
        nginx)
            log_file="$LOGS_DIR/nginx.log"
            ;;
        qdrant)
            log_file="$LOGS_DIR/qdrant.log"
            ;;
        whatsapp)
            log_file="$LOGS_DIR/whatsapp.log"
            ;;
        discord)
            log_file="$LOGS_DIR/discord.log"
            ;;
        person-config)
            log_file="$LOGS_DIR/person-config.log"
            ;;
        *)
            log_error "Unknown service: $service"
            return 1
            ;;
    esac

    if [ ! -f "$log_file" ]; then
        log_error "Log file not found: $log_file"
        return 1
    fi

    log_info "Following logs for $service (Ctrl+C to stop)..."
    echo ""
    tail -f "$log_file"
}

clear_logs() {
    log_info "Clearing all logs..."
    rm -f "$LOGS_DIR"/*.log
    log_success "Logs cleared"
}

main() {
    mkdir -p "$LOGS_DIR"

    local command="${1:-status}"

    case "$command" in
        status)
            show_status
            ;;
        tail)
            if [ -z "$2" ]; then
                log_error "Please specify a service"
                show_help
                exit 1
            fi
            show_logs "$2"
            ;;
        follow)
            if [ -z "$2" ]; then
                log_error "Please specify a service"
                show_help
                exit 1
            fi
            follow_logs "$2"
            ;;
        clear)
            clear_logs
            ;;
        help)
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
