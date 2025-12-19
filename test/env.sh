#!/usr/bin/env bash
# Source this file to set environment variables for the test harness
# Usage: source ./test/env.sh

export SCHEMA_REGISTRY_URL=http://localhost:8081
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092

echo "Environment configured for test harness:"
echo "  SCHEMA_REGISTRY_URL=$SCHEMA_REGISTRY_URL"
echo "  KAFKA_BOOTSTRAP_SERVERS=$KAFKA_BOOTSTRAP_SERVERS"
echo ""
echo "Run: ./avrocado"
