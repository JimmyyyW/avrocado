#!/usr/bin/env bash
# Create configuration for the test harness
# Usage: source ./test/env.sh

# Create config directory
mkdir -p ~/.config/avrocado

# Create test configuration file
cat > ~/.config/avrocado/config.yaml <<'EOF'
default: local

configurations:
  local:
    name: "Local Test Environment"
    schema_registry:
      url: http://localhost:8081
      auth_method: none
    kafka:
      bootstrap_servers: localhost:9092
      security_protocol: PLAINTEXT
EOF

echo "Test configuration created at: ~/.config/avrocado/config.yaml"
echo ""
echo "Run: ./avrocado"
echo "Or: ./avrocado --select-config"
