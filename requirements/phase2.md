# Phase 2 Requirements

## Overview

Extend avrocado to produce Avro-encoded messages to Kafka topics. Users can edit message payloads directly in the TUI or use an external editor, then publish to the selected topic.

## Goals

- Enable producing test messages to Kafka topics
- Provide an intuitive editing experience for crafting message payloads
- Validate messages against the schema before sending
- Support both inline editing and external editor workflows

## Features

### Message Editor

#### Option A: Inline Editing (textarea)
- Convert schema viewer from read-only to editable mode
- Toggle between view mode and edit mode
- Syntax-aware editing with JSON support
- Schema provides a template/scaffold for the message

#### Option B: External Editor
- Press keybind to open schema in `$EDITOR` (vim, nvim, code, etc.)
- Temp file with `.json` extension for syntax highlighting
- On save/exit, read back the edited content
- Validate and send or return to editing

#### Option C: Hybrid (Recommended?)
- Inline editing for quick tweaks
- Keybind to "pop out" to external editor for complex edits
- Best of both worlds

### Message Production

- Connect to Kafka broker (reuse existing config or add `KAFKA_BOOTSTRAP_SERVERS`)
- Serialize message using Avro with Schema Registry
- Produce to the topic associated with the selected subject
- Support optional message key (string or Avro)
- Display success/failure feedback in status bar

### Workflow

1. User browses subjects, selects one
2. Schema loads in viewer pane
3. User presses `e` to enter edit mode (or `E` for external editor)
4. Viewer transforms into editor with schema as template
5. User modifies the JSON payload
6. User presses `Ctrl+Enter` or similar to send
7. Message is validated against schema
8. If valid, produce to Kafka; show success
9. If invalid, show error, remain in edit mode

## New Keybindings

| Key | Action |
|-----|--------|
| `e` | Enter edit mode (inline) |
| `E` | Open in external editor |
| `Ctrl+Enter` | Send message |
| `Esc` | Cancel edit / return to view mode |
| `Ctrl+v` | Validate message without sending |

## Technical Requirements

### Kafka Producer
- Use `confluent-kafka-go` or `segmentio/kafka-go`
- Configure bootstrap servers, security (SASL if needed)
- Avro serialization with Schema Registry

### Schema-to-Template Generation
- Generate a valid JSON template from Avro schema
- Handle defaults, nullables, nested records
- Populate with sensible placeholder values

### Validation
- Parse JSON input
- Validate against Avro schema before sending
- Clear error messages indicating which field failed

### External Editor Integration
- Detect `$EDITOR` or fall back to common editors
- Create temp file, spawn editor process
- Wait for editor exit, read modified content
- Clean up temp files

## Configuration

New environment variables:
```
KAFKA_BOOTSTRAP_SERVERS - Kafka broker addresses (e.g., localhost:9092)
KAFKA_SASL_USERNAME     - Optional SASL username
KAFKA_SASL_PASSWORD     - Optional SASL password
KAFKA_SECURITY_PROTOCOL - Optional (PLAINTEXT, SASL_SSL, etc.)
```

## UI Changes

### Mode Indicator
- Status bar shows current mode: `[VIEW]` / `[EDIT]` / `[SENDING...]`

### Editor Pane
- When in edit mode:
  - Border color changes to indicate editable
  - Cursor visible and movable
  - Standard text editing keys work (arrows, backspace, delete, etc.)
  - Line numbers helpful for large messages

### Validation Feedback
- Inline error highlighting (if possible with textarea)
- Or error message in status bar with line/field info

## Design Decisions

1. **Editor approach**: Hybrid
   - Inline editing for quick tweaks (`e` key)
   - External editor for complex edits (`E` key, uses `$EDITOR`)

2. **Subject-to-topic mapping**: Convention-based
   - Strip `-value` or `-key` suffix from subject name
   - Example: `user-simple-value` → `user-simple` topic

3. **Kafka client**: `segmentio/kafka-go`
   - Pure Go, no CGO required
   - Simpler build and deployment

## Open Questions

1. **Message keys**: Should we support Avro-encoded keys or just string keys initially?

2. **Multi-message support**: Should users be able to send multiple messages in sequence?

## Acceptance Criteria

- [ ] Can enter edit mode from schema view
- [ ] Schema provides editable template
- [ ] Can modify JSON payload in editor
- [ ] Validation errors shown clearly
- [ ] Can produce valid message to Kafka
- [ ] Success/failure feedback displayed
- [ ] Can cancel and return to view mode
- [ ] External editor workflow works (if implemented)

## Dependencies

- `github.com/linkedin/goavro/v2` - Avro encoding/decoding
- `github.com/segmentio/kafka-go` - Kafka producer (pure Go)
