package resizer

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

// Resizer implements the ocr.Resizer interface for image resizing operations
type Resizer struct{}

// New creates a new Resizer instance
func New() *Resizer {
	return &Resizer{}
}

// ResizeImage resizes an image if its longest dimension exceeds maxDimension, maintaining aspect ratio
func (r *Resizer) ResizeImage(imageData []byte, maxDimension int) ([]byte, error) {
	if maxDimension <= 0 {
		return nil, fmt.Errorf("maxDimension must be positive")
	}

	// Decode image to determine format and dimensions
	img, format, err := r.decodeImage(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Find the longest dimension
	longestDim := width
	if height > width {
		longestDim = height
	}

	// If image is already small enough, return original
	if longestDim <= maxDimension {
		return imageData, nil
	}

	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		// Landscape: width is the longest
		newWidth = maxDimension
		newHeight = (height * maxDimension) / width
	} else {
		// Portrait or square: height is the longest
		newHeight = maxDimension
		newWidth = (width * maxDimension) / height
	}

	// Create new image with calculated dimensions
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Resize using high-quality resampling
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Encode back to the same format
	return r.encodeImage(dst, format)
}

// decodeImage decodes image data and returns the image, format, and error
func (r *Resizer) decodeImage(data []byte) (image.Image, string, error) {
	// Try to detect format by attempting to decode
	reader := bytes.NewReader(data)

	// Try WebP first (needs external library)
	if img, err := webp.Decode(reader); err == nil {
		return img, "webp", nil
	}

	// Reset reader for next attempt
	reader.Seek(0, 0)

	// Try PNG
	if img, err := png.Decode(reader); err == nil {
		return img, "png", nil
	}

	// Reset reader for next attempt
	reader.Seek(0, 0)

	// Try JPEG
	if img, err := jpeg.Decode(reader); err == nil {
		return img, "jpeg", nil
	}

	// Reset reader for next attempt
	reader.Seek(0, 0)

	// Try GIF
	if img, err := gif.Decode(reader); err == nil {
		return img, "gif", nil
	}

	return nil, "", fmt.Errorf("unsupported image format or invalid image data")
}

// encodeImage encodes an image to the specified format
func (r *Resizer) encodeImage(img image.Image, format string) ([]byte, error) {
	var buf bytes.Buffer

	switch format {
	case "jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}
	case "png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}
	case "gif":
		if err := gif.Encode(&buf, img, nil); err != nil {
			return nil, fmt.Errorf("failed to encode GIF: %w", err)
		}
	case "webp":
		// WebP encoding is more complex and requires additional library
		// For now, encode as JPEG as fallback
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
			return nil, fmt.Errorf("failed to encode WebP (fallback to JPEG): %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format for encoding: %s", format)
	}

	return buf.Bytes(), nil
}

