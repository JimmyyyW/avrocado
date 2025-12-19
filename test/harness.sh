#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_REGISTRY_URL="http://localhost:8081"

echo "=== Avrocado Test Harness ==="
echo ""

# Start docker compose
echo "Starting Confluent stack..."
docker compose -f "$SCRIPT_DIR/docker-compose.yml" up -d

# Wait for Schema Registry to be healthy
echo "Waiting for Schema Registry to be ready..."
max_attempts=30
attempt=0
until curl -s "$SCHEMA_REGISTRY_URL/subjects" > /dev/null 2>&1; do
    attempt=$((attempt + 1))
    if [ $attempt -ge $max_attempts ]; then
        echo "ERROR: Schema Registry failed to start after $max_attempts attempts"
        exit 1
    fi
    echo "  Attempt $attempt/$max_attempts - waiting..."
    sleep 2
done
echo "Schema Registry is ready!"
echo ""

# Register schemas
echo "Registering test schemas..."

register_schema() {
    local subject=$1
    local schema_file=$2

    # Read schema and escape for JSON
    local schema
    schema=$(cat "$schema_file" | jq -c '.')

    local payload
    payload=$(jq -n --arg schema "$schema" '{"schema": $schema}')

    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/vnd.schemaregistry.v1+json" \
        --data "$payload" \
        "$SCHEMA_REGISTRY_URL/subjects/${subject}/versions")

    local id
    id=$(echo "$response" | jq -r '.id // empty')

    if [ -n "$id" ]; then
        echo "  ✓ $subject (id: $id)"
    else
        echo "  ✗ $subject - Error: $response"
    fi
}

register_schema "user-simple-value" "$SCRIPT_DIR/schemas/user-simple.avsc"
register_schema "order-nested-value" "$SCRIPT_DIR/schemas/order-nested.avsc"
register_schema "event-wide-value" "$SCRIPT_DIR/schemas/event-wide.avsc"
register_schema "payment-value" "$SCRIPT_DIR/schemas/payment-value.avsc"

echo ""
echo "Creating Kafka topics..."

create_topic() {
    local topic=$1
    docker exec avrocado-kafka kafka-topics --create \
        --if-not-exists \
        --topic "$topic" \
        --bootstrap-server localhost:9092 \
        --partitions 1 \
        --replication-factor 1 \
        > /dev/null 2>&1 && echo "  ✓ $topic" || echo "  ✓ $topic (exists)"
}

create_topic "user-simple"
create_topic "order-nested"
create_topic "event-wide"
create_topic "payment"

echo ""
echo "=== Test Environment Ready ==="
echo ""
echo "Schema Registry URL: $SCHEMA_REGISTRY_URL"
echo ""
echo "Run avrocado with:"
echo "  SCHEMA_REGISTRY_URL=$SCHEMA_REGISTRY_URL ./avrocado"
echo ""
echo "To tear down:"
echo "  ./test/teardown.sh"
