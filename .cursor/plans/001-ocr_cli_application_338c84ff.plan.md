# OCR Image to Text Command - Implementation Plan

## Overview

Build a Go CLI application that processes journal images using OpenAI's OCR API. The application will process images in parallel, extract dates, maintain text formatting, and track costs.

## Architecture

The application follows a clean architecture pattern with:

- **Domain layer** (`internal/ocr`): Ports (interfaces), DTOs, entities, and application logic
- **Adapters** (submodules of `internal/ocr`):
- `client`: OpenAI API client
- `repository`: File system operations
- `command`: Bubbletea CLI for configuration
- **Entry point** (`cmd/ocr`): Dependency injection and initialization

## Implementation Tasks

### 1. Project Setup and Structure

- Initialize Go module (`go.mod`)
- Create directory structure:
- `cmd/ocr/` - Main entry point
- `internal/ocr/` - Domain package (ports, DTOs, entities, errors)
- `internal/ocr/client/` - OpenAI adapter
- `internal/ocr/repository/` - File system adapter
- `internal/ocr/command/` - CLI adapter (bubbletea)
- Set up `.gitignore` (already exists, may need updates)
- Add dependencies: OpenAI SDK, bubbletea, mockery

### 2. Domain Package - Core Types and Ports

**File**: `internal/ocr/ports.go`

- Define port interfaces:
- `OCRClient` - Interface for OCR operations
- `Repository` - Interface for file operations (getImageNames, loadImageByName, saveOutput)
- `ConfigCollector` - Interface for collecting configuration
- Define DTOs for:
- Image processing request/response
- OCR result with date and text
- Configuration parameters
- Define error variables for domain errors

**File**: `internal/ocr/errors.go`

- Define error variables for application-level errors

### 3. Repository Adapter - File Operations

**File**: `internal/ocr/repository/repository.go`

- Implement `Repository` interface:
- `GetImageNames(dir string) ([]string, error)` - Get sorted image filenames
- `LoadImageByName(dir, filename string) ([]byte, error)` - Load image bytes
- `SaveOutput(path string, content string) error` - Save output text
- Define adapter-specific errors as variables
- Handle file system errors

**File**: `internal/ocr/repository/repository_test.go`

- Test happy path and error cases

### 4. OpenAI Client Adapter

**File**: `internal/ocr/client/client.go`

- Implement `OCRClient` interface:
- `OCRImage(imageData []byte) (text string, err error)` - Process single image
- `ValidateAPIKey(apiKey string) error` - Validate API key using OpenAI endpoint
- Implement retry logic (up to 5 retries for failed API calls)
- Define API error struct with status and message
- Define adapter-specific errors as variables
- Track API costs (tokens used, cost calculation)

**File**: `internal/ocr/client/client_test.go`

- Test happy path, retry logic, and error cases
- Mock HTTP client for testing

### 5. Application Logic - Orchestration

**File**: `internal/ocr/app.go`

- Implement main application logic:
- Process images in parallel (configurable concurrency, default 10)
- Sort images alphabetically
- Extract dates from OCR results (carry forward if missing)
- Format output: horizontal rule, image name, date, transcript
- Concatenate results in order
- Track total cost and cost per image
- Handle date extraction and carry-forward logic
- Maintain line breaks, punctuation, and spacing from OCR

**File**: `internal/ocr/app_test.go`

- Use mockery-generated mocks for ports
- Test happy path and error paths
- Test date carry-forward logic
- Test parallel processing

### 6. Command Adapter - Bubbletea CLI

**File**: `internal/ocr/command/command.go`

- Implement `ConfigCollector` interface using bubbletea
- Collect configuration parameters:
- Input directory (default: current working directory)
- Output filename (default: "output.txt")
- OpenAI API key (required, validated)
- Concurrency level (default: 10)
- Start date (optional, for first page if no date found)
- Create interactive TUI for configuration collection

**File**: `internal/ocr/command/command_test.go`

- Test configuration collection
- Test validation logic

### 7. Main Function and Dependency Injection

**File**: `cmd/ocr/main.go`

- Initialize all adapters
- Wire dependencies:
- Create repository instance
- Create OpenAI client instance
- Create command/CLI instance
- Create application instance with all dependencies
- Execute application flow
- Print results: output file location, total cost, cost per image

### 8. Testing Infrastructure

- Set up mockery for generating mocks
- Create mock files for all port interfaces
- Add test fixtures (sample images if needed)
- Ensure test coverage for happy paths and error cases

### 9. Documentation and Finalization

- Add README with usage instructions
- Document configuration options
- Verify all requirements are met
- Run integration tests

## Key Implementation Details

### Date Extraction Logic

- Extract date from top of page OCR result
- If no date found, use date from previous page
- If first page has no date, use configurable start date
- Date format should be preserved as transcribed

### Parallel Processing

- Use worker pool pattern with configurable concurrency (default 10)
- Process images in parallel but maintain alphabetical order in output
- Collect results and sort before concatenation

### Cost Tracking

- Track API usage (tokens) from OpenAI responses
- Calculate cost based on OpenAI pricing
- Display total cost and cost per image at completion

### Error Handling

- Retry failed API calls up to 5 times
- Define errors as variables for testability
- Return structured errors with context

## Dependencies

- OpenAI Go SDK (or HTTP client for API calls)