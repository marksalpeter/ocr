package command

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// ErrConfigCancelled is returned when the user cancels configuration
	ErrConfigCancelled = fmt.Errorf("configuration cancelled")
	// ErrInvalidInput is returned when user input is invalid
	ErrInvalidInput = fmt.Errorf("invalid input")
)

// Config contains all configuration parameters
type Config struct {
	InputDir    string
	OutputFile  string
	APIKey      string
	Concurrency int
	StartDate   string
}

// configCollector collects configuration using bubbletea
type configCollector struct{}

// newConfigCollector creates a new configCollector instance
func newConfigCollector() *configCollector {
	return &configCollector{}
}

// Collect gathers configuration parameters from the user
func (c *configCollector) Collect() (*Config, error) {
	model := newConfigModel()
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return nil, err
	}

	configModel, ok := finalModel.(*configModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	if !configModel.completed {
		return nil, ErrConfigCancelled
	}

	return configModel.config, nil
}

type configModel struct {
	step      int
	config    *Config
	completed bool
	cancelled bool

	// Input fields
	inputDir    string
	outputFile  string
	apiKey      string
	concurrency string
	startDate   string

	// Validation errors
	err string
}

func newConfigModel() *configModel {
	wd, _ := os.Getwd()
	return &configModel{
		step:        0,
		config:      &Config{},
		inputDir:    wd,
		outputFile:  "output.txt",
		concurrency: "10",
	}
}

func (m *configModel) Init() tea.Cmd {
	return nil
}

func (m *configModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			return m.handleEnter()
		case "backspace":
			return m.handleBackspace()
		default:
			return m.handleInput(msg.String())
		}
	}
	return m, nil
}

func (m *configModel) handleEnter() (tea.Model, tea.Cmd) {
	m.err = ""

	switch m.step {
	case 0: // Input directory
		if m.inputDir == "" {
			m.err = "Input directory cannot be empty"
			return m, nil
		}
		m.step++
	case 1: // Output file
		if m.outputFile == "" {
			m.err = "Output file cannot be empty"
			return m, nil
		}
		m.step++
	case 2: // API key
		if m.apiKey == "" {
			m.err = "API key cannot be empty"
			return m, nil
		}
		m.step++
	case 3: // Concurrency
		concurrency, err := strconv.Atoi(m.concurrency)
		if err != nil || concurrency <= 0 {
			m.err = "Concurrency must be a positive integer"
			return m, nil
		}
		m.step++
	case 4: // Start date (optional)
		// Build final config
		concurrency, _ := strconv.Atoi(m.concurrency)
		m.config = &Config{
			InputDir:    m.inputDir,
			OutputFile:  m.outputFile,
			APIKey:      m.apiKey,
			Concurrency: concurrency,
			StartDate:   m.startDate,
		}
		m.completed = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *configModel) handleBackspace() (tea.Model, tea.Cmd) {
	switch m.step {
	case 0:
		if len(m.inputDir) > 0 {
			m.inputDir = m.inputDir[:len(m.inputDir)-1]
		}
	case 1:
		if len(m.outputFile) > 0 {
			m.outputFile = m.outputFile[:len(m.outputFile)-1]
		}
	case 2:
		if len(m.apiKey) > 0 {
			m.apiKey = m.apiKey[:len(m.apiKey)-1]
		}
	case 3:
		if len(m.concurrency) > 0 {
			m.concurrency = m.concurrency[:len(m.concurrency)-1]
		}
	case 4:
		if len(m.startDate) > 0 {
			m.startDate = m.startDate[:len(m.startDate)-1]
		}
	}
	return m, nil
}

func (m *configModel) handleInput(key string) (tea.Model, tea.Cmd) {
	// Filter out control characters
	if len(key) != 1 || key[0] < 32 {
		return m, nil
	}

	switch m.step {
	case 0:
		m.inputDir += key
	case 1:
		m.outputFile += key
	case 2:
		m.apiKey += key
	case 3:
		if key >= "0" && key <= "9" {
			m.concurrency += key
		}
	case 4:
		m.startDate += key
	}
	return m, nil
}

func (m *configModel) View() string {
	if m.cancelled {
		return "Configuration cancelled.\n"
	}

	if m.completed {
		return "Configuration completed!\n"
	}

	var content string
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	steps := []string{
		fmt.Sprintf("Input directory (default: %s): %s", m.inputDir, m.inputDir),
		fmt.Sprintf("Output file (default: %s): %s", m.outputFile, m.outputFile),
		"OpenAI API key: " + maskString(m.apiKey),
		fmt.Sprintf("Concurrency level (default: %s): %s", m.concurrency, m.concurrency),
		"Start date (optional, press Enter to skip): " + m.startDate,
	}

	content = titleStyle.Render("OCR Configuration\n\n")

	for i, step := range steps {
		if i == m.step {
			content += fmt.Sprintf("> %s\n", step)
		} else if i < m.step {
			content += fmt.Sprintf("✓ %s\n", step)
		} else {
			content += fmt.Sprintf("  %s\n", step)
		}
	}

	if m.err != "" {
		content += "\n" + errorStyle.Render("Error: "+m.err) + "\n"
	}

	content += "\nPress Enter to continue, Ctrl+C or 'q' to cancel"

	return content
}

func maskString(s string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}
