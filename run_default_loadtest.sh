### RUN K6 LOAD TESTING FOR OUR BASELINE -- NO INSTRUMENTATION

#!/bin/bash

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
fi

# Start the Go server in background
echo "Starting HTTP server..."
go run main.go &
SERVER_PID=$!

# Start InfluxDB and Grafana
echo "Starting InfluxDB and Grafana..."
docker-compose up -d
echo "Waiting for services to initialize..."
docker-compose ps

# Wait for all services to start
sleep 5

export BASE_URL="http://host.docker.internal:8080/hello"

# Run load tests
docker-compose --profile load-test run --rm k6-load-test

echo ""
echo "Load test completed!"
echo ""
echo "View the Grafana dashboard at: http://localhost:3000"
echo ""
echo "Grafana is pre-configured with InfluxDB data source and a custom k6 dashboard."
echo "You should see the 'k6 Load Testing Results' dashboard automatically in the K6 folder."
echo "You can also import additional dashboards for k6:"
echo "1. Dashboard ID: 2587 (k6 Load Testing Results)"
echo "2. Dashboard ID: 4411 (k6 Load Testing Results by Endpoint)"
echo ""
echo "To stop the visualization services when finished:"
echo "docker-compose down && kill $SERVER_PID"