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
	"github.com/charmbracelet/bubbles/viewport"
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
	stateSendMode
	stateSending
	stateSavingEvent
	stateLoadingEvent
	stateConsumerMode
)

type Model struct {
	client   *registry.Client
	producer *kafka.Producer
	cfg      *config.Config

	subjects         []string
	filteredSubjects []string
	selectedIndex    int
	selectedSubject  string
	currentSchema    string
	rawSchema        string // Original schema JSON for validation
	schemaID         int

	searchInput textinput.Model
	keyInput    textinput.Model  // Message key input
	viewer      viewport.Model   // Read-only schema view
	editor      textarea.Model   // Editable send mode
	help        help.Model

	focusedPane pane
	state       state
	sendKeyFocused bool // Track if key field has focus in send mode

	width  int
	height int

	err        error
	statusMsg  string
	copyNotify string

	// Event persistence
	lastPayload string
	eventSaver  EventSaverModel
	eventLoader EventLoaderModel

	// Consumer mode
	consumer         *kafka.Consumer
	consumedMessages []kafka.Message
	currentMsgIdx    int
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

func NewModel(client *registry.Client, producer *kafka.Producer, cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Search subjects..."
	ti.CharLimit = 100

	ki := textinput.New()
	ki.Placeholder = "Message key (optional)"
	ki.CharLimit = 256

	vp := viewport.New(40, 20)

	ta := textarea.New()
	ta.Placeholder = "Edit message payload..."
	ta.ShowLineNumbers = true
	ta.SetWidth(40)
	ta.SetHeight(20)

	h := help.New()
	h.ShowAll = false

	return Model{
		client:           client,
		producer:         producer,
		cfg:              cfg,
		subjects:         []string{},
		filteredSubjects: []string{},
		searchInput:      ti,
		keyInput:         ki,
		viewer:           vp,
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

		// Produce message with optional key
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = m.producer.ProduceWithStringKey(ctx, topic, m.schemaID, m.keyInput.Value(), binary)
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
		m.viewer.Width = m.width/2 - 6
		m.viewer.Height = m.height - 10
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
		m.viewer.SetContent(m.currentSchema)
		m.viewer.GotoTop()
		m.state = stateViewing
		m.focusedPane = viewerPane
		m.statusMsg = fmt.Sprintf("[VIEW] %s (v%d)", msg.schema.Subject, msg.schema.Version)
		return m, nil

	case messageSentMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateSendMode
			m.statusMsg = "[SEND MODE] Failed - press Ctrl+S to retry"
		} else {
			m.state = stateViewing
			m.editor.Blur()
			m.statusMsg = fmt.Sprintf("SUCCESS: Message produced to topic '%s'", msg.topic)
			m.copyNotify = fmt.Sprintf("Message produced to '%s'!", msg.topic)
		}
		return m, nil

	case externalEditorMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateViewing
		} else {
			m.editor.SetValue(msg.content)
			topic := config.SubjectToTopic(m.selectedSubject)
			m.state = stateSendMode
			m.statusMsg = fmt.Sprintf("[SEND MODE] Target: %s  |  Ctrl+S to send, Esc to cancel", topic)
		}
		return m, nil

	case tea.KeyMsg:
		m.copyNotify = ""
		m.err = nil

		// Handle state-specific input
		switch m.state {
		case stateSearching:
			return m.handleSearchInput(msg)
		case stateSendMode:
			return m.handleSendMode(msg)
		case stateSending:
			// Ignore input while sending
			return m, nil
		case stateSavingEvent:
			return m.handleSavingEvent(msg)
		case stateLoadingEvent:
			return m.handleLoadingEvent(msg)
		case stateConsumerMode:
			return m.handleConsumerMode(msg)
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

		case "e", "s":
			if m.state == stateViewing && m.currentSchema != "" {
				return m.enterSendMode()
			}
			return m, nil

		case "E":
			if m.state == stateViewing && m.currentSchema != "" {
				m.state = stateSendMode
				m.statusMsg = "Opening external editor..."
				return m, m.openExternalEditor()
			}
			return m, nil

		case "c":
			if m.state == stateViewing && m.currentSchema != "" {
				return m.enterConsumerMode()
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

func (m Model) enterSendMode() (tea.Model, tea.Cmd) {
	// Generate template from schema
	template, err := avro.GenerateTemplate(m.rawSchema)
	if err != nil {
		m.err = fmt.Errorf("generating template: %w", err)
		return m, nil
	}

	topic := config.SubjectToTopic(m.selectedSubject)
	m.editor.SetValue(template)
	m.editor.Focus()
	m.keyInput.SetValue("") // Clear key field
	m.keyInput.Blur()
	m.sendKeyFocused = false // Focus starts on message
	m.state = stateSendMode
	m.statusMsg = fmt.Sprintf("[SEND MODE] Target: %s  |  Ctrl+S send, Ctrl+N save, Ctrl+O load, Tab key, Esc cancel", topic)
	return m, textarea.Blink
}

func (m Model) handleSendMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Check send keys first (before passing to textarea)
	switch key {
	case "esc":
		// Cancel, return to view mode
		m.editor.Blur()
		m.state = stateViewing
		m.statusMsg = fmt.Sprintf("[VIEW] %s", m.selectedSubject)
		return m, nil

	case "ctrl+s":
		// Save the last payload before sending
		m.lastPayload = m.editor.Value()
		// Validate and send
		m.state = stateSending
		m.statusMsg = "[SENDING...] " + m.selectedSubject
		return m, m.sendMessage()

	case "ctrl+n":
		// Save current message
		topic := config.SubjectToTopic(m.selectedSubject)
		m.eventSaver = NewEventSaver(topic, m.schemaID, m.editor.Value())
		m.state = stateSavingEvent
		m.statusMsg = "[SAVE EVENT]"
		return m, nil

	case "ctrl+o":
		// Load saved message (Ctrl+L was intercepted by terminal)
		topic := config.SubjectToTopic(m.selectedSubject)
		m.eventLoader = NewEventLoader(topic)
		m.state = stateLoadingEvent
		m.statusMsg = "[LOAD EVENT]"
		return m, nil

	case "y":
		// Copy the message content
		if err := clipboard.WriteAll(m.editor.Value()); err != nil {
			m.err = fmt.Errorf("failed to copy: %w", err)
		} else {
			m.copyNotify = "Message copied to clipboard!"
		}
		return m, nil

	case "tab":
		// Toggle between key and message fields
		if m.sendKeyFocused {
			// Switch from key to message
			m.keyInput.Blur()
			m.editor.Focus()
			m.sendKeyFocused = false
		} else {
			// Switch from message to key
			m.editor.Blur()
			m.keyInput.Focus()
			m.sendKeyFocused = true
		}
		return m, nil

	case "shift+tab":
		// Toggle backwards between key and message fields
		if m.sendKeyFocused {
			// Switch from key to message
			m.keyInput.Blur()
			m.editor.Focus()
			m.sendKeyFocused = false
		} else {
			// Switch from message to key
			m.editor.Blur()
			m.keyInput.Focus()
			m.sendKeyFocused = true
		}
		return m, nil
	}

	// Pass other keys to the focused field
	var cmd tea.Cmd
	if m.sendKeyFocused {
		m.keyInput, cmd = m.keyInput.Update(msg)
	} else {
		m.editor, cmd = m.editor.Update(msg)
	}
	return m, cmd
}

func (m *Model) handleSavingEvent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	newModel, cmd := m.eventSaver.Update(msg)
	m.eventSaver = newModel.(EventSaverModel)

	if m.eventSaver.quit {
		if m.eventSaver.Saved() {
			m.statusMsg = fmt.Sprintf("[SEND MODE] Saved: %s", m.eventSaver.FilePath())
		}
		m.state = stateSendMode
	}

	return m, cmd
}

func (m *Model) handleLoadingEvent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	newModel, cmd := m.eventLoader.Update(msg)
	m.eventLoader = newModel.(EventLoaderModel)

	if m.eventLoader.Quit() {
		event := m.eventLoader.LoadedEvent()
		if event != nil {
			m.editor.SetValue(event.Payload)
			m.statusMsg = fmt.Sprintf("[SEND MODE] Loaded: %s", event.Name)
		}
		m.state = stateSendMode
	}

	return m, cmd
}

