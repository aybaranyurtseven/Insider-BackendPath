#!/bin/bash

# Stop script for Insider Backend

set -e

echo "Stopping Insider Backend..."

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "Error: docker-compose is not installed."
    exit 1
fi

# Stop services
echo "Stopping services..."
docker-compose down

echo "Services stopped successfully!"
echo ""
echo "To remove all data (including database): docker-compose down -v"
echo "To restart: ./scripts/start.sh"
