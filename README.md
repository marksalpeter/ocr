# OCR - Journal Image to Text Converter

A command-line tool that uses OpenAI's GPT-4 Vision API to perform OCR (Optical Character Recognition) on journal images, extracting handwritten and printed text with automatic date extraction and carry-forward logic.

## Features

- **Parallel Processing**: Process multiple images concurrently with configurable concurrency
- **Automatic Date Extraction**: Extracts dates from journal pages and carries them forward when missing
- **Image Resizing**: Automatically resizes large images (max 1500px) to optimize API usage and reduce costs
- **Progress Tracking**: Real-time progress indicator showing `[N / M]` images processed
- **Cost Tracking**: Displays total cost and cost per image
- **Retry Logic**: Automatic retries with exponential backoff for failed API calls
- **Beautiful CLI**: Interactive configuration using `huh` with a modern, step-by-step form interface

## Prerequisites

- **macOS** (tested on macOS 14+)
- **OpenAI API Key** with access to GPT-4 Vision API ([Get your API key](https://platform.openai.com/api-keys))

> **Note**: Go is only required if building from source. For most users, downloading the pre-built binary is recommended.

## Installation

### Option 1: Download Pre-built Binary (Recommended)

1. Download the latest release from GitHub [Releases](https://github.com/marksalpeter/ocr/releases) page or download directly:   
```bash
# For Apple Silicon (M1/M2/M3)
curl -L -o ocr https://github.com/marksalpeter/ocr/releases/latest/download/ocr-darwin-arm64

# For Intel Macs
curl -L -o ocr https://github.com/marksalpeter/ocr/releases/latest/download/ocr-darwin-amd64
```

2. Make it executable:
```bash
chmod +x ocr
```

3. Move to a directory in your PATH:
```bash
sudo mv ocr /usr/local/bin/
```

4. Verify installation:
```bash
ocr
```

If you see the interactive configuration form, installation was successful!

### Option 2: Build from Source

Assuming you have go installed and `$GOPATH/bin` or `$HOME/go/bin` is properly 
configured in your PATH, then you can install the app in one line.

```bash
go install github.com/marksalpeter/ocr/cmd/ocr@latest
```


## Usage

### Basic Usage

1. Navigate to the directory containing your journal images:
```bash
cd /path/to/journal/images
```

2. Run the OCR tool:
```bash
ocr
```

3. The tool will prompt you interactively for:
   - **Input Directory**: Directory containing images (defaults to current directory)
   - **Output File**: Path where transcribed text will be saved (defaults to `output.txt`)
   - **OpenAI API Key**: Your OpenAI API key (input is masked)
   - **Concurrency Level**: Number of images to process in parallel (default: 10)
   - **Start Date** (Optional): Date to use if the first page has no date

4. The tool will process all images and display progress:
```
Processing images... [5 / 20]
```

5. Upon completion, you'll see a summary:
```
✅ Processing completed
total images processed: 20
total cost:             $0.123
cost per image:         $0.006
total ocr attempts:     25
ocr attempts per image: 1.25
total duration:         2m30s
duration per image:     7s
```

### Output Format

The output file contains transcribed text for each image in the following format:

```
---
image-001.jpg
Monday, January 1, 2024
[Transcribed text from the image, preserving all line breaks and formatting]
---
image-002.jpg
Monday, January 1, 2024
[Transcribed text - date carried forward from previous page]
---
image-003.jpg
Tuesday, January 2, 2024
[Transcribed text with new date found on the page]
```

### Supported Image Formats

- JPEG (.jpg, .jpeg)
- PNG (.png)
- GIF (.gif)
- WebP (.webp)
- BMP (.bmp)

Images are automatically resized if they exceed 1500px on the longest side to optimize API usage and reduce costs.

## Configuration

### Concurrency

The default concurrency is 10 images processed in parallel. You can adjust this based on:
- Your API rate limits
- Available network bandwidth
- Desired processing speed

Higher concurrency = faster processing but more API calls simultaneously.

### Start Date

If your first journal page doesn't have a date, you can provide a start date that will be used until a date is found in subsequent pages. Dates are automatically extracted from the top of pages and carried forward when missing.

## Cost Estimation

The tool uses GPT-4 Vision API pricing:
- **Input**: $0.01 per 1K tokens
- **Output**: $0.03 per 1K tokens

Typical costs:
- Small images (resized to ~1500px): ~$0.005-0.01 per image
- Large images with lots of text: ~$0.01-0.02 per image

The tool displays total cost and cost per image after processing completes.

## Troubleshooting

### "No images found"
- Ensure you're in the correct directory or specify the full path to your images directory
- Check that your images have supported file extensions (.jpg, .png, .gif, .webp, .bmp)

### API Errors
- Verify your OpenAI API key is correct and has access to GPT-4 Vision API
- Check your API usage limits and billing status
- The tool automatically retries failed requests up to 5 times

### Refusal Responses
- If GPT refuses to transcribe certain images, the tool will retry automatically
- Refusals are tracked and displayed in the error output
- Very rare content may still be refused after retries

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o ocr ./cmd/ocr
```

### Project Structure

```
ocr/
├── cmd/ocr/          # Main entry point
├── internal/ocr/     # Core domain logic
│   ├── client/       # OpenAI API client
│   ├── repository/   # File system operations
│   ├── resizer/      # Image resizing
│   └── command/      # CLI command and configuration
└── demo/             # Example images
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

