package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	"github.com/JimmyyyW/avrocado/internal/config"
)

type selectorState int

const (
	stateSelecting selectorState = iota
	stateEditing
	stateConfirmDelete
)

type ConfigSelectorModel struct {
	configFile    *config.ConfigFile
	configPath    string
	profiles      []string
	selectedIdx   int
	width         int
	height        int
	selectedName  string
	state         selectorState
	editor        ConfigEditorModel
	err           string
	message       string
	messageTimer  int
}

// NewConfigSelector creates a new config selector model
func NewConfigSelector(configFile *config.ConfigFile) ConfigSelectorModel {
	profiles := make([]string, 0, len(configFile.Configurations))
	for name := range configFile.Configurations {
		profiles = append(profiles, name)
	}

	// Sort profiles with default first, then alphabetically
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i] == configFile.Default {
			return true
		}
		if profiles[j] == configFile.Default {
			return false
		}
		return profiles[i] < profiles[j]
	})

	return ConfigSelectorModel{
		configFile: configFile,
		configPath: config.GetConfigPath(),
		profiles:   profiles,
		selectedIdx: 0,
		state:      stateSelecting,
	}
}

func (m ConfigSelectorModel) Init() tea.Cmd {
	return nil
}

func (m ConfigSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle editor state
	if m.state == stateEditing {
		return m.handleEditorState(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			// Quit without selecting
			return m, tea.Quit
		case "esc":
			if m.message != "" {
				m.message = ""
			}
		case "enter":
			// Select the current profile
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.profiles) {
				m.selectedName = m.profiles[m.selectedIdx]
				return m, tea.Quit
			}
		case "j", "down":
			if m.selectedIdx < len(m.profiles)-1 {
				m.selectedIdx++
			}
		case "k", "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "n":
			// Create new configuration
			m.state = stateEditing
			m.editor = NewConfigEditor(m.configFile)
		case "e":
			// Edit selected configuration
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.profiles) {
				profileName := m.profiles[m.selectedIdx]
				m.state = stateEditing
				m.editor = NewConfigEditorForProfile(m.configFile, profileName)
			}
		case "d":
			// Set as default
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.profiles) {
				m.configFile.Default = m.profiles[m.selectedIdx]
				if err := m.saveConfigFile(); err != nil {
					m.err = err.Error()
				} else {
					m.message = fmt.Sprintf("Set '%s' as default", m.profiles[m.selectedIdx])
					m.messageTimer = 3 // Show for 3 seconds
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *ConfigSelectorModel) handleEditorState(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	newModel, cmd := m.editor.Update(msg)
	m.editor = newModel.(ConfigEditorModel)

	// Check if editor quit
	if m.editor.quit {
		if m.editor.saved {
			// Refresh profile list
			m.profiles = make([]string, 0, len(m.configFile.Configurations))
			for name := range m.configFile.Configurations {
				m.profiles = append(m.profiles, name)
			}
			sort.Slice(m.profiles, func(i, j int) bool {
				if m.profiles[i] == m.configFile.Default {
					return true
				}
				if m.profiles[j] == m.configFile.Default {
					return false
				}
				return m.profiles[i] < m.profiles[j]
			})

			// Save config file
			if err := m.saveConfigFile(); err != nil {
				m.err = err.Error()
			} else {
				m.message = "Configuration saved"
				m.messageTimer = 2
			}
		}

		m.state = stateSelecting
		m.editor = ConfigEditorModel{}
	}

	return m, cmd
}

func (m ConfigSelectorModel) View() string {
	if m.state == stateEditing {
		return m.editor.View()
	}

	if len(m.profiles) == 0 {
		return "No configurations found. Create one with 'n'.\n"
	}

	var s string
	s += lipgloss.NewStyle().Bold(true).Render("Select Configuration") + "\n\n"

	for i, name := range m.profiles {
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}

		profileName := name
		if name == m.configFile.Default {
			profileName += " (Default)"
		}

		if i == m.selectedIdx {
			s += lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")).
				Bold(true).
				Render(prefix + profileName) + "\n"
		} else {
			s += prefix + profileName + "\n"
		}
	}

	s += "\n"
	if m.message != "" {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓ "+m.message) + "\n\n"
	}
	if m.err != "" {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗ "+m.err) + "\n\n"
	}

	s += lipgloss.NewStyle().Faint(true).Render("[enter] Select  [n] New  [e] Edit  [d] Default  [q] Quit") + "\n"

	return s
}

// SelectedProfile returns the selected profile config
func (m ConfigSelectorModel) SelectedProfile() *config.ProfileConfig {
	if m.selectedName != "" {
		profile, _ := m.configFile.GetProfile(m.selectedName)
		return profile
	}
	return nil
}

func (m *ConfigSelectorModel) saveConfigFile() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(m.configFile)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
