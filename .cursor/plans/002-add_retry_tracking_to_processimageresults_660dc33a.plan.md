---
name: Add OCRAttempts Tracking to ProcessImageResults
overview: Add OCR attempt tracking to ProcessImageResults by updating the OCRClient interface to return attempt count, tracking attempts in OCRResult, and calculating total attempts and attempts per image in ProcessImageResults.
todos: []
---

# Add OCRAttempts Tracking to ProcessImageResults

## Overview

Track and report OCR attempt statistics by updating the OCRClient to return attempt count, adding attempt tracking to OCRResult, and including total attempts and attempts per image in ProcessImageResults.

## Implementation Details

### 1. Update OCRClient Interface (`internal/ocr/ports.go`)

- Change `OCRImage` signature from `(text string, cost float64, err error)` to `(text string, cost float64, attempts int, err error)`
- The attempts count should represent the number of attempts made (1 = first attempt succeeded, 2+ = retries occurred)

### 2. Update Client Implementation (`internal/ocr/client/client.go`)

- Modify `OCRImage` to track the number of attempts made
- Return the attempt count (which equals retries + 1, or just the attempt number)
- Update the return statement to include attempt count

### 3. Update OCRResult (`internal/ocr/ports.go`)

- Add `OCRAttempts int` field to `OCRResult` struct to track attempts per image

### 4. Update ProcessImageResults (`internal/ocr/app.go`)

- Add `TotalOCRAttempts int` field
- Add `OCRAttemptsPerImage float64` field (calculated as total attempts / total images)

### 5. Update App Logic (`internal/ocr/app.go`)

- In `processImage()`, capture attempt count from `OCRImage()` call
- Store attempt count in `OCRResult.OCRAttempts`
- In `ProcessImages()`, accumulate total attempts from all results
- Calculate `OCRAttemptsPerImage` in the returned `ProcessImageResults`

### 6. Update Command (`internal/ocr/command/command.go`)

- Update logging to display attempt statistics alongside cost information

### 7. Update Tests

- Update all mock expectations for `OCRImage` to include attempt count return value
- Update `app_test.go` to verify attempt tracking
- Update `client_test.go` if needed to verify attempt count is returned correctly

## File Changes

1. **`internal/ocr/ports.go`**: Update `OCRClient.OCRImage()` signature, add `OCRAttempts` to `OCRResult`, add attempt fields to `ProcessImageResults`
2. **`internal/ocr/client/client.go`**: Update `OCRImage()` to track and return attempt count
3. **`internal/ocr/app.go`**: Update `processImage()` to capture attempts, update `ProcessImages()` to calculate attempt statistics
4. **`internal/ocr/command/command.go`**: Update logging to show attempt statistics
5. **`internal/ocr/app_test.go`**: Update all mock expectations and add attempt verification tests
6. **`internal/ocr/client/client_test.go`**: Update tests if needed

## Attempt Count Logic

- Attempt count = number of attempts made (1 = first attempt succeeded, 2 = one retry, etc.)
- Total attempts = sum of all attempt counts across all images
- Attempts per image = total attempts / total images processed