### RUN K6 LOAD TESTING

#!/bin/bash

set -e

# Enable debug mode if DEBUG environment variable is set
if [[ "${DEBUG:-}" == "true" ]]; then
	set -x
fi

# Logging function
log() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

log_error() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] ❌ ERROR: $1" >&2
}

log_warning() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  WARNING: $1" >&2
}

log_info() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] ℹ️  INFO: $1"
}

# Store PID and PORT for cleanup
SERVER_PID=""
SERVER_PORT=""
EBPF_SERVICES_STARTED=false

# Cleanup function
cleanup() {
	# Only clean up if we're in run mode, not start/stop mode
	if [[ "$MODE" == "run" ]]; then
		echo "🧹 Cleaning up..."
		if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
			echo "Stopping Go server (PID: $SERVER_PID, PORT: $SERVER_PORT)"
			kill "$SERVER_PID" 2>/dev/null || true
			wait "$SERVER_PID" 2>/dev/null || true

			# Wait for port to be released
			if [[ -n "$SERVER_PORT" ]]; then
				echo "Waiting for port $SERVER_PORT to be released..."
				for i in {1..10}; do
					if ! lsof -i :"$SERVER_PORT" >/dev/null 2>&1; then
						echo "✅ Port $SERVER_PORT released"
						break
					fi
					sleep 1
				done
			fi
		fi
	fi
}

# Set up signal handling
trap cleanup EXIT INT TERM

# BASE_URL will be set dynamically after server starts
export K6_INFLUXDB_ORGANIZATION=gopherconus
export K6_INFLUXDB_BUCKET=k6testing
export K6_INFLUXDB_TOKEN=13NSkxbvAnGSbQIHAzWAQFsNVDXWHD94-NG2taWgmFCJ1FiLiFjjwNe_Vg37sKUc2Cn_kSWYMCR0egexhp3PRg==

# Ensure Go bin directory is in PATH
GO_BIN_DIR=$(go env GOPATH)/bin
if [[ -z "$GO_BIN_DIR" ]]; then
	GO_BIN_DIR="$HOME/go/bin"
fi
export PATH="$GO_BIN_DIR:$PATH"

# Docker Compose command detection (supports both docker-compose and docker compose)
if command -v docker-compose >/dev/null 2>&1; then
	DOCKER_COMPOSE="docker-compose"
elif docker compose version >/dev/null 2>&1; then
	DOCKER_COMPOSE="docker compose"
else
	log_error "Neither 'docker-compose' nor 'docker compose' found"
	exit 1
fi

# Parse command mode
MODE=""
SKIP_PREFLIGHT=false
TESTS_TO_RUN=()

