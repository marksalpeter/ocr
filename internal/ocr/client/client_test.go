package client

import (
	"context"
	"os"
	"testing"
)

var testKey = os.Getenv("OPENAI_API_KEY")

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		Status:  404,
		Message: "Not found",
	}
	expected := "API error (status 404): Not found"
	if err.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, err.Error())
	}
}

func TestClient_ValidateAPIKey(t *testing.T) {
	c := New(testKey)
	ctx := context.Background()

	err := c.ValidateAPIKey(ctx)
	if err != nil {
		t.Errorf("Expected API key validation to succeed, got error: %v", err)
	}
}

func TestClient_OCRImage_ErrorCase(t *testing.T) {
	c := New(testKey)
	ctx := context.Background()

	// Create a minimal test image (1x1 pixel PNG)
	// PNG signature: 89 50 4E 47 0D 0A 1A 0A
	testImageData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	text, cost, err := c.OCRImage(ctx, testImageData)

	// The test key doesn't have permission for vision API, so we expect an error
	if err == nil {
		t.Error("Expected OCRImage to fail with test key (no vision API permission), but it succeeded")
	}

	// Text should be empty on error
	if text != "" {
		t.Errorf("Expected empty text on error, got: %s", text)
	}

	// Cost should be 0 or positive (may have attempted retries)
	if cost < 0 {
		t.Errorf("Expected non-negative cost, got: %f", cost)
	}

	// Verify it's an API error
	apiErr, ok := err.(*APIError)
	if !ok {
		// Could also be ErrMaxRetriesExceeded after retries
		if err.Error() == "max retries exceeded" {
			// This is acceptable - retries exhausted
			return
		}
		t.Errorf("Expected APIError or max retries error, got: %T - %v", err, err)
		return
	}

	// Should be a permission/unauthorized error
	if apiErr.Status != 401 && apiErr.Status != 403 {
		t.Logf("Got API error with status %d (expected 401 or 403 for permission error): %s", apiErr.Status, apiErr.Message)
	}
}
