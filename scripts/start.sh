#!/bin/bash

# Start script for Insider Backend

set -e

echo "Starting Insider Backend..."

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "Error: docker-compose is not installed. Please install docker-compose first."
    exit 1
fi

# Create necessary directories
mkdir -p logs
mkdir -p data/postgres
mkdir -p data/redis

# Copy environment file if it doesn't exist
if [ ! -f .env ]; then
    if [ -f env.example ]; then
        cp env.example .env
        echo "Created .env file from env.example. Please review and update the configuration."
    else
        echo "Warning: No .env file found and no env.example to copy from."
    fi
fi

# Start services
echo "Starting services with docker-compose..."
docker-compose up -d

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 10

# Check service health
echo "Checking service health..."

# Check PostgreSQL
echo -n "PostgreSQL: "
if docker-compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; then
    echo "✅ Ready"
else
    echo "❌ Not ready"
fi

# Check Redis
echo -n "Redis: "
if docker-compose exec -T redis redis-cli ping >/dev/null 2>&1; then
    echo "✅ Ready"
else
    echo "❌ Not ready"
fi

# Check Application
echo -n "Application: "
if curl -f http://localhost:8080/api/v1/health >/dev/null 2>&1; then
    echo "✅ Ready"
else
    echo "❌ Not ready"
fi

echo ""
echo "🚀 Insider Backend is starting up!"
echo ""
echo "Services:"
echo "  Application: http://localhost:8080"
echo "  API Health:  http://localhost:8080/api/v1/health"
echo "  Grafana:     http://localhost:3000 (admin/admin)"
echo "  Prometheus:  http://localhost:9090"
echo ""
echo "To view logs: docker-compose logs -f"
echo "To stop:      docker-compose down"
