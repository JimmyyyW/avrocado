package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JimmyyyW/avrocado/internal/config"
)

type formField struct {
	label       string
	value       string
	placeholder string
	masked      bool
	hidden      bool
}

type ConfigEditorModel struct {
	configFile  *config.ConfigFile
	profileName string
	fields      []formField
	focusedIdx  int
	width       int
	height      int
	err         string
	saved       bool
	quit        bool
	isNewConfig bool
}

// NewConfigEditor creates a new config editor for a new profile
func NewConfigEditor(configFile *config.ConfigFile) ConfigEditorModel {
	return ConfigEditorModel{
		configFile:  configFile,
		isNewConfig: true,
		fields: []formField{
			{label: "Profile Name", value: "", placeholder: "e.g., local, production"},
			{label: "Schema Registry URL", value: "", placeholder: "http://localhost:8081"},
			{label: "Schema Registry Auth", value: "none", placeholder: "none|basic|sasl"},
			{label: "Schema Registry API Key", value: "", placeholder: "(for basic auth)", hidden: true},
			{label: "Schema Registry API Secret", value: "", placeholder: "(for basic auth)", masked: true, hidden: true},
			{label: "Schema Registry SASL Username", value: "", placeholder: "(for sasl auth)", hidden: true},
			{label: "Schema Registry SASL Password", value: "", placeholder: "(for sasl auth)", masked: true, hidden: true},
			{label: "Kafka Bootstrap Servers", value: "", placeholder: "localhost:9092"},
			{label: "Kafka Security Protocol", value: "PLAINTEXT", placeholder: "PLAINTEXT|SASL_SSL"},
			{label: "Kafka SASL Username", value: "", placeholder: "(for SASL_SSL)", hidden: true},
			{label: "Kafka SASL Password", value: "", placeholder: "(for SASL_SSL)", masked: true, hidden: true},
		},
	}
}

// NewConfigEditorForProfile creates a new config editor for editing an existing profile
func NewConfigEditorForProfile(configFile *config.ConfigFile, profileName string) ConfigEditorModel {
	m := NewConfigEditor(configFile)
	m.profileName = profileName
	m.isNewConfig = false

	if profile, err := configFile.GetProfile(profileName); err == nil {
		m.fields[0].value = profile.Name
		m.fields[1].value = profile.SchemaRegistry.URL

		// Set auth method
		authMethod := profile.SchemaRegistry.AuthMethod
		if authMethod == "" {
			// Infer from old config format
			if profile.SchemaRegistry.APIKey != "" {
				authMethod = "basic"
			} else {
				authMethod = "none"
			}
		}
		m.fields[2].value = authMethod

		// Load schema registry credentials
		m.fields[3].value = profile.SchemaRegistry.APIKey
		m.fields[4].value = profile.SchemaRegistry.APISecret
		m.fields[5].value = profile.SchemaRegistry.SASLUsername
		m.fields[6].value = profile.SchemaRegistry.SASLPassword

		// Load kafka settings
		m.fields[7].value = profile.Kafka.BootstrapServers
		m.fields[8].value = profile.Kafka.SecurityProtocol
		m.fields[9].value = profile.Kafka.SASLUsername
		m.fields[10].value = profile.Kafka.SASLPassword

		// Update field visibility based on auth methods
		if authMethod == "basic" {
			m.fields[3].hidden = false
			m.fields[4].hidden = false
		} else if authMethod == "sasl" {
			m.fields[5].hidden = false
			m.fields[6].hidden = false
		}

		// Show Kafka SASL fields if SASL_SSL is selected
		if profile.Kafka.SecurityProtocol == "SASL_SSL" {
			m.fields[9].hidden = false
			m.fields[10].hidden = false
		}
	}

	return m
}

func (m ConfigEditorModel) Init() tea.Cmd {
	return nil
}

