package events

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Event represents a saved message event
type Event struct {
	Topic     string    `json:"topic"`
	SchemaID  int       `json:"schema_id"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
	Name      string    `json:"name"`
}

// SaveEvent saves an event to disk
func SaveEvent(baseDir, topic, payload string, schemaID int, name string) (string, error) {
	// Create events directory structure
	eventDir := filepath.Join(baseDir, "events", topic)
	if err := os.MkdirAll(eventDir, 0700); err != nil {
		return "", fmt.Errorf("creating event directory: %w", err)
	}

	// Use provided name or timestamp
	filename := name
	if name == "" {
		filename = time.Now().Format("2006-01-02_15-04-05")
	}

	// Ensure filename is valid
	if filename == "" {
		filename = "event"
	}

	// Add .json extension if not present
	if filepath.Ext(filename) != ".json" {
		filename += ".json"
	}

	// Check if file exists, add counter if it does
	filePath := filepath.Join(eventDir, filename)
	counter := 1
	for {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			break
		}
		// File exists, modify filename
		ext := filepath.Ext(filename)
		base := filename[:len(filename)-len(ext)]
		newFilename := fmt.Sprintf("%s_%d%s", base, counter, ext)
		filePath = filepath.Join(eventDir, newFilename)
		counter++
	}

	// Create event
	event := Event{
		Topic:     topic,
		SchemaID:  schemaID,
		Payload:   payload,
		Timestamp: time.Now(),
		Name:      filepath.Base(filePath),
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling event: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("writing event file: %w", err)
	}

	return filePath, nil
}

// LoadEvent loads an event from disk
func LoadEvent(filePath string) (*Event, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading event file: %w", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("parsing event file: %w", err)
	}

	return &event, nil
}

// ListEvents lists all events for a topic
func ListEvents(baseDir, topic string) ([]string, error) {
	eventDir := filepath.Join(baseDir, "events", topic)

	// Check if directory exists
	if _, err := os.Stat(eventDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(eventDir)
	if err != nil {
		return nil, fmt.Errorf("reading event directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			files = append(files, entry.Name())
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		pathI := filepath.Join(eventDir, files[i])
		pathJ := filepath.Join(eventDir, files[j])

		infoI, _ := os.Stat(pathI)
		infoJ, _ := os.Stat(pathJ)

		if infoI == nil || infoJ == nil {
			return false
		}

		return infoI.ModTime().After(infoJ.ModTime())
	})

	return files, nil
}

// GetEventPath returns the full path to an event file
func GetEventPath(baseDir, topic, filename string) string {
	return filepath.Join(baseDir, "events", topic, filename)
}

// GetEventsDir returns the base events directory
func GetEventsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "avrocado")
	}
	return filepath.Join(home, ".config", "avrocado")
}