# Determine mode
if [[ $# -eq 0 ]]; then
	MODE="run"
	TESTS_TO_RUN=("default")
elif [[ "$1" == "start" || "$1" == "stop" ]]; then
	MODE="$1"
	shift
elif [[ "$1" == "run" ]]; then
	MODE="run"
	shift
else
	MODE="run"
fi

# Process remaining arguments for start, stop, and run modes
if [[ "$MODE" == "start" || "$MODE" == "run" ]]; then
	while [[ $# -gt 0 ]]; do
		case $1 in
		--skip-preflight)
			SKIP_PREFLIGHT=true
			shift
			;;
		all)
			if [[ "$MODE" == "run" ]]; then
				TESTS_TO_RUN=("default" "manual" "orchestrion" "ebpf")
				shift
			else
				echo "Test arguments only valid for run mode: $1"
				exit 1
			fi
			;;
		default | manual | orchestrion | ebpf)
			if [[ "$MODE" == "run" ]]; then
				TESTS_TO_RUN+=("$1")
				shift
			else
				echo "Test arguments only valid for run mode: $1"
				exit 1
			fi
			;;
		*)
			echo "Unknown argument: $1"
			echo "Usage:"
			echo "  $0 start [--skip-preflight]                # Start Docker services"
			echo "  $0 stop                                     # Stop Docker services"
			echo "  $0 run [--skip-preflight] [tests...]       # Run load tests"
			echo "  $0 [--skip-preflight] [tests...]           # Run load tests (default)"
			echo ""
			echo "Tests: default, manual, orchestrion, ebpf, all"
			exit 1
			;;
		esac
	done

	# Default to 'default' if no tests specified
	if [[ ${#TESTS_TO_RUN[@]} -eq 0 ]]; then
		TESTS_TO_RUN=("default")
	fi
fi

# Handle different modes
case "$MODE" in
"start")
	# Skip service checks for start mode - we're about to start them
	if [[ "$SKIP_PREFLIGHT" != "true" ]]; then
		# Only check dependencies, not services
		if ! command -v docker >/dev/null 2>&1; then
			log_error "Docker not found"
			exit 1
		fi
		if ! command -v docker-compose >/dev/null 2>&1 && ! docker compose version >/dev/null 2>&1; then
			log_error "Neither 'docker-compose' nor 'docker compose' found"
			exit 1
		fi
	fi

	echo "🚀 Starting Docker services..."
	if ! $DOCKER_COMPOSE up -d --remove-orphans; then
		log_error "Failed to start Docker services"
		exit 1
	fi

	echo "⏳ Waiting for services to initialize..."
	sleep 5

	echo "📊 Service status:"
	$DOCKER_COMPOSE ps
	echo "✅ Services started successfully!"
	echo "   - Grafana: http://localhost:3000"
	echo "   - InfluxDB: http://localhost:8086"
	echo "   - Jaeger: http://localhost:16686"
	echo "   - Prometheus: http://localhost:9090"
	exit 0
	;;
"stop")
	echo "🛑 Stopping all Docker services..."
	if ! $DOCKER_COMPOSE down; then
		log_error "Failed to stop Docker services"
		exit 1
	fi
	echo "✅ All services stopped"
	exit 0
	;;
"run")
	# Run pre-flight checks for run mode
	if [[ "$SKIP_PREFLIGHT" != "true" ]]; then
		if ! ./pre-flight-checks.sh; then
			log_error "Pre-flight checks failed. Aborting."
			exit 1
		fi
	fi
	# Continue with test execution
	;;
*)
	log_error "Unknown mode: $MODE"
	exit 1
	;;
esac

# Check if k6 binary exists for current platform, build if needed
OS=$(uname -s)
ARCH=$(uname -m)
# Normalize architecture names for consistency
case "$ARCH" in
"x86_64") ARCH="amd64" ;;
"arm64" | "aarch64") ARCH="arm64" ;;
"i386" | "i686") ARCH="386" ;;
esac
K6_BINARY="./k6-${OS}-${ARCH}"

if [[ ! -f "$K6_BINARY" ]]; then
	echo "Building k6 with InfluxDB extension..."

	# Install xk6 if needed
	if ! command -v xk6 >/dev/null 2>&1; then
		echo "Installing xk6..."
		if ! go install go.k6.io/xk6/cmd/xk6@latest; then
			log_error "Failed to install xk6"
			exit 1
		fi

		# Verify xk6 is now available
		if ! command -v xk6 >/dev/null 2>&1; then
			log_error "xk6 still not found after installation"
			exit 1
		fi
	fi

	# Install orchestrion if needed
	if ! command -v orchestrion >/dev/null 2>&1; then
		echo "Installing orchestrion..."
		if ! go install github.com/DataDog/orchestrion@latest; then
			log_error "Failed to install orchestrion"
			exit 1
		fi

		# Verify orchestrion is now available
		if ! command -v orchestrion >/dev/null 2>&1; then
			log_error "orchestrion still not found after installation"
			exit 1
		fi
	fi

	# Build k6 with extensions
	echo "Building k6 with InfluxDB extension for ${OS}-${ARCH}..."
	if ! xk6 build --with github.com/grafana/xk6-output-influxdb; then
		log_error "Failed to build k6 with extensions"
		echo "You may need to install build tools:"
		echo "  - macOS: xcode-select --install"
		echo "  - Linux: apt-get install build-essential (Ubuntu) or yum groupinstall 'Development Tools' (RHEL/CentOS)"
		exit 1
	fi

	# Rename the default k6 binary to platform-specific name
	if [[ -f "./k6" ]]; then
		mv "./k6" "$K6_BINARY"
		echo "Created platform-specific k6 binary: $K6_BINARY"
	fi

	# Verify the binary was created
	if [[ ! -f "$K6_BINARY" ]]; then
		log_error "k6 binary not found after build"
		exit 1
	fi
