package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JimmyyyW/avrocado/internal/events"
)

type EventLoaderModel struct {
	topic       string
	files       []string
	selectedIdx int
	selectedEvent *events.Event
	quit        bool
	err         string
}

// NewEventLoader creates a new event loader model
func NewEventLoader(topic string) EventLoaderModel {
	m := EventLoaderModel{
		topic: topic,
	}

	// Load files for this topic
	basePath := events.GetEventsDir()
	files, err := events.ListEvents(basePath, topic)
	if err != nil {
		m.err = err.Error()
		return m
	}

	m.files = files
	return m
}

func (m EventLoaderModel) Init() tea.Cmd {
	return nil
}

func (m EventLoaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.quit = true
			return m, nil
		case "enter":
			// Load selected event
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.files) {
				basePath := events.GetEventsDir()
				filePath := events.GetEventPath(basePath, m.topic, m.files[m.selectedIdx])
				event, err := events.LoadEvent(filePath)
				if err != nil {
					m.err = err.Error()
				} else {
					m.selectedEvent = event
					m.quit = true
					return m, nil
				}
			}
		case "j", "down":
			if m.selectedIdx < len(m.files)-1 {
				m.selectedIdx++
			}
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		}
	}
	return m, nil
}

func (m EventLoaderModel) View() string {
	if len(m.files) == 0 {
		return "No saved events for topic: " + m.topic + "\n"
	}

	if m.err != "" {
		return "Error: " + m.err + "\n"
	}

	var s string
	s += lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Load Event - %s", m.topic)) + "\n\n"

	for i, file := range m.files {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}

		if i == m.selectedIdx {
			s += lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")).
				Bold(true).
				Render(prefix+file) + "\n"
		} else {
			s += prefix + file + "\n"
		}
	}

	s += "\n"
	s += lipgloss.NewStyle().Faint(true).Render("[enter] Load  [q] Quit") + "\n"

	return s
}

// LoadedEvent returns the loaded event
func (m EventLoaderModel) LoadedEvent() *events.Event {
	return m.selectedEvent
}

// Quit returns whether the user quit
func (m EventLoaderModel) Quit() bool {
	return m.quit
}
