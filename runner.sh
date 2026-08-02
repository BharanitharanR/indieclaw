#!/bin/bash

# Constantscat
PID_DIR="/tmp/process_manager"
PID_FILE="${PID_DIR}/processes.json"
LOG_DIR="${PID_DIR}/log"
LOG_FILE="$LOG_DIR/run_script.log"

# Ensure directories exist
mkdir -p "$LOG_DIR" "$PID_DIR"

# Log function
log() {
    echo "$(date +'%Y-%m-%d %T') - $1" >> "$LOG_FILE"
}

# Validate command
if [[ "$1" != "start" && "$1" != "stop" && "$1" != "status" ]]; then
    log "Invalid command. Use: start, stop, or status."
    exit 1
fi


# Function to start services
start_services() {
    local services=(
        "npx mcp-proxy --port 3000 -- npx -y apple-mail-mcp"
        "nginx -c /Users/bharani/Desktop/aiAgentCompaction/indieclaw/nginx/nginx.conf"
        "/Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/bin/qdrant --config-path /Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/bin/qdrant-config.yaml"
        "/Users/bharani/Desktop/aiAgentCompaction/indieclaw/gateway-service/buildGo.sh"
    )

    local pid_map=()
    local success=true

    log "Starting services..."
    for service in "${services[@]}"; do
        log "Starting: $service"
        eval "$service &"
        local pid=$!
        local name=$(echo "$service" | awk '{print $1}')
        pid_map+=("{\"name\":\"$name\",\"pid\":$pid,\"status\":\"running\"}")
        log "Started: $name (PID: $pid)"
    done

    # Save PID data to JSON
    cat > "$PID_FILE" <<EOF
{
  "services": [
    ${pid_map[@]}
  ]
}
EOF
    chmod 600 "$PID_FILE"
    log "Saved process IDs to $PID_FILE"
}

# Function to stop services
stop_services() {
    log "Stopping services..."
    if [[ ! -f "$PID_FILE" ]]; then
        log "No running services found."
        return
    fi

    local services=($(jq -r '.services[] | .name' "$PID_FILE"))
    local pids=($(jq -r '.services[] | .pid' "$PID_FILE"))

    for i in "${!services[@]}"; do
        local name="${services[$i]}"
        local pid="${pids[$i]}"
        log "Stopping: $name (PID: $pid)"
        kill -9 "$pid" 2>/dev/null
        log "Stopped: $name"
    done

    # Clear JSON file
    > "$PID_FILE"
    log "Cleared process IDs from $PID_FILE"
}

# Function to check service status
check_status() {
    log "Checking service status..."
    if [[ ! -f "$PID_FILE" ]]; then
        log "No services are running."
        return
    fi

    local services=($(jq -r '.services[] | .name' "$PID_FILE"))
    local pids=($(jq -r '.services[] | .pid' "$PID_FILE"))
    local statuses=($(jq -r '.services[] | .status' "$PID_FILE"))

    for i in "${!services[@]}"; do
        local name="${services[$i]}"
        local pid="${pids[$i]}"
        local status="${statuses[$i]}"
        log "Service: $name (PID: $pid, Status: $status)"
    done
}

# Main logic
case "$1" in
    start)
        start_services
        ;;
    stop)
        stop_services
        ;;
    status)
        check_status
        ;;
esac
