#!/bin/bash

# Docker Rebuild Script
# This script will:
# 1. Stop current compose services
# 2. Rebuild images (using cache when possible)
# 3. Start services in detached mode
#
# Use './build.sh --clean' for full cleanup if needed

if [ "$1" == "--clean" ]; then
    echo "🧹 Full cleanup mode - removing unused Docker objects..."
    docker system prune -f
    echo "🗑️  Clearing build cache..."
    docker builder prune -f
fi

echo "🛑 Stopping current services..."
docker compose down

echo "🔨 Building and starting services..."
docker compose up --build -d

echo "✅ Build complete! Services are running in background."
echo "📋 Check status with: docker compose ps"
echo "📊 View logs with: docker compose logs -f"
echo ""
echo "💡 Tip: Use './build.sh --clean' for full cleanup if you have issues"