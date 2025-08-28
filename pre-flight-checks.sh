#!/bin/bash

# Pre-flight Checks - Simple dependency and service validation

set -e

# =============================================================================
# CONFIGURATION - Edit these values as needed
# =============================================================================

# Required Commands
REQUIRED_COMMANDS=("docker" "docker-compose" "go")
OPTIONAL_COMMANDS=("k6" "xk6" "orchestrion")

# Required Files  
REQUIRED_FILES=("docker-compose.yml" "otel-collector-config.yaml" "main.go" "go.mod")

# Service URLs for health checks
GRAFANA_URL="http://localhost:3000/api/health"
INFLUXDB_URL="http://localhost:8086/health"  
PROMETHEUS_URL="http://localhost:9090/-/ready"
JAEGER_URL="http://localhost:16686/"

# =============================================================================
# SCRIPT LOGIC
# =============================================================================

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

check_deps() {
    local missing=0
    
    # Check required commands
    for cmd in "${REQUIRED_COMMANDS[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            echo -e "${RED}✗${NC} $cmd missing"
            ((missing++))
        fi
    done
    
    # Check optional commands  
    for cmd in "${OPTIONAL_COMMANDS[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            echo -e "${YELLOW}⚠${NC} $cmd missing (optional)"
        fi
    done
    
    # Check files
    for file in "${REQUIRED_FILES[@]}"; do
        if [ ! -f "$file" ]; then
            echo -e "${RED}✗${NC} $file missing"
            ((missing++))
        fi
    done
    
    # Check Docker daemon
    if ! docker info >/dev/null 2>&1; then
        echo -e "${RED}✗${NC} Docker daemon not running"
        ((missing++))
    fi
    
    return $missing
}

check_services() {
    local services=(
        "$INFLUXDB_URL|InfluxDB"
        "$GRAFANA_URL|Grafana"
        "$PROMETHEUS_URL|Prometheus" 
        "$JAEGER_URL|Jaeger"
    )
    
    local healthy=0
    for service in "${services[@]}"; do
        local url=$(echo "$service" | cut -d'|' -f1)
        local name=$(echo "$service" | cut -d'|' -f2)
        
        if curl -sf "$url" >/dev/null 2>&1; then
            ((healthy++))
        fi
    done
    
    echo -e "${GREEN}✓${NC} $healthy/4 services healthy"
}

# Run checks
echo "🔍 Checking dependencies..."
if check_deps; then
    echo -e "${GREEN}✓${NC} Dependencies OK"
else
    echo -e "${RED}✗${NC} Missing dependencies"
    exit 1
fi

echo "🏥 Checking services..."
check_services

echo -e "${GREEN}✅${NC} Pre-flight complete"