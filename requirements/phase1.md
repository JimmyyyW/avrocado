# Phase 1 Requirements

## Overview

A terminal-based schema registry browser built with Go. Connects to remote schema registries (e.g., Confluent) to browse, search, and inspect Avro schemas.

## Goals

- Provide a fast, keyboard-driven interface for exploring schema registries
- Enable quick discovery of topics and their associated schemas
- Allow easy inspection and copying of schema definitions

## Tech Stack

- **Language:** Go
- **TUI Framework:** Bubbletea
- **Styling:** Lipgloss

## Features

### Core Functionality
- Connect to a remote schema registry (Confluent Schema Registry API)
- List available topics/subjects
- Display schemas associated with each topic
- Search/filter topics and schemas

### Schema Viewer
- Render schema definitions in a readable format
- Syntax highlighting for Avro schema JSON
- Scrollable view for large schemas

### Keybindings
- Navigate topics/schemas list
- Search/filter mode
- Copy schema to clipboard
- Quit application

## Technical Requirements

- Support Confluent Schema Registry REST API
- Handle authentication (API key/secret)
- Graceful error handling for network issues
- Responsive UI that doesn't block on API calls

## Acceptance Criteria

- [ ] Can connect to a Confluent schema registry with credentials
- [ ] Displays list of all subjects/topics
- [ ] Can search/filter the subject list
- [ ] Selecting a subject shows its schema
- [ ] Schema renders in a scrollable pane
- [ ] Can copy schema to clipboard with a keybind
- [ ] Clean exit with `q` or `Ctrl+C`

## Phase 2 Preview

> **Note:** Phase 2 will introduce message production to topics. This has design implications for Phase 1:

- The schema viewer pane may need to support text editing (not just read-only display)
- Alternative: Open schemas in user's preferred `$EDITOR`
- Decision pending on inline editing vs external editor approach

This should be considered when architecting the schema viewer component to avoid major refactoring later.
