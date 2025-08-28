#!/bin/bash

# Pre-flight Checks - Simple dependency and service validation

set -e

# =============================================================================
# CONFIGURATION - Edit these values as needed
# =============================================================================

# Required Commands
REQUIRED_COMMANDS=("docker" "go" "xk6" "orchestrion")

# Required Files
REQUIRED_FILES=("docker-compose.yml" "main.go" "go.mod" "k6_loadtesting.js")

# Service URLs for health checks
GRAFANA_URL="http://localhost:3000/api/health"
INFLUXDB_URL="http://localhost:8086/health"
PROMETHEUS_URL="http://localhost:9090/-/ready"
JAEGER_URL="http://localhost:16686/"

# =============================================================================
# SCRIPT LOGIC
# =============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

install_missing_deps() {
	local installed=0

	# Install xk6 if missing
	if ! command -v "xk6" >/dev/null 2>&1; then
		echo -e "${YELLOW}Installing xk6...${NC}"
		if command -v "go" >/dev/null 2>&1; then
			go install go.k6.io/xk6/cmd/xk6@latest && ((installed++))
		else
			echo -e "${RED}✗${NC} go not found. Cannot install xk6"
			return 1
		fi
	fi

	# Install orchestrion if missing
	if ! command -v "orchestrion" >/dev/null 2>&1; then
		echo -e "${YELLOW}Installing orchestrion...${NC}"
		if command -v "go" >/dev/null 2>&1; then
			go install github.com/DataDog/orchestrion@latest && ((installed++))
		else
			echo -e "${RED}✗${NC} go not found. Cannot install orchestrion"
			return 1
		fi
	fi

	if [ $installed -gt 0 ]; then
		echo -e "${GREEN}✓${NC} Installed $installed missing dependencies"
	fi

	return 0
}

check_deps() {
	local missing=0

	# Check required commands
	for cmd in "${REQUIRED_COMMANDS[@]}"; do
		if ! command -v "$cmd" >/dev/null 2>&1; then
			echo -e "${RED}✗${NC} $cmd missing"
			((missing++))
		else
			echo -e "${GREEN}✓${NC} $cmd"
		fi
	done

	# Check files
	for file in "${REQUIRED_FILES[@]}"; do
		if [ ! -f "$file" ]; then
			echo -e "${RED}✗${NC} $file missing"
			((missing++))
		else
			echo -e "${GREEN}✓${NC} $file"
		fi
	done

	# Check Docker daemon
	if ! docker info >/dev/null 2>&1; then
		echo -e "${RED}✗${NC} Docker daemon not running"
		((missing++))
	else
		echo -e "${GREEN}✓${NC} Docker daemon"
	fi

	# Check Docker Compose (both variants)
	if command -v docker-compose >/dev/null 2>&1; then
		echo -e "${GREEN}✓${NC} docker-compose"
	elif docker compose version >/dev/null 2>&1; then
		echo -e "${GREEN}✓${NC} docker compose"
	else
		echo -e "${RED}✗${NC} Neither 'docker-compose' nor 'docker compose' found"
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
			echo -e "${GREEN}✓${NC} $name"
			((healthy++))
		else
			echo -e "${YELLOW}⚠${NC} $name - not responding"
		fi
	done

	echo "$healthy/4 services responding"
	return 0
}

# Run dependency checks
echo "🔍 Checking dependencies..."
if check_deps; then
	echo -e "${GREEN}✓${NC} All dependencies OK"
else
	echo -e "${YELLOW}⚠${NC} Missing dependencies detected. Attempting to install..."

	# Try to install missing optional dependencies
	if install_missing_deps; then
		echo "Re-checking dependencies after installation..."
		if check_deps; then
			echo -e "${GREEN}✓${NC} Dependencies OK after installation"
		else
			echo -e "${RED}✗${NC} Some dependencies still missing"
			exit 1
		fi
	else
		echo -e "${RED}✗${NC} Failed to install missing dependencies"
		exit 1
	fi
fi

echo "🏥 Checking services..."
check_services
echo -e "${GREEN}✅${NC} Pre-flight complete"
