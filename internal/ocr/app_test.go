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
		mockRepo.On("GetImageNames", "").Return(testFiles, nil)
		mockRepo.On("LoadImageByName", "", "Img-0001.jpg").Return([]byte("image1"), nil)
		mockRepo.On("LoadImageByName", "", "Img-0002.jpg").Return([]byte("image2"), nil)
		mockRepo.On("SaveOutput", mock.Anything, mock.Anything).Return(nil)

		// Setup OCR client mocks
		mockClient.On("OCRImage", mock.Anything, []byte("image1")).Return("Monday, January 1, 2024\nTest text 1", nil)
		mockClient.On("OCRImage", mock.Anything, []byte("image2")).Return("Test text 2", nil)

		// Create output file path
		outputFile := filepath.Join(tmpDir, "output.txt")

		// Create app and process
		app := NewApp(mockClient, mockRepo)
		config := &AppConfig{
			OutputFile:  outputFile,
			Concurrency: 2,
			StartDate:   "",
		}

		err = app.ProcessImages(context.Background(), config)
		assert.NoError(t, err)

		// Verify all mocks were called
		mockRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)

		// Verify SaveOutput was called with the correct file path
		mockRepo.AssertCalled(t, "SaveOutput", outputFile, mock.MatchedBy(func(content string) bool {
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

		mockRepo.On("GetImageNames", "").Return([]string{}, nil)

		app := NewApp(mockClient, mockRepo)
		config := &AppConfig{
			OutputFile:  filepath.Join(tmpDir, "output.txt"),
			Concurrency: 2,
		}

		err = app.ProcessImages(context.Background(), config)
		assert.Error(t, err)
		assert.Equal(t, ErrNoImagesFound, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockClient := new(MockOCRClient)

		mockRepo.On("GetImageNames", "").Return(nil, os.ErrNotExist)

		app := NewApp(mockClient, mockRepo)
		config := &AppConfig{
			OutputFile:  "output.txt",
			Concurrency: 2,
		}

		err := app.ProcessImages(context.Background(), config)
		assert.Error(t, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestApp_formatOutput(t *testing.T) {
	app := NewApp(nil, nil)

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

	assert.Contains(t, output, "---")
	assert.Contains(t, output, "Img-0001.jpg")
	assert.Contains(t, output, "Monday, January 1, 2024")
	assert.Contains(t, output, "First page text")
	assert.Contains(t, output, "Img-0002.jpg")
	assert.Contains(t, output, "Monday, January 1, 2024") // Should carry forward
	assert.Contains(t, output, "Second page text")
	assert.Contains(t, output, "Img-0003.jpg")
	assert.Contains(t, output, "Tuesday, January 2, 2024")
	assert.Contains(t, output, "Third page text")
}

func TestApp_formatOutput_WithStartDate(t *testing.T) {
	app := NewApp(nil, nil)

	results := []OCRResult{
		{
			ImageName: "Img-0001.jpg",
			Date:      "", // No date in first image
			Text:      "First page text",
		},
	}

	output := app.formatOutput(results, "Sunday, December 31, 2023")

	assert.Contains(t, output, "Sunday, December 31, 2023")
	assert.Contains(t, output, "First page text")
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

func TestApp_GetCost(t *testing.T) {
	mockClient := new(MockOCRClient)
	mockRepo := new(MockRepository)

	mockClient.On("GetCost").Return(0.50, 0.10)

	app := NewApp(mockClient, mockRepo)
	total, perImage := app.GetCost()

	assert.Equal(t, 0.50, total)
	assert.Equal(t, 0.10, perImage)
	mockClient.AssertExpectations(t)
}
