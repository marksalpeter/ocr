---
name: Image Resizer Adapter
overview: Add an image resizer adapter that resizes images only when the longest side exceeds 1500px, maintaining aspect ratio. The resizer will be integrated into the processing pipeline between the repository and OCR client.
todos: []
---

# Image Re

sizer Adapter Implementation

## Overview

Add an image resizer adapter that automatically resizes images when the longest dimension exceeds 1500px, maintaining aspect ratio. This will reduce file sizes, improve API performance, and reduce costs while maintaining sufficient quality for handwritten journal OCR.

## Architecture

The resizer will be a new adapter in `internal/ocr/resizer/` that implements a `Resizer` interface defined in the domain package. It will be integrated into the processing flow:

```javascript
Repository → Resizer → OCRClient
```



## Implementation Details

### 1. Create Resizer Port (`internal/ocr/ports.go`)

- Add `Resizer` interface with method `ResizeImage(imageData []byte, maxDimension int) ([]byte, error)`

- This allows the resizer to be mocked for testing

### 2. Create Resizer Adapter (`internal/ocr/resizer/resizer.go`)

- Implement the `Resizer` interface

- Use Go standard library `image`, `image/jpeg`, `image/png`, `image/gif`, `image/webp` packages

- For WebP support, may need `golang.org/x/image/webp` (external dependency)

- Logic:

- Decode image to determine format and dimensions

- If longest side <= maxDimension, return original image data unchanged

- Otherwise, calculate new dimensions maintaining aspect ratio

- Resize using a quality resampling algorithm (e.g., `draw.CatmullRom` or `draw.ApproxBiLinear`)

- Re-encode in the same format

- Return resized image data

### 3. Update App Logic (`internal/ocr/app.go`)

- Add `resizer Resizer` field to `App` struct

- Update `NewApp()` to accept `Resizer` parameter

- In `processImage()`, resize image data after loading but before OCR:
  ```go
    imageData, err := a.repo.LoadImageByName(imageName)
    // ... error handling ...
    
    // Resize if needed (max 1500px on longest side)
    imageData, err = a.resizer.ResizeImage(imageData, 1500)
    // ... error handling ...
    
    text, cost, err := a.ocrClient.OCRImage(ctx, imageData)
  ```




### 4. Update Command (`internal/ocr/command/command.go`)

- Create resizer instance: `resizer := resizer.New()`

- Pass resizer to `NewApp()` call

### 5. Update Main (`cmd/ocr/main.go`)

- No changes needed (resizer created in command layer)

### 6. Add Tests (`internal/ocr/resizer/resizer_test.go`)

- Test resizing large images (e.g., 4032x2707 → ~1500x1006)

- Test small images are unchanged (e.g., 570x562)

- Test aspect ratio preservation

- Test different image formats (JPEG, PNG, WebP)

- Test error cases (invalid image data, unsupported format)

### 7. Dependencies

- Add `golang.org/x/image/webp` for WebP support (if not using standard library)

- Standard library packages: `image`, `image/jpeg`, `image/png`, `image/gif`

- Standard library: `image/draw` for resampling

## File Changes

1. **`internal/ocr/ports.go`**: Add `Resizer` interface

2. **`internal/ocr/resizer/resizer.go`**: New file - resizer implementation

3. **`internal/ocr/resizer/resizer_test.go`**: New file - unit tests

4. **`internal/ocr/app.go`**: Add resizer field and integration

5. **`internal/ocr/command/command.go`**: Create and inject resizer

6. **`go.mod`**: Add `golang.org/x/image/webp` dependency (if needed)

7. **`.mockery.yaml`**: Add `Resizer` to mock generation (optional, for future use)

## Resizing Algorithm

- Use `draw.CatmullRom` or `draw.ApproxBiLinear` for quality resampling

- Maintain aspect ratio: `newWidth = maxDimension * (originalWidth / max(originalWidth, originalHeight))`

- Preserve original image format

- For JPEG, use quality 90-95 to balance file size and quality

## Edge Cases

- Images smaller than threshold: return unchanged

- Invalid image data: return error

- Unsupported formats: return error with clear message