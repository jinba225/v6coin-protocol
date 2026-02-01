#!/bin/bash
set -e

NETWORK=${1:-testnet}
NODE_COUNT=${2:-5}

echo "Deploying V6Coin Network (Type: $NETWORK, Nodes: $NODE_COUNT)..."

# Start Docker Compose
docker-compose up -d v6coin-node

echo "✓ Network started"
echo ""
echo "Access points:"
echo "  - RPC API: http://localhost:9090"
echo "  - P2P Port: 38901"
echo "  - Grafana: http://localhost:3000 (admin/admin)"
echo "  - Prometheus: http://localhost:9091"
echo ""
echo "Check logs: docker-compose logs -f"
echo "Stop network: docker-compose down"
