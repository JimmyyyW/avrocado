package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
)

type Model struct {
	client *registry.Client

	subjects         []string
	filteredSubjects []string
	selectedIndex    int
	selectedSubject  string
	currentSchema    string

	searchInput textinput.Model
	viewer      viewport.Model
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

func NewModel(client *registry.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "Search subjects..."
	ti.CharLimit = 100

	vp := viewport.New(40, 20)

	h := help.New()
	h.ShowAll = false

	return Model{
		client:           client,
		subjects:         []string{},
		filteredSubjects: []string{},
		searchInput:      ti,
		viewer:           vp,
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewer.Width = m.width/2 - 4
		m.viewer.Height = m.height - 8
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
		m.currentSchema = registry.PrettyPrintSchema(msg.schema.Schema)
		m.viewer.SetContent(m.currentSchema)
		m.viewer.GotoTop()
		m.state = stateViewing
		m.focusedPane = viewerPane
		m.statusMsg = fmt.Sprintf("Schema: %s (v%d)", msg.schema.Subject, msg.schema.Version)
		return m, nil

	case tea.KeyMsg:
		m.copyNotify = ""
		m.err = nil

		if m.state == stateSearching {
			return m.handleSearchInput(msg)
		}

		switch {
		case msg.String() == "q" || msg.String() == "ctrl+c":
			return m, tea.Quit

		case msg.String() == "/":
			m.state = stateSearching
			m.searchInput.Focus()
			return m, textinput.Blink

		case msg.String() == "tab":
			if m.focusedPane == listPane {
				m.focusedPane = viewerPane
			} else {
				m.focusedPane = listPane
			}
			return m, nil

		case msg.String() == "y":
			if m.currentSchema != "" {
				if err := clipboard.WriteAll(m.currentSchema); err != nil {
					m.err = fmt.Errorf("failed to copy: %w", err)
				} else {
					m.copyNotify = "Schema copied to clipboard!"
				}
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
	var cmd tea.Cmd
	m.viewer, cmd = m.viewer.Update(msg)
	return m, cmd
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
		rightStyle = FocusedPaneStyle.Width(rightWidth)
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

	title := ListTitleStyle.Render("Schema")
	b.WriteString(title)
	b.WriteString("\n\n")

	if m.currentSchema == "" {
		b.WriteString(HelpStyle.Render("Select a subject to view its schema"))
		return b.String()
	}

	m.viewer.Width = width - 2
	m.viewer.Height = height - 4
	b.WriteString(m.viewer.View())

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

	bar := StatusBarStyle.Width(m.width).Render(status)
	return bar
}
