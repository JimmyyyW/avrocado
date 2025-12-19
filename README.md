# Avrocado

A terminal-based schema registry browser and message producer built with Go, Bubbletea, and Lipgloss.

## Features

- Browse and search schema registry subjects
- View Avro schemas with syntax highlighting
- Edit and produce messages to Kafka topics
- Copy schemas/messages to clipboard
- External editor support (`$EDITOR`)

## Installation

```bash
go build -o avrocado .
```

## Configuration

### Environment Variables

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
# With environment variables
SCHEMA_REGISTRY_URL=https://your-registry.confluent.cloud ./avrocado

# Or export them first
export SCHEMA_REGISTRY_URL=https://your-registry.confluent.cloud
export KAFKA_BOOTSTRAP_SERVERS=your-broker:9092
./avrocado
```

## Keybindings

| Key | Mode | Action |
|-----|------|--------|
| `j/k` or `↑/↓` | All | Navigate list |
| `/` | Browse | Search subjects |
| `Enter` | Browse | Select subject |
| `Tab` | All | Switch pane focus |
| `s` or `e` | View | Enter send mode |
| `E` | View | Open in `$EDITOR` |
| `Ctrl+S` | Send | Send message to Kafka |
| `Esc` | Send | Cancel, return to view |
| `y` | All | Copy to clipboard |
| `q` | All | Quit |

## Local Development

### Start Test Environment

```bash
# Start Kafka and Schema Registry with test schemas
./test/harness.sh

# Set environment variables
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
