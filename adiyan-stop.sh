#!/bin/bash

###############################################################################
# ADIYAN STOP SCRIPT
# Gracefully stops all running Adiyan services
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

log_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

main() {
    log_info "Stopping Adiyan suite..."

    if [ ! -f "$PIDS_FILE" ]; then
        log_warning "No PID file found. No services to stop."
        return 0
    fi

    local stopped_count=0
    while IFS=: read -r service pid; do
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping $service (PID: $pid)..."
            kill -TERM "$pid" 2>/dev/null || true
            sleep 1
            if ! kill -0 "$pid" 2>/dev/null; then
                log_success "$service stopped"
                ((stopped_count++))
            else
                log_warning "$service did not stop gracefully, forcing..."
                kill -9 "$pid" 2>/dev/null || true
                log_success "$service forced stop"
                ((stopped_count++))
            fi
        else
            log_warning "$service (PID: $pid) not running"
        fi
    done < "$PIDS_FILE"

    rm -f "$PIDS_FILE"


    echo ""
    log_success "Adiyan suite stopped ($stopped_count services)"
}

main "$@"
