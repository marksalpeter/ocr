package command

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marksalpeter/ocr/internal/ocr"
	"github.com/marksalpeter/ocr/internal/ocr/client"
	"github.com/marksalpeter/ocr/internal/ocr/repository"
)

// processingDoneMsg is sent when processing completes successfully
type processingDoneMsg struct {
	totalCost    float64
	costPerImage float64
}

// processingErrorMsg is sent when processing fails
type processingErrorMsg struct {
	err error
}

// statusModel represents the execution status display
type statusModel struct {
	ctx         context.Context
	config      *Config
	status      string
	message     string
	err         error
	totalCost   float64
	costPerImage float64
	completed   bool
}

// newStatusModel creates a new status model
func newStatusModel(ctx context.Context, config *Config) *statusModel {
	return &statusModel{
		ctx:     ctx,
		config:  config,
		status:  "processing",
		message: "Processing images...",
	}
}

func (m *statusModel) Init() tea.Cmd {
	return m.processImages
}

func (m *statusModel) processImages() tea.Msg {
	// Create repository with the input directory and output file from config
	repo := repository.New(m.config.InputDir, m.config.OutputFile)

	// Create the OCR client with the API key from config
	ocrClient := client.New(m.config.APIKey)

	// Create application instance
	app := ocr.NewApp(ocrClient, repo)

	// Convert command config to app config (only fields the app needs)
	appConfig := &ocr.AppConfig{
		OutputFile:  m.config.OutputFile,
		Concurrency: m.config.Concurrency,
		StartDate:   m.config.StartDate,
	}

	// Process images
	if err := app.ProcessImages(m.ctx, appConfig); err != nil {
		return processingErrorMsg{err: err}
	}

	// Get cost information
	totalCost, costPerImage := app.GetCost()
	return processingDoneMsg{
		totalCost:    totalCost,
		costPerImage: costPerImage,
	}
}

func (m *statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.completed {
			switch msg.String() {
			case "q", "ctrl+c", "enter":
				return m, tea.Quit
			}
		}
	case processingDoneMsg:
		m.status = "success"
		m.message = "Processing completed successfully!"
		m.totalCost = msg.totalCost
		m.costPerImage = msg.costPerImage
		m.completed = true
		return m, nil
	case processingErrorMsg:
		m.status = "error"
		m.err = msg.err
		m.completed = true
		return m, nil
	}
	return m, nil
}

func (m *statusModel) View() string {
	var content string
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	content = titleStyle.Render("OCR Processing\n\n")

	switch m.status {
	case "processing":
		content += fmt.Sprintf("Status: %s\n", m.message)
		content += infoStyle.Render("Please wait...\n")

	case "success":
		content += successStyle.Render("✓ " + m.message + "\n\n")
		content += fmt.Sprintf("Output file: %s\n", m.config.OutputFile)
		content += fmt.Sprintf("Total cost: $%.4f\n", m.totalCost)
		content += fmt.Sprintf("Cost per image: $%.4f\n", m.costPerImage)
		content += "\n" + infoStyle.Render("Press Enter or 'q' to exit")

	case "error":
		if m.err != nil {
			content += errorStyle.Render("✗ Error: " + m.err.Error() + "\n")
		} else {
			content += errorStyle.Render("✗ Error occurred\n")
		}
		content += "\n" + infoStyle.Render("Press Enter or 'q' to exit")
	}

	return content
}

// runStatusModel runs the status model and returns the result
func runStatusModel(ctx context.Context, config *Config) error {
	model := newStatusModel(ctx, config)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("error running status model: %w", err)
	}

	statusModel, ok := finalModel.(*statusModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	if statusModel.status == "error" {
		return statusModel.err
	}

	return nil
}

