# Testing Requirements

## Overview

A local test harness for development and testing of avrocado. Spins up a minimal Confluent stack with pre-populated schemas for manual and automated testing.

## Test Infrastructure

### Docker Compose Stack

Minimal components required:
- **Zookeeper** - Required by Kafka
- **Kafka** - Single broker for local testing
- **Schema Registry** - The main component we're testing against

Excluded (not needed):
- ksqlDB
- Kafka Connect
- Control Center
- REST Proxy

### Test Schemas

The harness should register several schemas to test different scenarios:

#### 1. Simple Schema (`user-simple`)
Basic flat schema with a few fields to verify basic functionality.

#### 2. Deep Nested Schema (`order-nested`)
Schema with 3-4 levels of nesting to test:
- Viewer rendering of complex structures
- Indentation display
- Navigation within deeply nested JSON

#### 3. Wide Schema (`event-wide`)
Schema with 30+ fields to test:
- Scrolling behavior in the viewer
- Performance with large schemas
- List truncation if needed

#### 4. Schema with References (`payment-refs`)
Schema using Avro references/imports to test how referenced types render.

## Harness Scripts

### `test/harness.sh`
- Starts docker-compose stack
- Waits for Schema Registry to be healthy
- Registers all test schemas
- Prints connection info for avrocado

### `test/teardown.sh`
- Stops and removes containers
- Cleans up volumes

## Usage

```bash
# Start the test environment
./test/harness.sh

# Run avrocado against local stack
SCHEMA_REGISTRY_URL=http://localhost:8081 ./avrocado

# Tear down when done
./test/teardown.sh
```

## Test Scenarios

### Manual Testing Checklist
- [ ] Subjects list loads and displays all test schemas
- [ ] Search filters subjects correctly
- [ ] Selecting a subject loads and displays the schema
- [ ] Deep nested schema renders with proper indentation
- [ ] Wide schema scrolls correctly in viewer
- [ ] Copy to clipboard works (y key)
- [ ] Pane switching works (tab key)
- [ ] Quit works (q key)

### Future: Automated Tests
- Unit tests for registry client
- Integration tests against docker stack
- UI tests using bubbletea test utilities