func (m *Model) enterConsumerMode() (tea.Model, tea.Cmd) {
	topic := config.SubjectToTopic(m.selectedSubject)

	// Create consumer
	consumer, err := kafka.NewConsumer(m.cfg, topic)
	if err != nil {
		m.err = fmt.Errorf("[DEBUG] Failed to create consumer for topic %s: %w", topic, err)
		return m, nil
	}

	m.consumer = consumer
	m.consumedMessages = []kafka.Message{}
	m.currentMsgIdx = 0
	m.state = stateConsumerMode
	m.statusMsg = fmt.Sprintf("[CONSUMER MODE] Topic: %s  |  Ctrl+M consume, Esc cancel, j/k navigate", topic)
	m.err = fmt.Errorf("[DEBUG] Consumer created for topic: %s | Bootstrap: %s | Security: %s | Press Ctrl+M to fetch messages",
		topic, m.cfg.KafkaBootstrapServers, m.cfg.KafkaSecurityProtocol)
	return m, nil
}

func (m *Model) handleConsumerMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		// Exit consumer mode
		if m.consumer != nil {
			m.consumer.Close()
			m.consumer = nil
		}
		m.consumedMessages = []kafka.Message{}
		m.currentMsgIdx = 0
		m.state = stateViewing
		m.statusMsg = fmt.Sprintf("[VIEW] %s", m.selectedSubject)
		return m, nil

	case "ctrl+m":
		// Consume messages
		topic := config.SubjectToTopic(m.selectedSubject)
		m.statusMsg = fmt.Sprintf("[CONSUMER MODE] Fetching from topic: %s (timeout: 5s)", topic)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		messages, err := m.consumer.FetchMessages(ctx, 10)
		if err != nil {
			// Surface detailed error information
			m.err = fmt.Errorf("[DEBUG] Topic: %s | Consumer error: %w", topic, err)
			m.statusMsg = fmt.Sprintf("[CONSUMER MODE] ERROR - Check error area above. Topic: %s", topic)
			return m, nil
		}

		if len(messages) == 0 {
			m.statusMsg = fmt.Sprintf("[CONSUMER MODE] No messages available | Topic: %s | Bootstrap: %s | Protocol: %s",
				topic, m.cfg.KafkaBootstrapServers, m.cfg.KafkaSecurityProtocol)
			m.err = fmt.Errorf("[DEBUG] Fetch returned 0 messages for topic %s. This means either: (1) topic has no messages, (2) consumer started reading from end instead of beginning, or (3) connection issue", topic)
			return m, nil
		}

		// Success - show what we fetched
		m.consumedMessages = messages
		m.currentMsgIdx = 0
		debugMsg := fmt.Sprintf("[DEBUG] Successfully fetched %d messages from topic: %s", len(messages), topic)
		m.statusMsg = fmt.Sprintf("[CONSUMER MODE] Fetched %d messages. Showing 1/%d | %s", len(messages), len(messages), debugMsg)
		return m, nil

	case "j", "down":
		if m.currentMsgIdx < len(m.consumedMessages)-1 {
			m.currentMsgIdx++
			m.statusMsg = fmt.Sprintf("[CONSUMER MODE] Message %d/%d", m.currentMsgIdx+1, len(m.consumedMessages))
		}
		return m, nil

	case "k", "up":
		if m.currentMsgIdx > 0 {
			m.currentMsgIdx--
			m.statusMsg = fmt.Sprintf("[CONSUMER MODE] Message %d/%d", m.currentMsgIdx+1, len(m.consumedMessages))
		}
		return m, nil

	case "y":
		// Copy current message
		if len(m.consumedMessages) > 0 {
			msg := m.consumedMessages[m.currentMsgIdx]
			if err := clipboard.WriteAll(msg.Value); err != nil {
				m.err = fmt.Errorf("failed to copy: %w", err)
			} else {
				m.copyNotify = "Message copied to clipboard!"
			}
		}
		return m, nil
	}

	return m, nil
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
	// Pass all keys to viewport for scrolling
	var cmd tea.Cmd
	m.viewer, cmd = m.viewer.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Handle event saving/loading overlays
	if m.state == stateSavingEvent {
		return m.eventSaver.View()
	}
	if m.state == stateLoadingEvent {
		return m.eventLoader.View()
	}

	// Handle consumer mode
	if m.state == stateConsumerMode {
		return m.renderConsumerMode()
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
		if m.state == stateSendMode {
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

	switch m.state {
	case stateSendMode:
		topic := config.SubjectToTopic(m.selectedSubject)
		title := EditTitleStyle.Render("Send Mode")
		b.WriteString(title)
		b.WriteString("\n")
		topicLine := fmt.Sprintf("→ Topic: %s", topic)
		b.WriteString(SelectedItemStyle.Render(topicLine))
		b.WriteString("\n\n")
	case stateSending:
		topic := config.SubjectToTopic(m.selectedSubject)
		title := ListTitleStyle.Render("Sending...")
		b.WriteString(title)
		b.WriteString("\n")
		topicLine := fmt.Sprintf("→ Topic: %s", topic)
		b.WriteString(HelpStyle.Render(topicLine))
		b.WriteString("\n\n")
	default:
		title := ListTitleStyle.Render("Schema")
		b.WriteString(title)
		b.WriteString("\n\n")
	}

	if m.currentSchema == "" {
		b.WriteString(HelpStyle.Render("Select a subject to view its schema"))
		return b.String()
	}

	contentHeight := height - 6
	if m.state == stateSendMode || m.state == stateSending {
		contentHeight = height - 10 // Account for topic line + key field

		// Render key input field
		m.keyInput.Width = width - 2
		keyStyle := lipgloss.NewStyle()
		if m.sendKeyFocused && m.state == stateSendMode {
			keyStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true).
				BorderForeground(lipgloss.Color("11"))
		}
		b.WriteString(keyStyle.Render(m.keyInput.View()))
		b.WriteString("\n")

		// Render message editor
		m.editor.SetWidth(width - 2)
		m.editor.SetHeight(contentHeight)
		b.WriteString(m.editor.View())
	} else {
		m.viewer.Width = width - 2
		m.viewer.Height = contentHeight
		b.WriteString(m.viewer.View())
	}

	return b.String()
}

