package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/marksalpeter/ocr/internal/ocr/command"
)

func main() {
	// Create a context that cancels on interrupt or terminate signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create command instance
	cmd := command.New()

	// Run the command
	if err := cmd.Run(ctx); err != nil {
		os.Exit(1)
	}
}
