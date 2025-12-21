package ocr

import (
	"context"
)

// OCRClient defines the interface for OCR operations
type OCRClient interface {
	// OCRImage processes an image and returns the transcribed text and total cost from all attempts
	OCRImage(ctx context.Context, imageData []byte) (text string, cost float64, err error)
	// ValidateAPIKey validates the OpenAI API key
	ValidateAPIKey(ctx context.Context) error
}

// Repository defines the interface for file operations
type Repository interface {
	// GetImageNames returns sorted image filenames from the repository's base directory
	GetImageNames() ([]string, error)
	// LoadImageByName loads image data by filename from the repository's base directory
	LoadImageByName(filename string) ([]byte, error)
	// SaveOutput saves the output text to the repository's configured output path
	SaveOutput(content string) error
}

// OCRResult represents the result of processing a single image
type OCRResult struct {
	ImageName string
	Date      string
	Text      string
	Cost      float64
}
