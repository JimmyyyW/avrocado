package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JimmyyyW/avrocado/internal/events"
)

type EventSaverModel struct {
	topic       string
	payload     string
	schemaID    int
	eventName   string
	focusedIdx  int
	saved       bool
	quit        bool
	err         string
	filePath    string
}

// NewEventSaver creates a new event saver model
func NewEventSaver(topic string, schemaID int, payload string) EventSaverModel {
	return EventSaverModel{
		topic:       topic,
		payload:     payload,
		schemaID:    schemaID,
		eventName:   "",
		focusedIdx:  0,
	}
}

func (m EventSaverModel) Init() tea.Cmd {
	return nil
}

func (m EventSaverModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.quit = true
			return m, nil
		case "enter":
			// Save event
			basePath := events.GetEventsDir()
			path, err := events.SaveEvent(basePath, m.topic, m.payload, m.schemaID, m.eventName)
			if err != nil {
				m.err = err.Error()
			} else {
				m.saved = true
				m.filePath = path
				m.quit = true
				return m, nil
			}
		default:
			// Handle text input
			if len(msg.String()) == 1 {
				m.eventName += msg.String()
			} else if msg.String() == "backspace" {
				if len(m.eventName) > 0 {
					m.eventName = m.eventName[:len(m.eventName)-1]
				}
			} else if msg.String() == "ctrl+u" {
				m.eventName = ""
			}
		}
	}
	return m, nil
}

func (m EventSaverModel) View() string {
	var s string
	s += lipgloss.NewStyle().Bold(true).Render("Save Event") + "\n\n"
	s += fmt.Sprintf("Topic: %s\n", m.topic)
	s += fmt.Sprintf("Schema ID: %d\n", m.schemaID)
	s += "\n"

	s += "Event Name (optional, defaults to timestamp):\n"
	s += "> " + m.eventName + "\n"

	s += "\n"
	if m.err != "" {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗ Error: "+m.err) + "\n\n"
	}

	s += lipgloss.NewStyle().Faint(true).Render("[enter] Save  [esc] Cancel") + "\n"

	return s
}

// Saved returns whether the event was saved
func (m EventSaverModel) Saved() bool {
	return m.saved
}

// FilePath returns the path to the saved file
func (m EventSaverModel) FilePath() string {
	return m.filePath
}
