# Avrocado

A terminal-based schema registry browser and message producer built with Go, Bubbletea, and Lipgloss.

## Features

- **Multi-Profile Configuration**: Manage multiple named configurations (local, staging, production, etc.)
- **YAML Configuration**: Store settings at `~/.config/avrocado/config.yaml`
- **Schema Registry Browsing**: Search and view Avro schemas with syntax highlighting
- **Message Production**: Edit and produce messages to Kafka topics
- **Event Persistence**: Save and load previously sent messages per topic
- **Authentication Support**:
  - Schema Registry: Basic auth or SASL/PLAIN
  - Kafka: PLAINTEXT or SASL_SSL (for Confluent Cloud)
- **Copy to Clipboard**: Quick copy of schemas and messages
- **External Editor**: Full-featured editing with `$EDITOR`
- **Clipboard Paste**: Paste long credentials directly into config forms

## Installation

```bash
go build -o avrocado .
```

## Configuration

### YAML Configuration File

Configuration is stored at `~/.config/avrocado/config.yaml`. On first launch, a default local configuration is created automatically.

Example configuration:
```yaml
default: local

configurations:
  local:
    name: "Local Development"
    schema_registry:
      url: http://localhost:8081
      auth_method: none
    kafka:
      bootstrap_servers: localhost:9092
      security_protocol: PLAINTEXT

  confluent-cloud:
    name: "Confluent Cloud"
    schema_registry:
      url: https://psrc-xxxxx.us-east-1.aws.confluent.cloud
      auth_method: basic
      api_key: YOUR_API_KEY
      api_secret: YOUR_API_SECRET
    kafka:
      bootstrap_servers: pkc-xxxxx.us-east-1.aws.confluent.cloud:9092
      security_protocol: SASL_SSL
      sasl_username: KAFKA_API_KEY
      sasl_password: KAFKA_API_SECRET
```

### Schema Registry Auth Methods
- `none`: No authentication
- `basic`: API Key and Secret (Confluent Cloud)
- `sasl`: SASL username and password

### Kafka Security Protocols
- `PLAINTEXT`: No security
- `SASL_SSL`: SASL/PLAIN with TLS (Confluent Cloud)

### Environment Variables (Backward Compatibility)

For backward compatibility, environment variables are still supported:

| Variable | Required | Description |
|----------|----------|-------------|
| `SCHEMA_REGISTRY_URL` | Yes | Schema registry URL |
| `SCHEMA_REGISTRY_API_KEY` | No | API key for authentication |
| `SCHEMA_REGISTRY_API_SECRET` | No | API secret for authentication |
| `KAFKA_BOOTSTRAP_SERVERS` | No | Kafka broker addresses (for message production) |
| `KAFKA_SASL_USERNAME` | No | SASL username |
| `KAFKA_SASL_PASSWORD` | No | SASL password |

## Usage

```bash
# Launch with default configuration
./avrocado

# Launch with interactive configuration selector
./avrocado --select-config
# or
./avrocado -s

# Legacy: Use environment variables (if no config file exists)
export SCHEMA_REGISTRY_URL=https://your-registry.confluent.cloud
export KAFKA_BOOTSTRAP_SERVERS=your-broker:9092
./avrocado
```

## Keybindings

### Configuration Selection
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate profiles |
| `Enter` | Select profile |
| `n` | Create new configuration |
| `e` | Edit selected configuration |
| `d` | Set as default |
| `q` | Quit |

### Browse Mode
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate subjects |
| `/` | Search subjects |
| `Enter` | View subject schema |
| `Tab` | Switch pane focus |
| `y` | Copy schema to clipboard |
| `q` | Quit |

### View Mode
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Scroll schema |
| `s` or `e` | Enter send mode |
| `E` | Open in `$EDITOR` |
| `y` | Copy schema to clipboard |
| `q` | Quit |

### Send Mode
| Key | Action |
|-----|--------|
| `Ctrl+S` | Send message to Kafka |
| `Ctrl+N` | Save current message as event |
| `Ctrl+L` | Load previously saved message |
| `y` | Copy message to clipboard |
| `Esc` | Cancel, return to view |

### Configuration Editor
| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Navigate fields |
| `Enter` | Next field / Save on last field |
| `Ctrl+U` | Clear field |
| `Cmd+V` / `Ctrl+Shift+V` | Paste from clipboard |
| `Esc` | Cancel |

## Event Persistence

Messages you send are automatically saved to `~/.config/avrocado/events/<topic>/`. You can:

- **Save**: Press `Ctrl+N` in send mode to save the current message with an optional name (defaults to timestamp)
- **Load**: Press `Ctrl+L` in send mode to browse and load previously sent messages
- **Format**: Events are stored as JSON files for easy inspection and editing

Events directory structure:
```
~/.config/avrocado/events/
└── user-topic/
    ├── 2024-01-20_15-30-45.json
    ├── test-user.json
    └── production-migration.json
```

## Local Development

### Start Test Environment

```bash
# Start Kafka and Schema Registry with test schemas
./test/harness.sh

# Create test configuration
source ./test/env.sh

# Run avrocado
./avrocado
```

### Stop Test Environment

```bash
./test/teardown.sh
```

## License

MIT
