package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepository_GetImageNames(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "ocr_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo := New("")

	// Test empty directory
	names, err := repo.GetImageNames(tmpDir)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("Expected empty slice, got %v", names)
	}

	// Create test image files
	testFiles := []string{"Img-0001.jpg", "Img-0002.jpg", "Img-0003.png", "notanimage.txt"}
	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test getting image names
	names, err = repo.GetImageNames(tmpDir)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(names) != 3 {
		t.Errorf("Expected 3 images, got %d", len(names))
	}
	expected := []string{"Img-0001.jpg", "Img-0002.jpg", "Img-0003.png"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, name)
		}
	}

	// Test non-existent directory
	_, err = repo.GetImageNames("/nonexistent/dir")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
	if err != ErrDirectoryNotFound && !os.IsNotExist(err) {
		t.Errorf("Expected ErrDirectoryNotFound or IsNotExist, got %v", err)
	}
}

func TestRepository_LoadImageByName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ocr_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo := New("")

	// Create test image file
	testFile := "test.jpg"
	testContent := []byte("test image content")
	path := filepath.Join(tmpDir, testFile)
	if err := os.WriteFile(path, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test loading existing file
	data, err := repo.LoadImageByName(tmpDir, testFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if string(data) != string(testContent) {
		t.Errorf("Expected %s, got %s", string(testContent), string(data))
	}

	// Test loading non-existent file
	_, err = repo.LoadImageByName(tmpDir, "nonexistent.jpg")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	// Check if error contains ErrImageNotFound or is os.ErrNotExist
	if err != nil && err.Error() != "image not found: nonexistent.jpg" && !os.IsNotExist(err) {
		t.Errorf("Expected error containing 'image not found' or IsNotExist, got %v", err)
	}
}

func TestRepository_SaveOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ocr_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo := New("")

	// Test saving output
	outputPath := filepath.Join(tmpDir, "output.txt")
	content := "test output content"
	err = repo.SaveOutput(outputPath, content)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify file was created and content is correct
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	if string(data) != content {
		t.Errorf("Expected %s, got %s", content, string(data))
	}
}