func (m Model) renderStatusBar() string {
	var status string

	if m.copyNotify != "" {
		status = SuccessStyle.Render(m.copyNotify)
	} else if m.err != nil {
		status = ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	} else if strings.HasPrefix(m.statusMsg, "SUCCESS:") {
		status = SuccessStyle.Render(m.statusMsg)
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

func (m Model) renderConsumerMode() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Render("Consumer Mode")
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(m.consumedMessages) == 0 {
		b.WriteString("No messages. Press Ctrl+M to consume.\n")
	} else {
		currentMsg := m.consumedMessages[m.currentMsgIdx]

		// Header with counter
		header := fmt.Sprintf("Message %d/%d (Offset: %d)", m.currentMsgIdx+1, len(m.consumedMessages), currentMsg.Offset)
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")).Render(header))
		b.WriteString("\n\n")

		// Key
		if currentMsg.Key != "" {
			b.WriteString("Key:\n")
			b.WriteString(currentMsg.Key)
			b.WriteString("\n\n")
		}

		// Value (payload)
		b.WriteString("Value:\n")
		b.WriteString(currentMsg.Value)
		b.WriteString("\n\n")

		// Timestamp
		b.WriteString(lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("Timestamp: %s", currentMsg.Timestamp)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("[Ctrl+M] Consume  [j/k] Navigate  [y] Copy  [Esc] Exit"))

	return b.String()
}
