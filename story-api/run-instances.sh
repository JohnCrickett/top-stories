#!/bin/bash

# Script to run multiple story-api instances with different filters
# Each instance consumes from the same Kafka topic but filters stories differently

set -e

echo "Building story-api..."
go build -o story-api

echo ""
echo "Starting multiple story-api instances with different filters..."
echo ""

# Function to run an instance in the background
run_instance() {
    local config=$1
    local name=$2
    
    echo "Starting $name with config: $config"
    ./story-api -config "$config" &
    PIDS+=($!)
    
    # Small delay between starts
    sleep 1
}

# Array to store process IDs
PIDS=()

# Start instances - uncomment the ones you want to run
run_instance "config.ask-hn.yaml" "Ask HN Instance (port 8081)"
run_instance "config.rust.yaml" "Rust Stories Instance (port 8082)"
run_instance "config.top.yaml" "High-Scoring Stories Instance (port 8083)"
run_instance "config.show-hn.yaml" "Show HN Instance (port 8084)"

echo ""
echo "All instances started. Test the APIs:"
echo ""
echo "# Get Ask HN stories"
echo "curl http://localhost:8081/stories"
echo ""
echo "# Get Rust stories"
echo "curl http://localhost:8082/stories"
echo ""
echo "# Get high-scoring stories"
echo "curl http://localhost:8083/stories"
echo ""
echo "# Get Show HN stories"
echo "curl http://localhost:8084/stories"
echo ""
echo "Press Ctrl+C to shut down all instances..."
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Shutting down instances..."
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
        fi
    done
    wait 2>/dev/null || true
    echo "All instances shut down"
}

# Register cleanup on exit
trap cleanup EXIT

# Wait for all background processes
if [ ${#PIDS[@]} -gt 0 ]; then
    wait
fi
