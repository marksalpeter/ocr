package resizer

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/image/draw"
)

// createTestImage creates a test image with the specified dimensions
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a simple pattern for testing
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	return img
}

// encodeJPEG encodes an image as JPEG
func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	return buf.Bytes(), err
}

// encodePNG encodes an image as PNG
func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}

func TestResizer_ResizeImage_SmallImage(t *testing.T) {
	r := New()

	// Create a small image (570x562) that doesn't need resizing
	img := createTestImage(570, 562)
	imageData, err := encodeJPEG(img)
	assert.NoError(t, err)

	// Resize with maxDimension 1500 (should return unchanged)
	result, err := r.ResizeImage(imageData, 1500)
	assert.NoError(t, err)
	assert.Equal(t, imageData, result, "Small image should be returned unchanged")
}

func TestResizer_ResizeImage_LargeImage(t *testing.T) {
	r := New()

	// Create a large image (4032x2707) that needs resizing
	img := createTestImage(4032, 2707)
	imageData, err := encodeJPEG(img)
	assert.NoError(t, err)

	// Resize with maxDimension 1500
	result, err := r.ResizeImage(imageData, 1500)
	assert.NoError(t, err)
	assert.NotEqual(t, imageData, result, "Large image should be resized")

	// Verify the resized image dimensions
	decoded, _, err := r.decodeImage(result)
	assert.NoError(t, err)
	bounds := decoded.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Longest dimension should be 1500
	longestDim := width
	if height > width {
		longestDim = height
	}
	assert.Equal(t, 1500, longestDim, "Longest dimension should be 1500")

	// Verify aspect ratio is maintained
	// Original: 4032/2707 ≈ 1.489
	// Resized: width/height should be approximately the same
	expectedWidth := (4032 * 1500) / 4032 // = 1500
	expectedHeight := (2707 * 1500) / 4032 // ≈ 1006
	assert.Equal(t, expectedWidth, width)
	assert.InDelta(t, expectedHeight, height, 1, "Height should maintain aspect ratio")
}

func TestResizer_ResizeImage_PortraitImage(t *testing.T) {
	r := New()

	// Create a tall portrait image (1000x2000)
	img := createTestImage(1000, 2000)
	imageData, err := encodeJPEG(img)
	assert.NoError(t, err)

	// Resize with maxDimension 1500
	result, err := r.ResizeImage(imageData, 1500)
	assert.NoError(t, err)

	// Verify the resized image dimensions
	decoded, _, err := r.decodeImage(result)
	assert.NoError(t, err)
	bounds := decoded.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Height should be 1500 (longest dimension)
	assert.Equal(t, 1500, height, "Height should be 1500")
	// Width should maintain aspect ratio: (1000 * 1500) / 2000 = 750
	assert.Equal(t, 750, width, "Width should maintain aspect ratio")
}

func TestResizer_ResizeImage_LandscapeImage(t *testing.T) {
	r := New()

	// Create a wide landscape image (3000x1500)
	img := createTestImage(3000, 1500)
	imageData, err := encodeJPEG(img)
	assert.NoError(t, err)

	// Resize with maxDimension 1500
	result, err := r.ResizeImage(imageData, 1500)
	assert.NoError(t, err)

	// Verify the resized image dimensions
	decoded, _, err := r.decodeImage(result)
	assert.NoError(t, err)
	bounds := decoded.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Width should be 1500 (longest dimension)
	assert.Equal(t, 1500, width, "Width should be 1500")
	// Height should maintain aspect ratio: (1500 * 1500) / 3000 = 750
	assert.Equal(t, 750, height, "Height should maintain aspect ratio")
}

func TestResizer_ResizeImage_PNGFormat(t *testing.T) {
	r := New()

	// Create a large PNG image
	img := createTestImage(2000, 2000)
	imageData, err := encodePNG(img)
	assert.NoError(t, err)

	// Resize with maxDimension 1500
	result, err := r.ResizeImage(imageData, 1500)
	assert.NoError(t, err)
	assert.NotEqual(t, imageData, result, "Large PNG image should be resized")

	// Verify it's still a valid PNG
	decoded, format, err := r.decodeImage(result)
	assert.NoError(t, err)
	assert.Equal(t, "png", format, "Format should remain PNG")
	bounds := decoded.Bounds()
	assert.Equal(t, 1500, bounds.Dx(), "Width should be 1500")
	assert.Equal(t, 1500, bounds.Dy(), "Height should be 1500")
}

func TestResizer_ResizeImage_InvalidMaxDimension(t *testing.T) {
	r := New()

	img := createTestImage(100, 100)
	imageData, err := encodeJPEG(img)
	assert.NoError(t, err)

	// Test with zero maxDimension
	_, err = r.ResizeImage(imageData, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maxDimension must be positive")

	// Test with negative maxDimension
	_, err = r.ResizeImage(imageData, -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maxDimension must be positive")
}

func TestResizer_ResizeImage_InvalidImageData(t *testing.T) {
	r := New()

	// Test with invalid image data
	invalidData := []byte("not an image")
	_, err := r.ResizeImage(invalidData, 1500)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode image")
}

func TestResizer_ResizeImage_ExactThreshold(t *testing.T) {
	r := New()

	// Create an image exactly at the threshold (1500x1500)
	img := createTestImage(1500, 1500)
	imageData, err := encodeJPEG(img)
	assert.NoError(t, err)

	// Resize with maxDimension 1500 (should return unchanged)
	result, err := r.ResizeImage(imageData, 1500)
	assert.NoError(t, err)
	assert.Equal(t, imageData, result, "Image at exact threshold should be returned unchanged")
}

func TestResizer_ResizeImage_JustOverThreshold(t *testing.T) {
	r := New()

	// Create an image just over the threshold (1501x1501)
	img := createTestImage(1501, 1501)
	imageData, err := encodeJPEG(img)
	assert.NoError(t, err)

	// Resize with maxDimension 1500
	result, err := r.ResizeImage(imageData, 1500)
	assert.NoError(t, err)
	assert.NotEqual(t, imageData, result, "Image just over threshold should be resized")

	// Verify dimensions
	decoded, _, err := r.decodeImage(result)
	assert.NoError(t, err)
	bounds := decoded.Bounds()
	assert.Equal(t, 1500, bounds.Dx(), "Width should be 1500")
	assert.Equal(t, 1500, bounds.Dy(), "Height should be 1500")
}

