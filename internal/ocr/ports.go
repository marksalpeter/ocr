package ocr

import (
	"context"
)

// OCRClient defines the interface for OCR operations
//
//go:generate go run github.com/vektra/mockery/v2 --name OCRClient
type OCRClient interface {
	// OCRImage processes an image and returns the transcribed text, total cost from all attempts, and the number of attempts made
	OCRImage(ctx context.Context, imageData []byte) (text string, cost float64, attempts int, err error)
	// ValidateAPIKey validates the OpenAI API key
	ValidateAPIKey(ctx context.Context) error
}

// Repository defines the interface for file operations
//
//go:generate go run github.com/vektra/mockery/v2 --name Repository
type Repository interface {
	// GetImageNames returns sorted image filenames from the repository's base directory
	GetImageNames() ([]string, error)
	// LoadImageByName loads image data by filename from the repository's base directory
	LoadImageByName(filename string) ([]byte, error)
	// SaveOutput saves the output text to the repository's configured output path
	SaveOutput(content string) error
}

// Resizer defines the interface for image resizing operations
//
//go:generate go run github.com/vektra/mockery/v2 --name Resizer
type Resizer interface {
	// ResizeImage resizes an image if its longest dimension exceeds maxDimension, maintaining aspect ratio
	ResizeImage(imageData []byte, maxDimension int) ([]byte, error)
}

// ProgressUpdater defines the interface for updating progress during image processing
type ProgressUpdater interface {
	// UpdateProgress is called after each image is processed with the current count and total
	UpdateProgress(completed, total int)
}

// OCRResult represents the result of processing a single image
type OCRResult struct {
	ImageName   string
	Date        string
	Text        string
	Cost        float64
	OCRAttempts int
	Error       error
}
