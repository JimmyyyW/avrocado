package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JimmyyyW/avrocado/internal/avro"
	"github.com/JimmyyyW/avrocado/internal/config"
	"github.com/JimmyyyW/avrocado/internal/editor"
	"github.com/JimmyyyW/avrocado/internal/kafka"
	"github.com/JimmyyyW/avrocado/internal/registry"
)

type pane int

const (
	listPane pane = iota
	viewerPane
)

type state int

const (
	stateLoading state = iota
	stateBrowsing
	stateSearching
	stateViewing
	stateEditing
	stateSending
)

type Model struct {
	client   *registry.Client
	producer *kafka.Producer

	subjects         []string
	filteredSubjects []string
	selectedIndex    int
	selectedSubject  string
	currentSchema    string
	rawSchema        string // Original schema JSON for validation
	schemaID         int

	searchInput textinput.Model
	editor      textarea.Model
	help        help.Model

	focusedPane pane
	state       state

	width  int
	height int

	err        error
	statusMsg  string
	copyNotify string
}

type subjectsLoadedMsg struct {
	subjects []string
	err      error
}

type schemaLoadedMsg struct {
	schema *registry.SchemaResponse
	err    error
}

type messageSentMsg struct {
	topic string
	err   error
}

type externalEditorMsg struct {
	content string
	err     error
}

func NewModel(client *registry.Client, producer *kafka.Producer) Model {
	ti := textinput.New()
	ti.Placeholder = "Search subjects..."
	ti.CharLimit = 100

	ta := textarea.New()
	ta.Placeholder = "Schema will appear here..."
	ta.ShowLineNumbers = true
	ta.SetWidth(40)
	ta.SetHeight(20)

	h := help.New()
	h.ShowAll = false

	return Model{
		client:           client,
		producer:         producer,
		subjects:         []string{},
		filteredSubjects: []string{},
		searchInput:      ti,
		editor:           ta,
		help:             h,
		focusedPane:      listPane,
		state:            stateLoading,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadSubjects
}

func (m Model) loadSubjects() tea.Msg {
	subjects, err := m.client.ListSubjects()
	return subjectsLoadedMsg{subjects: subjects, err: err}
}

func (m Model) loadSchema(subject string) tea.Cmd {
	return func() tea.Msg {
		schema, err := m.client.GetLatestSchema(subject)
		return schemaLoadedMsg{schema: schema, err: err}
	}
}

func (m Model) sendMessage() tea.Cmd {
	return func() tea.Msg {
		if m.producer == nil {
			return messageSentMsg{err: fmt.Errorf("Kafka not configured")}
		}

		// Validate and encode
		binary, err := avro.ValidateAndEncode(m.rawSchema, m.editor.Value())
		if err != nil {
			return messageSentMsg{err: err}
		}

		// Determine topic from subject
		topic := config.SubjectToTopic(m.selectedSubject)

		// Produce message
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = m.producer.Produce(ctx, topic, m.schemaID, nil, binary)
		return messageSentMsg{topic: topic, err: err}
	}
}

func (m Model) openExternalEditor() tea.Cmd {
	return func() tea.Msg {
		content, err := editor.Open(m.editor.Value())
		return externalEditorMsg{content: content, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.SetWidth(m.width/2 - 6)
		m.editor.SetHeight(m.height - 10)
		return m, nil

	case subjectsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateBrowsing
			return m, nil
		}
		m.subjects = msg.subjects
		m.filteredSubjects = msg.subjects
		m.state = stateBrowsing
		m.statusMsg = fmt.Sprintf("Loaded %d subjects", len(m.subjects))
		return m, nil

	case schemaLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.rawSchema = msg.schema.Schema
		m.schemaID = msg.schema.ID
		m.currentSchema = registry.PrettyPrintSchema(msg.schema.Schema)
		m.editor.SetValue(m.currentSchema)
		m.state = stateViewing
		m.focusedPane = viewerPane
		m.statusMsg = fmt.Sprintf("[VIEW] %s (v%d)", msg.schema.Subject, msg.schema.Version)
		return m, nil

	case messageSentMsg:
		m.state = stateEditing
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = "[EDIT] Send failed"
		} else {
			m.statusMsg = fmt.Sprintf("Message sent to %s!", msg.topic)
			m.copyNotify = fmt.Sprintf("Message sent to %s!", msg.topic)
		}
		return m, nil

	case externalEditorMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.editor.SetValue(msg.content)
		}
		m.state = stateEditing
		m.statusMsg = "[EDIT] " + m.selectedSubject
		return m, nil

	case tea.KeyMsg:
		m.copyNotify = ""
		m.err = nil

		// Handle state-specific input
		switch m.state {
		case stateSearching:
			return m.handleSearchInput(msg)
		case stateEditing:
			return m.handleEditMode(msg)
		case stateSending:
			// Ignore input while sending
			return m, nil
		}

		// Global keybindings
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "/":
			m.state = stateSearching
			m.searchInput.Focus()
			return m, textinput.Blink

		case "tab":
			if m.focusedPane == listPane {
				m.focusedPane = viewerPane
			} else {
				m.focusedPane = listPane
			}
			return m, nil

		case "y":
			content := m.currentSchema
			if content != "" {
				if err := clipboard.WriteAll(content); err != nil {
					m.err = fmt.Errorf("failed to copy: %w", err)
				} else {
					m.copyNotify = "Copied to clipboard!"
				}
			}
			return m, nil

		case "e":
			if m.state == stateViewing && m.currentSchema != "" {
				return m.enterEditMode()
			}
			return m, nil

		case "E":
			if m.state == stateViewing && m.currentSchema != "" {
				m.state = stateEditing
				m.statusMsg = "Opening external editor..."
				return m, m.openExternalEditor()
			}
			return m, nil
		}

		if m.focusedPane == listPane {
			return m.handleListNavigation(msg)
		} else {
			return m.handleViewerNavigation(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) enterEditMode() (tea.Model, tea.Cmd) {
	// Generate template from schema
	template, err := avro.GenerateTemplate(m.rawSchema)
	if err != nil {
		m.err = fmt.Errorf("generating template: %w", err)
		return m, nil
	}

	m.editor.SetValue(template)
	m.editor.Focus()
	m.state = stateEditing
	m.statusMsg = "[EDIT] " + m.selectedSubject
	return m, textarea.Blink
}

func (m Model) handleEditMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel edit, return to view mode
		m.editor.SetValue(m.currentSchema)
		m.editor.Blur()
		m.state = stateViewing
		m.statusMsg = fmt.Sprintf("[VIEW] %s", m.selectedSubject)
		return m, nil

	case "ctrl+s", "ctrl+enter":
		// Validate and send
		m.state = stateSending
		m.statusMsg = "[SENDING...] " + m.selectedSubject
		return m, m.sendMessage()

	case "y":
		// In edit mode, copy the message content
		if err := clipboard.WriteAll(m.editor.Value()); err != nil {
			m.err = fmt.Errorf("failed to copy: %w", err)
		} else {
			m.copyNotify = "Message copied to clipboard!"
		}
		return m, nil

	default:
		// Pass to textarea
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}
}

