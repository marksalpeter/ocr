package command

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/marksalpeter/ocr/internal/ocr"
	"github.com/marksalpeter/ocr/internal/ocr/client"
	"github.com/marksalpeter/ocr/internal/ocr/repository"
	"github.com/marksalpeter/ocr/internal/ocr/resizer"
)

// Command represents the command adapter that orchestrates the OCR workflow
type Command struct {
	configCollector *configCollector
	logger          *log.Logger
	spinner         *spinner
}

// New creates a new Command instance
func New() *Command {
	logger := log.New(os.Stderr)
	logger.SetReportCaller(false)
	logger.SetReportTimestamp(false)

	return &Command{
		configCollector: newConfigCollector(),
		logger:          logger,
		spinner:         new(spinner),
	}
}

// Run executes the OCR workflow: collects configuration, processes images, and displays results
func (c *Command) Run(ctx context.Context) error {
	// Collect configuration
	cfg, err := c.configCollector.Collect()
	if err != nil {
		c.logger.Error("Error collecting configuration", "error", err)
		return err
	}

	// Create repository with the input directory and output file from config
	repo, err := repository.New(cfg.InputDir, cfg.OutputFile)
	if err != nil {
		c.logger.Error("Error creating repository", "error", err)
		return err
	}

	// Create the OCR client with the API key from config
	ocrClient := client.New(cfg.APIKey)

	// Create resizer instance
	imgResizer := resizer.New()

	// Start the loading spinner
	c.spinner.Start("Processing images...")

	// Create application instance (spinner implements ProgressUpdater)
	app := ocr.NewApp(ocrClient, repo, imgResizer, c.spinner, &ocr.AppConfig{
		Concurrency: cfg.Concurrency,
		StartDate:   cfg.StartDate,
	})

	// Process images
	results, err := app.ProcessImages(ctx)
	if err != nil {
		c.spinner.Stop()
		c.logger.Error("Failed to process images", "error", err)
		return err
	}

	// Stop the loading spinner
	c.spinner.Stop()

	// Display results
	c.logger.Info("✅ Processing completed", "results", results)

	return nil
}
