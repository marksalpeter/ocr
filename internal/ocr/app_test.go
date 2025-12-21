package ocr

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestApp_ProcessImages(t *testing.T) {
	t.Run("successful processing", func(t *testing.T) {
		// Create temporary directory with test images
		tmpDir, err := os.MkdirTemp("", "ocr_test_*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create test image files
		testFiles := []string{"Img-0001.jpg", "Img-0002.jpg"}
		for _, f := range testFiles {
			path := filepath.Join(tmpDir, f)
			err := os.WriteFile(path, []byte("test image"), 0644)
			assert.NoError(t, err)
		}

		// Create mocks
		mockRepo := new(MockRepository)
		mockClient := new(MockOCRClient)

		// Setup repository mocks
		mockRepo.On("GetImageNames").Return(testFiles, nil)
		mockRepo.On("LoadImageByName", "Img-0001.jpg").Return([]byte("image1"), nil)
		mockRepo.On("LoadImageByName", "Img-0002.jpg").Return([]byte("image2"), nil)
		mockRepo.On("SaveOutput", mock.Anything).Return(nil)

		// Setup resizer mock (returns image unchanged for tests)
		mockResizer := new(MockResizer)
		mockResizer.On("ResizeImage", []byte("image1"), 1500).Return([]byte("image1"), nil)
		mockResizer.On("ResizeImage", []byte("image2"), 1500).Return([]byte("image2"), nil)

		// Setup OCR client mocks
		mockClient.On("ValidateAPIKey", mock.Anything).Return(nil)
		mockClient.On("OCRImage", mock.Anything, []byte("image1")).Return("Monday, January 1, 2024\nTest text 1", 0.01, 1, nil)
		mockClient.On("OCRImage", mock.Anything, []byte("image2")).Return("Test text 2", 0.01, 1, nil)

		// Create app config
		config := &AppConfig{
			Concurrency: 2,
			StartDate:   "",
		}

		// Create app and process
		app := NewApp(mockClient, mockRepo, mockResizer, config)

		results, err := app.ProcessImages(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.Equal(t, 2, results.TotalImagesProcessed)
		// Verify attempt tracking: both images had 1 attempt each = 2 total
		assert.Equal(t, 2, results.TotalOCRAttempts)
		assert.InDelta(t, 1.0, results.OCRAttemptsPerImage, 0.0001)

		// Verify all mocks were called
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)

		// Verify SaveOutput was called with the correct content
		mockRepo.AssertCalled(t, "SaveOutput", mock.MatchedBy(func(content string) bool {
			return assert.Contains(t, content, "Img-0001.jpg") &&
				assert.Contains(t, content, "Img-0002.jpg") &&
				assert.Contains(t, content, "Monday, January 1, 2024") &&
				assert.Contains(t, content, "Test text 1") &&
				assert.Contains(t, content, "Test text 2")
		}))
	})

	t.Run("no images found", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "ocr_test_*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		mockRepo := new(MockRepository)
		mockClient := new(MockOCRClient)

		mockRepo.On("GetImageNames").Return([]string{}, nil)
		mockClient.On("ValidateAPIKey", mock.Anything).Return(nil)

		mockResizer := new(MockResizer)

		config := &AppConfig{
			Concurrency: 2,
		}
		app := NewApp(mockClient, mockRepo, mockResizer, config)

		_, err = app.ProcessImages(context.Background())
		assert.Error(t, err)
		assert.Equal(t, ErrNoImagesFound, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockClient := new(MockOCRClient)

		mockRepo.On("GetImageNames").Return(nil, os.ErrNotExist)
		mockClient.On("ValidateAPIKey", mock.Anything).Return(nil)

		mockResizer := new(MockResizer)

		config := &AppConfig{
			Concurrency: 2,
		}
		app := NewApp(mockClient, mockRepo, mockResizer, config)

		_, err := app.ProcessImages(context.Background())
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestApp_formatOutput(t *testing.T) {
	mockResizer := new(MockResizer)
	app := NewApp(nil, nil, mockResizer, &AppConfig{})

	results := []OCRResult{
		{
			ImageName: "Img-0001.jpg",
			Date:      "Monday, January 1, 2024",
			Text:      "First page text",
		},
		{
			ImageName: "Img-0002.jpg",
			Date:      "", // No date, should carry forward
			Text:      "Second page text",
		},
		{
			ImageName: "Img-0003.jpg",
			Date:      "Tuesday, January 2, 2024",
			Text:      "Third page text",
		},
	}

	output := app.formatOutput(results, "Sunday, December 31, 2023")

	expected := `---
Img-0001.jpg
Monday, January 1, 2024
First page text
---
Img-0002.jpg
Monday, January 1, 2024
Second page text
---
Img-0003.jpg
Tuesday, January 2, 2024
Third page text
`
	assert.Equal(t, expected, output)
}

func TestApp_formatOutput_WithStartDate(t *testing.T) {
	mockResizer := new(MockResizer)
	app := NewApp(nil, nil, mockResizer, &AppConfig{})

	results := []OCRResult{
		{
			ImageName: "Img-0001.jpg",
			Date:      "", // No date in first image, should use start date
			Text:      "First page text",
		},
	}

	output := app.formatOutput(results, "Sunday, December 31, 2023")

	expected := `---
Img-0001.jpg
Sunday, December 31, 2023
First page text
`
	assert.Equal(t, expected, output)
}

func TestExtractDate(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "date at top",
			text:     "Monday, January 1, 2024\nSome text here",
			expected: "Monday, January 1, 2024",
		},
		{
			name:     "date without day",
			text:     "January 1, 2024\nSome text",
			expected: "January 1, 2024",
		},
		{
			name:     "date with slashes",
			text:     "01/01/2024\nSome text",
			expected: "01/01/2024",
		},
		{
			name:     "no date",
			text:     "Some text without date",
			expected: "",
		},
		{
			name:     "date in second line",
			text:     "\nMonday, January 1, 2024\nSome text",
			expected: "Monday, January 1, 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDate(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApp_ProcessImages_Results(t *testing.T) {
	// Create temporary directory with test images
	tmpDir, err := os.MkdirTemp("", "ocr_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test image files
	testFiles := []string{"Img-0001.jpg", "Img-0002.jpg"}
	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		err := os.WriteFile(path, []byte("test image"), 0644)
		assert.NoError(t, err)
	}

	// Create mocks
	mockRepo := new(MockRepository)
	mockClient := new(MockOCRClient)

	// Setup repository mocks
	mockRepo.On("GetImageNames").Return(testFiles, nil)
	mockRepo.On("LoadImageByName", "Img-0001.jpg").Return([]byte("image1"), nil)
	mockRepo.On("LoadImageByName", "Img-0002.jpg").Return([]byte("image2"), nil)
	mockRepo.On("SaveOutput", mock.Anything).Return(nil)

	// Setup resizer mock (returns image unchanged for tests)
	mockResizer := new(MockResizer)
	mockResizer.On("ResizeImage", []byte("image1"), 1500).Return([]byte("image1"), nil)
	mockResizer.On("ResizeImage", []byte("image2"), 1500).Return([]byte("image2"), nil)

	// Setup OCR client mocks with different costs
	mockClient.On("ValidateAPIKey", mock.Anything).Return(nil)
	mockClient.On("OCRImage", mock.Anything, []byte("image1")).Return("Test text 1", 0.10, 1, nil)
	mockClient.On("OCRImage", mock.Anything, []byte("image2")).Return("Test text 2", 0.20, 2, nil)

	// Create app config
	config := &AppConfig{
		Concurrency: 2,
		StartDate:   "",
	}

	// Create app and process
	app := NewApp(mockClient, mockRepo, mockResizer, config)

	results, err := app.ProcessImages(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Equal(t, 2, results.TotalImagesProcessed)
	assert.InDelta(t, 0.30, results.TotalCost, 0.0001)
	assert.InDelta(t, 0.15, results.CostPerImage, 0.0001)
	// Verify attempt tracking: image1 had 1 attempt, image2 had 2 attempts = 3 total
	assert.Equal(t, 3, results.TotalOCRAttempts)
	assert.InDelta(t, 1.5, results.OCRAttemptsPerImage, 0.0001)

	mockRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}
