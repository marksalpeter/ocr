package command

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
)

// Config contains all configuration parameters
type Config struct {
	InputDir    string
	OutputFile  string
	APIKey      string
	Concurrency int
	StartDate   string
}

var (
	// ErrConfigCancelled is returned when the user cancels configuration
	ErrConfigCancelled = fmt.Errorf("configuration cancelled")
	// ErrInvalidInput is returned when user input is invalid
	ErrInvalidInput = fmt.Errorf("invalid input")
)

// configCollector collects configuration using huh
type configCollector struct{}

// newConfigCollector creates a new configCollector instance
func newConfigCollector() *configCollector {
	return &configCollector{}
}

// Collect gathers configuration parameters from the user
func (c *configCollector) Collect() (*Config, error) {
	wd, _ := os.Getwd()

	var concurrencyStr string = "10"
	config := &Config{
		InputDir:    wd,
		OutputFile:  "output.txt",
		Concurrency: 10,
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("📁 Input Directory").
				Description("Directory containing images to process").
				Value(&config.InputDir).
				Placeholder(wd).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("input directory cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("💾 Output File").
				Description("Path where the output text will be saved").
				Value(&config.OutputFile).
				Placeholder("output.txt").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("output file cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("🔑 OpenAI API Key").
				Description("Your OpenAI API key for OCR operations").
				Value(&config.APIKey).
				Password(true).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("API key cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("⚡ Concurrency Level").
				Description("Number of images to process in parallel (default: 10)").
				Value(&concurrencyStr).
				Placeholder("10").
				Validate(func(s string) error {
					if s == "" {
						concurrencyStr = "10"
						return nil
					}
					conv, err := strconv.Atoi(s)
					if err != nil || conv <= 0 {
						return fmt.Errorf("must be a positive integer")
					}
					config.Concurrency = conv
					return nil
				}),

			huh.NewInput().
				Title("📅 Start Date (Optional)").
				Description("Date to use if the first page has no date. Leave empty to skip.").
				Value(&config.StartDate).
				Placeholder("e.g., Monday, January 1, 2024"),
		),
	).WithTheme(huh.ThemeBase16())

	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("error collecting configuration: %w", err)
	}

	// Ensure concurrency is set if validation didn't set it
	if config.Concurrency == 0 {
		config.Concurrency = 10
	}

	return config, nil
}
