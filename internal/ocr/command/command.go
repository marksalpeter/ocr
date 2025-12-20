package command

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/marksalpeter/ocr/internal/ocr"
	"github.com/marksalpeter/ocr/internal/ocr/client"
	"github.com/marksalpeter/ocr/internal/ocr/repository"
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
	repo := repository.New(cfg.InputDir, cfg.OutputFile)

	// Create the OCR client with the API key from config
	ocrClient := client.New(cfg.APIKey)

	// Create application instance
	app := ocr.NewApp(ocrClient, repo, &ocr.AppConfig{
		Concurrency: cfg.Concurrency,
		StartDate:   cfg.StartDate,
	})

	// Start the loading spinner\
	c.spinner.Start("Processing images...")

	// Process images
	if err := app.ProcessImages(ctx); err != nil {
		c.logger.Error("Failed to process images", "error", err)
		return err
	}

	// Stop the loading spinner
	c.spinner.Stop()

	// Display results
	totalCost, costPerImage := app.GetCost()
	c.logger.Info("Processing completed",
		"totalCost", fmt.Sprintf("$%.4f", totalCost),
		"costPerImage", fmt.Sprintf("$%.4f", costPerImage),
		"outputFile", cfg.OutputFile)

	return nil
}