func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateBrowsing
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.filteredSubjects = m.subjects
		m.selectedIndex = 0
		return m, nil
	case "enter":
		m.state = stateBrowsing
		m.searchInput.Blur()
		return m, nil
	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.filterSubjects()
		return m, cmd
	}
}

func (m *Model) filterSubjects() {
	query := strings.ToLower(m.searchInput.Value())
	if query == "" {
		m.filteredSubjects = m.subjects
	} else {
		filtered := []string{}
		for _, s := range m.subjects {
			if strings.Contains(strings.ToLower(s), query) {
				filtered = append(filtered, s)
			}
		}
		m.filteredSubjects = filtered
	}
	m.selectedIndex = 0
}

func (m Model) handleListNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
	case "down", "j":
		if m.selectedIndex < len(m.filteredSubjects)-1 {
			m.selectedIndex++
		}
	case "enter":
		if len(m.filteredSubjects) > 0 {
			m.selectedSubject = m.filteredSubjects[m.selectedIndex]
			m.statusMsg = fmt.Sprintf("Loading schema for %s...", m.selectedSubject)
			return m, m.loadSchema(m.selectedSubject)
		}
	case "pgup", "ctrl+u":
		m.selectedIndex -= 10
		if m.selectedIndex < 0 {
			m.selectedIndex = 0
		}
	case "pgdown", "ctrl+d":
		m.selectedIndex += 10
		if m.selectedIndex >= len(m.filteredSubjects) {
			m.selectedIndex = len(m.filteredSubjects) - 1
		}
		if m.selectedIndex < 0 {
			m.selectedIndex = 0
		}
	}
	return m, nil
}