else
	echo "Using existing k6 binary: $K6_BINARY ($(ls -lh "$K6_BINARY" | awk '{print $5}'))"
fi

echo "Starting InfluxDB, Grafana, and OTel Collector..."
if ! $DOCKER_COMPOSE up -d --remove-orphans; then
	log_error "Failed to start Docker services"
	exit 1
fi
$DOCKER_COMPOSE ps

# Test counter for inter-test delays
TEST_COUNT=0

# Check critical files exist
CRITICAL_FILES=("main.go" "k6_loadtesting.js" "docker-compose.yml")
for file in "${CRITICAL_FILES[@]}"; do
	if [[ ! -f "$file" ]]; then
		log_error "Critical file missing: $file"
		exit 1
	fi
done

for INSTRUMENTATION in "${TESTS_TO_RUN[@]}"; do
	echo "=== Testing instrumentation: $INSTRUMENTATION ==="

	# Add delay between tests (except for the first one)
	if [[ $TEST_COUNT -gt 0 ]]; then
		echo "⏳ Pausing 5 seconds between tests for better graph separation..."
		sleep 5
	fi

	# Safer arithmetic that won't fail with set -e
	TEST_COUNT=$((TEST_COUNT + 1))
	export INSTRUMENTATION=$INSTRUMENTATION

	# Set the service name for each instrumentation type
	case "$INSTRUMENTATION" in
	"manual")
		export OTEL_SERVICE_NAME="gopherconus-manual"
		;;
	"orchestrion")
		export OTEL_SERVICE_NAME="gopherconus-orchestrion"
		;;
	"ebpf")
		export OTEL_SERVICE_NAME="gopherconus-ebpf"
		;;
	"default")
		export OTEL_SERVICE_NAME="gopherconus-default"
		;;
	*)
		log_error "Unknown instrumentation type: $INSTRUMENTATION"
		exit 1
		;;
	esac

	# If we are instrumenting, setup the OTel Collector
	if [[ "$INSTRUMENTATION" != "default" ]]; then
		echo "Starting OpenTelemetry Collector for Datadog..."
		if $DOCKER_COMPOSE up -d otel-collector 2>/dev/null; then
			echo "✅ OTel Collector started"
		else
			echo "⚠️ OTel Collector failed to start!"
		fi
		export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
	fi

	# Start the Go server in background and capture PID and PORT
	echo "Starting Go server for $INSTRUMENTATION instrumentation..."

	# Create temporary output file
	SERVER_OUTPUT=$(mktemp)

	if [[ "$INSTRUMENTATION" == "orchestrion" ]]; then
		if ! command -v orchestrion >/dev/null 2>&1; then
			log_error "orchestrion command not found"
			exit 1
		fi
		GOTOOLCHAIN=go1.24.1 orchestrion go run main.go >"$SERVER_OUTPUT" 2>&1 &
		SERVER_PID=$!
	elif [[ "$INSTRUMENTATION" == "ebpf" ]]; then
		# Build and run binary for eBPF instrumentation
		echo "Building Go binary for eBPF instrumentation..."
		if ! GOTOOLCHAIN=go1.24.1 go build -o gopherconus-server main.go; then
			log_error "Failed to build Go binary for eBPF"
			exit 1
		fi
		./gopherconus-server >"$SERVER_OUTPUT" 2>&1 &
		SERVER_PID=$!
	else
		GOTOOLCHAIN=go1.24.1 go run main.go >"$SERVER_OUTPUT" 2>&1 &
		SERVER_PID=$!
	fi

	# Verify the process is still running
	sleep 1
	if ! kill -0 "$SERVER_PID" 2>/dev/null; then
		log_error "Server process $SERVER_PID died immediately. Output:"
		cat "$SERVER_OUTPUT" 2>/dev/null || echo "No output captured"
		cleanup
		exit 1
	fi

	# Wait for server to output the port
	echo "Waiting for server to start and output port..."
	for i in {1..30}; do
		# Check if process is still alive
		if ! kill -0 "$SERVER_PID" 2>/dev/null; then
			log_error "Server process $SERVER_PID died during startup (attempt $i/30). Output:"
			cat "$SERVER_OUTPUT" 2>/dev/null || echo "No output captured"
			cleanup
			exit 1
		fi

		# Check for port in output
		if [[ -f "$SERVER_OUTPUT" ]] && grep -q "SERVER_PORT=" "$SERVER_OUTPUT"; then
			SERVER_PORT=$(grep "SERVER_PORT=" "$SERVER_OUTPUT" | cut -d= -f2 | head -1)
			echo "✅ Server started with PID: $SERVER_PID, PORT: $SERVER_PORT"
			break
		fi

		sleep 1
	done

	if [[ -z "$SERVER_PORT" ]]; then
		log_error "Failed to get server port after 30 attempts"
		echo "❌ Failed to get server port. Server output:"
		cat "$SERVER_OUTPUT" 2>/dev/null || echo "No output file found"
		echo "--- Checking for 'SERVER_PORT=' pattern ---"
    	grep "SERVER_PORT=" "$SERVER_OUTPUT" 2>/dev/null || echo "Pattern not found"
		cleanup
		exit 1
	fi

	# Set dynamic BASE_URL
	export BASE_URL="http://localhost:$SERVER_PORT/hello"
	echo "Using BASE_URL: $BASE_URL"

	# Clean up temp output file
	rm -f "$SERVER_OUTPUT"

	# Initialize eBPF if we are using it
	if [[ "$INSTRUMENTATION" == "ebpf" ]]; then
		echo "Starting OTel eBPF..."
		$DOCKER_COMPOSE --profile with-auto-instrumentation up -d --remove-orphans
		EBPF_SERVICES_STARTED=true
	fi

	# Wait for services to start
	echo "Waiting for services to be ready..."
	sleep 5

	# Basic health check for the Go server
	echo "Testing Go server connectivity..."
	if curl -s "$BASE_URL" >/dev/null 2>&1; then
		echo "✅ Go server is ready at $BASE_URL"
	else
		echo "⚠️ Go server not responding immediately, but continuing..."
	fi

	# Run load tests
	echo "🚀 Starting k6 load test..."

	# Build InfluxDB URL for the xk6-influxdb extension
	INFLUXDB_URL="http://localhost:8086?bucket=${K6_INFLUXDB_BUCKET}&organization=${K6_INFLUXDB_ORGANIZATION}&token=${K6_INFLUXDB_TOKEN}"
	echo "Running k6 with InfluxDB extension..."
	"$K6_BINARY" run --out "xk6-influxdb=${INFLUXDB_URL}" k6_loadtesting.js

	# Stop eBPF services if they were started for this test
	if [[ "$INSTRUMENTATION" == "ebpf" && "$EBPF_SERVICES_STARTED" == "true" ]]; then
		echo "Stopping eBPF auto-instrumentation services..."
		$DOCKER_COMPOSE stop go-auto
		EBPF_SERVICES_STARTED=false
	fi

	# Cleanup will be handled by trap
done

echo ""
echo "Load test completed!"
echo ""
echo "View the Grafana dashboard at: http://localhost:3000"
echo ""
echo "You should see the 'k6 Load Testing Results' dashboard automatically in the K6 folder."
echo ""
echo "To stop the visualization services when finished:"
echo "$DOCKER_COMPOSE down"
