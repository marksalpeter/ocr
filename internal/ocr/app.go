package ocr

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

// AppConfig contains only the configuration parameters needed by the app
type AppConfig struct {
	Concurrency int
	StartDate   string
}

// ProcessImageResults contains the results of processing images
type ProcessImageResults struct {
	TotalImagesProcessed int
	TotalCost            float64
	CostPerImage         float64
	TotalOCRAttempts     int
	OCRAttemptsPerImage  float64
	TotalDuration        time.Duration
	DurationPerImage     time.Duration
}

func (r ProcessImageResults) String() string {
	return fmt.Sprintf("total images processed: %d\ntotal cost:             $%.3f\ncost per image:         $%.3f\ntotal ocr attempts:     %d\nocr attempts per image: %.2f\ntotal duration:         %s\nduration per image:     %s\n",
		r.TotalImagesProcessed, r.TotalCost, r.CostPerImage, r.TotalOCRAttempts, r.OCRAttemptsPerImage,
		r.TotalDuration.Round(time.Millisecond), r.DurationPerImage.Round(time.Millisecond))
}

// App represents the main application logic
type App struct {
	ocrClient       OCRClient
	repo            Repository
	resizer         Resizer
	progressUpdater ProgressUpdater
	config          *AppConfig
}

// NewApp creates a new App instance with the given configuration
func NewApp(ocrClient OCRClient, repo Repository, resizer Resizer, progressUpdater ProgressUpdater, config *AppConfig) *App {
	return &App{
		ocrClient:       ocrClient,
		repo:            repo,
		resizer:         resizer,
		progressUpdater: progressUpdater,
		config:          config,
	}
}

// ProcessImages processes all images in the specified directory
func (a *App) ProcessImages(ctx context.Context) (*ProcessImageResults, error) {
	// Validate API key
	if err := a.ocrClient.ValidateAPIKey(ctx); err != nil {
		return nil, fmt.Errorf("invalid api key: %w", err)
	}

	// Get image names (uses repository's base directory)
	imageNames, err := a.repo.GetImageNames()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoImagesFound, err)
	} else if len(imageNames) == 0 {
		return nil, ErrNoImagesFound
	}

	// Process images in parallel
	results := a.processImagesParallel(ctx, imageNames)

	// Format and concatenate output
	output := a.formatOutput(results, a.config.StartDate)

	// Save output
	if err := a.repo.SaveOutput(output); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProcessingFailed, err)
	}

	// Calculate total cost, total attempts, and total duration
	var totalCost float64
	var totalAttempts int
	var totalDuration time.Duration
	for _, result := range results {
		totalCost += result.Cost
		totalAttempts += result.OCRAttempts
		totalDuration += result.Duration
	}

	// Return results
	return &ProcessImageResults{
		TotalImagesProcessed: len(results),
		TotalCost:            totalCost,
		CostPerImage:         totalCost / float64(len(results)),
		TotalOCRAttempts:     totalAttempts,
		OCRAttemptsPerImage:  float64(totalAttempts) / float64(len(results)),
		TotalDuration:        totalDuration,
		DurationPerImage:     totalDuration / time.Duration(len(results)),
	}, nil
}

// processImagesParallel processes images in parallel with configurable concurrency
func (a *App) processImagesParallel(ctx context.Context, imageNames []string) []OCRResult {
	concurrency := a.config.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	// Pre-allocate results slice
	results := make([]OCRResult, len(imageNames))
	total := len(imageNames)
	var completed int64

	// Semaphore to limit concurrency
	sem := make(chan struct{}, concurrency)
	defer close(sem)

	// Process each image
	for i, imageName := range imageNames {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return results
		default:
		}

		sem <- struct{}{}
		go func(idx int, name string) {
			// Process image and write directly to results at index
			results[idx] = a.processImage(ctx, name)

			// Update progress after processing
			if a.progressUpdater != nil {
				current := int(atomic.AddInt64(&completed, 1))
				a.progressUpdater.UpdateProgress(current, total)
			}

			<-sem
		}(i, imageName)
	}

	// Wait for all goroutines to complete
	for range cap(sem) {
		sem <- struct{}{}
	}

	return results
}

// processImage processes a single image
func (a *App) processImage(ctx context.Context, imageName string) OCRResult {
	startTime := time.Now()

	var result OCRResult
	result.ImageName = imageName

	// Load image (uses repository's base directory)
	imageData, err := a.repo.LoadImageByName(imageName)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		return result
	}

	// Resize if needed (max 1500px on longest side)
	imageData, err = a.resizer.ResizeImage(imageData, 1500)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		return result
	}

	// Perform OCR
	text, cost, attempts, err := a.ocrClient.OCRImage(ctx, imageData)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		return result
	}

	// Return the result
	result.Date = extractDate(text)
	result.Text = text
	result.Cost = cost
	result.OCRAttempts = attempts
	result.Duration = time.Since(startTime)
	return result
}

// extractDate extracts a date from the beginning of the text
// Looks for common date patterns at the top of the page
func extractDate(text string) string {
	lines := strings.Split(text, "\n")
	// Check first 5 lines for date
	for i := 0; i < len(lines) && i < 5; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// Try to match common date patterns
		datePatterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\w+day,?\s+)?(\w+\s+\d{1,2},?\s+\d{4})`), // "Monday, January 1, 2024" or "January 1, 2024"
			regexp.MustCompile(`(?i)(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})`),         // "1/1/2024" or "01-01-2024"
			regexp.MustCompile(`(?i)(\w+\s+\d{1,2},?\s+\d{4})`),               // "January 1, 2024"
		}
		for _, pattern := range datePatterns {
			if match := pattern.FindString(line); match != "" {
				return match
			}
		}
	}
	return ""
}

// formatOutput formats the results into the final output string
func (a *App) formatOutput(results []OCRResult, startDate string) string {
	var builder strings.Builder
	lastDate := startDate

	for _, result := range results {
		// Horizontal rule
		builder.WriteString("---\n")

		// Image name
		builder.WriteString(result.ImageName)
		builder.WriteString("\n")

		if result.Error != nil {
			builder.WriteString("Error: ")
			builder.WriteString(result.Error.Error())
			builder.WriteString("\n")
			continue
		}

		// Date (use extracted date or carry forward)
		date := result.Date
		if date == "" {
			date = lastDate
		} else {
			lastDate = date
		}
		if date != "" {
			builder.WriteString(date)
			builder.WriteString("\n")
		}

		// Transcript
		builder.WriteString(result.Text)
		builder.WriteString("\n")
	}

	return builder.String()
}
