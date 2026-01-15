package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JimmyyyW/avrocado/internal/config"
)

type ConfigSelectorModel struct {
	configFile    *config.ConfigFile
	profiles      []string
	selectedIdx   int
	width         int
	height        int
	selectedName  string
	err           error
	editingNew    bool
	editingIdx    int
}

// NewConfigSelector creates a new config selector model
func NewConfigSelector(configFile *config.ConfigFile) ConfigSelectorModel {
	profiles := make([]string, 0, len(configFile.Configurations))
	for name := range configFile.Configurations {
		profiles = append(profiles, name)
	}

	// Sort profiles with default first
	sortProfilesWithDefault(profiles, configFile.Default)

	return ConfigSelectorModel{
		configFile:  configFile,
		profiles:    profiles,
		selectedIdx: 0,
	}
}

func (m ConfigSelectorModel) Init() tea.Cmd {
	return nil
}

func (m ConfigSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			// Quit without selecting
			return m, tea.Quit
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
			// TODO: Create new configuration
		case "e":
			// TODO: Edit selected configuration
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m ConfigSelectorModel) View() string {
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
	s += lipgloss.NewStyle().Faint(true).Render("[enter] Select  [n] New  [e] Edit  [q] Quit") + "\n"

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

func sortProfilesWithDefault(profiles []string, defaultName string) {
	// Simple bubble sort to move default to front
	for i, name := range profiles {
		if name == defaultName {
			// Swap with first element
			profiles[0], profiles[i] = profiles[i], profiles[0]
			break
		}
	}
}