func (m ConfigEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel editing
			m.quit = true
			return m, nil
		case "tab":
			// Move to next visible field
			for i := 0; i < len(m.fields); i++ {
				m.focusedIdx = (m.focusedIdx + 1) % len(m.fields)
				if !m.fields[m.focusedIdx].hidden {
					break
				}
			}
		case "shift+tab":
			// Move to previous visible field
			for i := 0; i < len(m.fields); i++ {
				m.focusedIdx = (m.focusedIdx - 1 + len(m.fields)) % len(m.fields)
				if !m.fields[m.focusedIdx].hidden {
					break
				}
			}
		case "enter":
			if m.focusedIdx == len(m.fields)-1 {
				// Save configuration
				if err := m.saveProfile(); err != nil {
					m.err = err.Error()
				} else {
					m.saved = true
					m.quit = true
					return m, nil
				}
			} else {
				// Move to next field
				for i := 0; i < len(m.fields); i++ {
					m.focusedIdx = (m.focusedIdx + 1) % len(m.fields)
					if !m.fields[m.focusedIdx].hidden {
						break
					}
				}
			}
		default:
			// Handle text input
			if len(msg.String()) == 1 {
				m.fields[m.focusedIdx].value += msg.String()
			} else if msg.String() == "backspace" {
				if len(m.fields[m.focusedIdx].value) > 0 {
					m.fields[m.focusedIdx].value = m.fields[m.focusedIdx].value[:len(m.fields[m.focusedIdx].value)-1]
				}
			} else if msg.String() == "ctrl+u" {
				m.fields[m.focusedIdx].value = ""
			}

			// Update hidden fields based on schema registry auth method
			if m.focusedIdx == 2 { // Schema Registry Auth field
				if m.fields[2].value == "basic" {
					m.fields[3].hidden = false
					m.fields[4].hidden = false
					m.fields[5].hidden = true
					m.fields[6].hidden = true
				} else if m.fields[2].value == "sasl" {
					m.fields[3].hidden = true
					m.fields[4].hidden = true
					m.fields[5].hidden = false
					m.fields[6].hidden = false
				} else { // none
					m.fields[3].hidden = true
					m.fields[4].hidden = true
					m.fields[5].hidden = true
					m.fields[6].hidden = true
				}
			}

			// Update hidden fields based on kafka security protocol
			if m.focusedIdx == 8 { // Kafka Security Protocol field
				if m.fields[8].value == "SASL_SSL" {
					m.fields[9].hidden = false
					m.fields[10].hidden = false
				} else if m.fields[8].value == "PLAINTEXT" {
					m.fields[9].hidden = true
					m.fields[10].hidden = true
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m ConfigEditorModel) View() string {
	var s string
	title := "New Configuration"
	if !m.isNewConfig {
		title = fmt.Sprintf("Edit Configuration: %s", m.profileName)
	}
	s += lipgloss.NewStyle().Bold(true).Render(title) + "\n\n"

	visibleFieldCount := 0
	for _, field := range m.fields {
		if !field.hidden {
			visibleFieldCount++
		}
	}

	for i, field := range m.fields {
		if field.hidden {
			continue
		}

		prefix := "  "
		if i == m.focusedIdx {
			prefix = "> "
		}

		label := lipgloss.NewStyle().Width(25).Render(field.label + ":")
		value := field.value
		if field.masked && len(value) > 0 {
			masked := ""
			for range value {
				masked += "*"
			}
			value = masked
		}
		if value == "" {
			value = lipgloss.NewStyle().Faint(true).Render(field.placeholder)
		}

		if i == m.focusedIdx {
			s += lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")).
				Bold(true).
				Render(prefix + label + " " + value) + "\n"
		} else {
			s += prefix + label + " " + value + "\n"
		}
	}

	s += "\n"

	// Determine what button text to show
	buttonText := "[tab] Next  [shift+tab] Prev  [enter] Save  [esc] Cancel"
	if m.err != "" {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗ Error: "+m.err) + "\n\n"
	}

	s += lipgloss.NewStyle().Faint(true).Render(buttonText) + "\n"

	return s
}

func (m *ConfigEditorModel) saveProfile() error {
	profileName := m.fields[0].value
	if profileName == "" {
		return fmt.Errorf("profile name is required")
	}

	srURL := m.fields[1].value
	if srURL == "" {
		return fmt.Errorf("schema registry URL is required")
	}

	kafkaServers := m.fields[7].value
	if kafkaServers == "" {
		return fmt.Errorf("kafka bootstrap servers is required")
	}

	// Build schema registry config
	srAuthMethod := m.fields[2].value
	srConfig := config.SchemaRegistryConfig{
		URL:        srURL,
		AuthMethod: srAuthMethod,
	}

	// Load auth credentials based on method
	if srAuthMethod == "basic" {
		srConfig.APIKey = m.fields[3].value
		srConfig.APISecret = m.fields[4].value
	} else if srAuthMethod == "sasl" {
		srConfig.SASLUsername = m.fields[5].value
		srConfig.SASLPassword = m.fields[6].value
		srConfig.SecurityProtocol = "SASL_SSL"
	}

	// Create profile config
	profile := &config.ProfileConfig{
		Name:           profileName,
		SchemaRegistry: srConfig,
		Kafka: config.KafkaConfig{
			BootstrapServers: kafkaServers,
			SecurityProtocol: m.fields[8].value,
			SASLUsername:     m.fields[9].value,
			SASLPassword:     m.fields[10].value,
		},
	}

	// Update config file (in memory)
	if m.configFile.Configurations == nil {
		m.configFile.Configurations = make(map[string]*config.ProfileConfig)
	}

	// Use the original name for edit, new name for create
	keyName := profileName
	if !m.isNewConfig {
		keyName = m.profileName
	}

	m.configFile.Configurations[keyName] = profile

	return nil
}

// SavedProfile returns the saved profile if configuration was saved
func (m ConfigEditorModel) SavedProfile() *config.ProfileConfig {
	if m.saved && len(m.fields) > 0 {
		return m.configFile.Configurations[m.fields[0].value]
	}
	return nil
}