func (m Model) handleViewerNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// In view mode, only allow scrolling, not editing
	switch msg.String() {
	case "up", "k":
		m.editor.CursorUp()
	case "down", "j":
		m.editor.CursorDown()
	case "pgup", "ctrl+u":
		for i := 0; i < 10; i++ {
			m.editor.CursorUp()
		}
	case "pgdown", "ctrl+d":
		for i := 0; i < 10; i++ {
			m.editor.CursorDown()
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth - 4

	left := m.renderList(leftWidth, m.height-4)
	right := m.renderViewer(rightWidth, m.height-4)

	var leftStyle, rightStyle lipgloss.Style
	if m.focusedPane == listPane {
		leftStyle = FocusedPaneStyle.Width(leftWidth)
		rightStyle = PaneStyle.Width(rightWidth)
	} else {
		leftStyle = PaneStyle.Width(leftWidth)
		if m.state == stateEditing {
			rightStyle = EditPaneStyle.Width(rightWidth)
		} else {
			rightStyle = FocusedPaneStyle.Width(rightWidth)
		}
	}

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(left),
		rightStyle.Render(right),
	)

	status := m.renderStatusBar()
	helpView := m.help.View(Keys)

	return lipgloss.JoinVertical(lipgloss.Left, main, status, HelpStyle.Render(helpView))
}

func (m Model) renderList(width, height int) string {
	var b strings.Builder

	title := ListTitleStyle.Render("Subjects")
	b.WriteString(title)
	b.WriteString("\n\n")

	if m.state == stateSearching {
		prompt := SearchPromptStyle.Render("/")
		b.WriteString(prompt)
		b.WriteString(m.searchInput.View())
		b.WriteString("\n\n")
	} else if m.searchInput.Value() != "" {
		b.WriteString(fmt.Sprintf("Filter: %s\n\n", m.searchInput.Value()))
	}

	if m.err != nil && m.state == stateBrowsing && len(m.subjects) == 0 {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		return b.String()
	}

	visibleHeight := height - 6
	if m.state == stateSearching || m.searchInput.Value() != "" {
		visibleHeight -= 2
	}

	start := 0
	if m.selectedIndex >= visibleHeight {
		start = m.selectedIndex - visibleHeight + 1
	}

	end := start + visibleHeight
	if end > len(m.filteredSubjects) {
		end = len(m.filteredSubjects)
	}

	for i := start; i < end; i++ {
		subject := m.filteredSubjects[i]
		if len(subject) > width-4 {
			subject = subject[:width-7] + "..."
		}

		if i == m.selectedIndex {
			b.WriteString(SelectedItemStyle.Render("> " + subject))
		} else {
			b.WriteString(NormalItemStyle.Render("  " + subject))
		}
		b.WriteString("\n")
	}

	if len(m.filteredSubjects) == 0 {
		b.WriteString(HelpStyle.Render("No subjects found"))
	}

	return b.String()
}

func (m Model) renderViewer(width, height int) string {
	var b strings.Builder

	var title string
	switch m.state {
	case stateEditing:
		title = EditTitleStyle.Render("Editor [EDIT]")
	case stateSending:
		title = ListTitleStyle.Render("Editor [SENDING...]")
	default:
		title = ListTitleStyle.Render("Schema [VIEW]")
	}
	b.WriteString(title)
	b.WriteString("\n\n")

	if m.currentSchema == "" {
		b.WriteString(HelpStyle.Render("Select a subject to view its schema"))
		return b.String()
	}

	m.editor.SetWidth(width - 2)
	m.editor.SetHeight(height - 4)
	b.WriteString(m.editor.View())

	return b.String()
}

func (m Model) renderStatusBar() string {
	var status string

	if m.copyNotify != "" {
		status = m.copyNotify
	} else if m.err != nil {
		status = ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	} else if m.statusMsg != "" {
		status = m.statusMsg
	} else {
		status = "Ready"
	}

	// Add Kafka status indicator
	if m.producer == nil {
		status += "  " + HelpStyle.Render("[Kafka: not configured]")
	}

	bar := StatusBarStyle.Width(m.width).Render(status)
	return bar
}
