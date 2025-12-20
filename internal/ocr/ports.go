package ocr

import (
	"context"
)

// OCRClient defines the interface for OCR operations
type OCRClient interface {
	// OCRImage processes an image and returns the transcribed text
	OCRImage(ctx context.Context, imageData []byte) (text string, err error)
	// ValidateAPIKey validates the OpenAI API key
	ValidateAPIKey(ctx context.Context) error
	// GetCost returns the total cost and cost per image
	GetCost() (totalCost float64, costPerImage float64)
}

// Repository defines the interface for file operations
type Repository interface {
	// GetImageNames returns sorted image filenames from the specified directory
	GetImageNames(dir string) ([]string, error)
	// LoadImageByName loads image data by filename from the specified directory
	LoadImageByName(dir, filename string) ([]byte, error)
	// SaveOutput saves the output text to the specified path
	SaveOutput(path string, content string) error
}

// OCRResult represents the result of processing a single image
type OCRResult struct {
	ImageName string
	Date      string
	Text      string
	Cost      float64
}
