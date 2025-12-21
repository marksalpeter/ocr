package ocr

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
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
}

// App represents the main application logic
type App struct {
	ocrClient OCRClient
	repo      Repository
	config    *AppConfig
}

// NewApp creates a new App instance with the given configuration
func NewApp(ocrClient OCRClient, repo Repository, config *AppConfig) *App {
	return &App{
		ocrClient: ocrClient,
		repo:      repo,
		config:    config,
	}
}

// ProcessImages processes all images in the specified directory
func (a *App) ProcessImages(ctx context.Context) (*ProcessImageResults, error) {
	// Get image names (uses repository's base directory)
	imageNames, err := a.repo.GetImageNames()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoImagesFound, err)
	}
	if len(imageNames) == 0 {
		return nil, ErrNoImagesFound
	}

	// Process images in parallel
	results := a.processImagesParallel(ctx, imageNames)

	// Sort results by image name to maintain order
	sort.Slice(results, func(i, j int) bool {
		return results[i].ImageName < results[j].ImageName
	})

	// Format and concatenate output
	output := a.formatOutput(results, a.config.StartDate)

	// Save output
	if err := a.repo.SaveOutput(output); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProcessingFailed, err)
	}

	// Calculate totals
	totalImages := len(results)
	var totalCost float64
	for _, result := range results {
		totalCost += result.Cost
	}

	var costPerImage float64
	if totalImages > 0 {
		costPerImage = totalCost / float64(totalImages)
	}

	return &ProcessImageResults{
		TotalImagesProcessed: totalImages,
		TotalCost:            totalCost,
		CostPerImage:         costPerImage,
	}, nil
}

// processImagesParallel processes images in parallel with configurable concurrency
func (a *App) processImagesParallel(ctx context.Context, imageNames []string) []OCRResult {
	concurrency := a.config.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	// Create channels for work and results
	jobs := make(chan string, len(imageNames))
	results := make(chan OCRResult, len(imageNames))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for imageName := range jobs {
				result := a.processImage(ctx, imageName)
				results <- result
			}
		}()
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for _, name := range imageNames {
			select {
			case <-ctx.Done():
				return
			case jobs <- name:
			}
		}
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []OCRResult
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults
}

// processImage processes a single image
func (a *App) processImage(ctx context.Context, imageName string) OCRResult {
	result := OCRResult{
		ImageName: imageName,
	}

	// Load image (uses repository's base directory)
	imageData, err := a.repo.LoadImageByName(imageName)
	if err != nil {
		result.Text = fmt.Sprintf("Error loading image: %v", err)
		return result
	}

	// Perform OCR
	text, cost, err := a.ocrClient.OCRImage(ctx, imageData)
	if err != nil {
		result.Text = fmt.Sprintf("Error processing image: %v", err)
		return result
	}

	result.Text = text
	result.Cost = cost

	// Extract date from text (look at first few lines)
	date := extractDate(text)
	result.Date = date

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
