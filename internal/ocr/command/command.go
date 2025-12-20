package command

import (
	"context"
	"fmt"
)

// Command represents the command adapter that orchestrates the OCR workflow
type Command struct {
	configCollector *configCollector
}

// New creates a new Command instance
func New() *Command {
	return &Command{
		configCollector: newConfigCollector(),
	}
}

// Run executes the OCR workflow: collects configuration, processes images, and displays results
func (c *Command) Run(ctx context.Context) error {
	// Collect configuration
	cfg, err := c.configCollector.Collect()
	if err != nil {
		return fmt.Errorf("error collecting configuration: %w", err)
	}

	// Run status model to process images and display results
	return runStatusModel(ctx, cfg)
}
