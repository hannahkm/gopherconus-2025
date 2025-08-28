### RUN K6 LOAD TESTING

#!/bin/bash

set -e

# Store PID for cleanup
SERVER_PID=""

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up..."
    if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
        echo "Stopping Go server (PID: $SERVER_PID)"
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
}

# Set up signal handling
trap cleanup EXIT INT TERM

export BASE_URL="http://localhost:8080/hello"
export K6_INFLUXDB_ORGANIZATION=gopherconus
export K6_INFLUXDB_BUCKET=k6testing
export K6_INFLUXDB_TOKEN=13NSkxbvAnGSbQIHAzWAQFsNVDXWHD94-NG2taWgmFCJ1FiLiFjjwNe_Vg37sKUc2Cn_kSWYMCR0egexhp3PRg==

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
    MODE="run"  # Default to run mode for backward compatibility
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
            default|manual|orchestrion|ebpf)
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
        # Run pre-flight checks for start mode
        if [[ "$SKIP_PREFLIGHT" != "true" ]]; then
            if ! ./pre-flight-checks.sh; then
                echo "❌ Pre-flight checks failed. Aborting."
                exit 1
            fi
        else
            echo "⚠️  Skipping pre-flight checks"
        fi
        
        echo "🚀 Starting Docker services..."
        docker-compose up -d --remove-orphans
        echo ""
        echo "📊 Service status:"
        docker-compose ps
        echo ""
        echo "✅ Services started successfully!"
        echo "   - Grafana: http://localhost:3000"
        echo "   - InfluxDB: http://localhost:8086"
        echo "   - Jaeger: http://localhost:16686"
        echo "   - Prometheus: http://localhost:9090"
        exit 0
        ;;
    "stop")
        echo "🛑 Stopping all Docker services..."
        docker-compose down
        echo "✅ All services stopped"
        exit 0
        ;;
    "run")
        # Run pre-flight checks for run mode
        if [[ "$SKIP_PREFLIGHT" != "true" ]]; then
            if ! ./pre-flight-checks.sh; then
                echo "❌ Pre-flight checks failed. Aborting."
                exit 1
            fi
        else
            echo "⚠️  Skipping pre-flight checks"
        fi
        # Continue with test execution
        ;;
esac

# Check if k6 binary exists, build if needed
K6_BINARY="./k6"
if [[ ! -f "$K6_BINARY" ]]; then
    echo "Building k6 with InfluxDB extension..."
    
    # Install xk6 if needed
    if ! command -v xk6 >/dev/null 2>&1; then
        echo "Installing xk6..."
        go install go.k6.io/xk6/cmd/xk6@latest
    fi
    
    # Build k6 with extensions
    xk6 build --with github.com/grafana/xk6-output-influxdb
else
    echo "Using existing k6 binary: $K6_BINARY"
fi

echo "Starting InfluxDB, Grafana, and OTel Collector..."
docker-compose up -d --remove-orphans
docker-compose ps

# Test counter for inter-test delays
TEST_COUNT=0

for INSTRUMENTATION in "${TESTS_TO_RUN[@]}"; do
    # Add delay between tests (except for the first one)
    if [[ $TEST_COUNT -gt 0 ]]; then
        echo ""
        echo "⏳ Pausing 5 seconds between tests for better graph separation..."
        sleep 5
        echo ""
    fi
    
    echo "Testing instrumentation: $INSTRUMENTATION"
    echo "================================================"
    
    ((TEST_COUNT++))

    export INSTRUMENTATION=$INSTRUMENTATION

    # If we are instrumenting, setup the OTel Collector
    if [[ "$INSTRUMENTATION" != "default" ]]; then
        echo "Starting OpenTelemetry Collector for Datadog..."
        docker-compose --profile with-instrumentation up -d otel-collector

        export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
    fi

    # Start the Go server in background and capture PID
    if [[ "$INSTRUMENTATION" == "orchestrion" ]]; then
        orchestrion go run main.go &
        SERVER_PID=$!
    elif [[ "$INSTRUMENTATION" == "ebpf" ]]; then
        # Build and run binary for eBPF instrumentation
        echo "Building Go binary for eBPF instrumentation..."
        go build -o gopherconus-server main.go
        ./gopherconus-server &
        SERVER_PID=$!
    else
        go run main.go &
        SERVER_PID=$!
    fi
    
    echo "Started Go server with PID: $SERVER_PID"

    # Initialize eBPF if we are using it
    if [[ "$INSTRUMENTATION" == "ebpf" ]]; then
        echo "Starting OTel eBPF..."
        docker-compose --profile ebpf-auto-instrumentation up -d --remove-orphans
    fi

    # Wait for services to start
    echo "Waiting for services to be ready..."
    sleep 5
    
    # Basic health check for the Go server
    for i in {1..12}; do
        if curl -s http://localhost:8080/hello >/dev/null 2>&1; then
            echo "✅ Go server is ready"
            break
        fi
        echo "⏳ Waiting for Go server... ($i/12)"
        sleep 2
    done

    # Run load tests
    echo "🚀 Starting k6 load test..."
    "$K6_BINARY" run \
        --out xk6-influxdb=http://localhost:8086 \
        k6_loadtesting.js

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
echo "docker-compose down"
