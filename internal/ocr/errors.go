package ocr

import "errors"

// Application-level errors
var (
	ErrNoImagesFound     = errors.New("no images found in directory")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrDateExtractionFailed = errors.New("failed to extract date from image")
	ErrProcessingFailed   = errors.New("failed to process images")
)

