### RUN K6 LOAD TESTING

#!/bin/bash

export BASE_URL="http://localhost:8080/hello"
export K6_INFLUXDB_ORGANIZATION=gopherconus
export K6_INFLUXDB_BUCKET=k6testing  
export K6_INFLUXDB_TOKEN=13NSkxbvAnGSbQIHAzWAQFsNVDXWHD94-NG2taWgmFCJ1FiLiFjjwNe_Vg37sKUc2Cn_kSWYMCR0egexhp3PRg==

# given no parameters, we will only run the default load test. to run specific tests, pass in one or more
# of the following: default, manual, orchestrion, ebpf.
# passing in "all" will run all four load tests in sequence.
if [[ $# -eq 0 ]]; then
    INSTRUMENTATION_ARRAY=("default")
elif [[ "$1" == "all" ]]; then
    INSTRUMENTATION_ARRAY=("default" "manual" "orchestrion" "ebpf")
else
    INSTRUMENTATION_ARRAY=("$@")
fi

# Check if docker and docker-compose are installed
if ! command -v docker >/dev/null 2>&1; then
  echo "Docker and docker-compose are required to run this script."
  echo "Please install Docker Desktop (macOS) or Docker Engine."
  exit 1
fi

# Check if k6 is installed and install if it isn't
if ! command -v k6 >/dev/null 2>&1; then
  echo "k6 is not installed. Installing now..."
  brew install k6
  go install go.k6.io/xk6@latest
fi

# Setup xk6 instance
xk6 build --with github.com/grafana/xk6-output-influxdb

# Start InfluxDB and Grafana
echo "Starting InfluxDB and Grafana..."
docker-compose up -d --remove-orphans
docker-compose ps

for INSTRUMENTATION in "${INSTRUMENTATION_ARRAY[@]}"; do
    echo ""
    echo "Testing instrumentation: $INSTRUMENTATION"
    echo "================================================"

    export INSTRUMENTATION=$INSTRUMENTATION

    # Start the Go server
    if [[ "$INSTRUMENTATION" == "orchestrion" ]]; then
        orchestrion go run main.go
    else
        go run main.go 
    fi

    # Initialize eBPF if we are using it
    if [[ "$INSTRUMENTATION" == "ebpf" ]]; then
        echo "Starting OTel eBPF..."
        docker-compose --profile ebpf-auto-instrumentation up -d --remove-orphans
    fi

    # Wait for all services to start
    sleep 3

    # Run load tests
    ./k6 run \
    --out xk6-influxdb=http://localhost:8086 \
    k6_loadtesting.js

    # Cleanup
    SERVER_PID=$(lsof -i :8080 -t)
    kill $SERVER_PID
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