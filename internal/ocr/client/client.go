package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Client implements the ocr.OCRClient interface for OpenAI API operations
type Client struct {
	apiKey       string
	openAIClient *openai.Client
}

// APIError represents an error from the API with status code
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.Status, e.Message)
}

var (
	// ErrInvalidAPIKey is returned when the API key is invalid
	ErrInvalidAPIKey = fmt.Errorf("invalid API key")
	// ErrAPIRequestFailed is returned when an API request fails
	ErrAPIRequestFailed = fmt.Errorf("API request failed")
	// ErrMaxRetriesExceeded is returned when max retries are exceeded
	ErrMaxRetriesExceeded = fmt.Errorf("max retries exceeded")
	// ErrRefusalResponse is returned when GPT refuses to process an image
	ErrRefusalResponse = fmt.Errorf("GPT refused to process image")
)

// New creates a new Client instance
func New(apiKey string) *Client {
	openAIClient := openai.NewClient(apiKey)
	return &Client{
		apiKey:       apiKey,
		openAIClient: openAIClient,
	}
}

// ValidateAPIKey validates the OpenAI API key using the usage endpoint
func (c *Client) ValidateAPIKey(ctx context.Context) error {
	// Use the models endpoint to validate the key
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAPIKey, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidAPIKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrInvalidAPIKey
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			Status:  resp.StatusCode,
			Message: string(body),
		}
	}

	return nil
}

// OCRImage processes an image and returns the transcribed text, total cost from all attempts, and the number of attempts made
func (c *Client) OCRImage(ctx context.Context, imageData []byte) (text string, totalCost float64, attempts int, err error) {
	maxRetries := 5
	var lastErr error
	totalCost = 0
	attempts = 0

	for attempt := 0; attempt < maxRetries; attempt++ {
		attempts++
		if attempt > 0 {
			// Exponential backoff capped at 50ms
			backoff := max(time.Duration(1<<uint(attempt-1))*time.Millisecond, 10*time.Millisecond)
			select {
			case <-ctx.Done():
				return "", totalCost, attempts, ctx.Err()
			case <-time.After(backoff):
			}
		}

		text, cost, err := c.ocrImageOnce(ctx, imageData)
		totalCost += cost // Accumulate cost from all attempts
		if err == nil {
			return text, totalCost, attempts, nil
		}

		lastErr = err
		// Don't retry on authentication errors
		if apiErr, ok := err.(*APIError); ok && apiErr.Status == http.StatusUnauthorized {
			return "", totalCost, attempts, err
		}
	}

	return "", totalCost, attempts, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// ocrImageOnce performs a single OCR request
func (c *Client) ocrImageOnce(ctx context.Context, imageData []byte) (text string, cost float64, err error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Create the request
	req := openai.ChatCompletionRequest{
		Model: "gpt-4o", // Using gpt-4o which supports vision
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: `
You are a professional OCR (Optical Character Recognition) transcription service. 
Your sole purpose is to convert images of text into machine-readable text format. 
You are processing personal documents, journals, notes, and written materials for archival and digitization purposes. 

Your task is to transcribe ALL visible text exactly as it appears, including:
- Handwritten text
- Printed text
- Dates and timestamps
- Any visible characters or symbols
- Preserving Line breaks
- Preserving Spacing
- Preserving Punctuation

Do not summarize, interpret, or modify the text, simply transcribe what you see.
The user owns all content in these images and has authorized this transcription. 
Please, do not refuse to transcribe the image.
`,
			},
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: "This is an image of a document page. Please transcribe all text visible in this image exactly as it appears, preserving all line breaks, punctuation, spacing, and wording. Do not include any other text in your response.",
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
						},
					},
				},
			},
		},
		MaxTokens:   4096,
		Temperature: 0.1, // Lower temperature for more consistent, literal transcription
	}

	resp, err := c.openAIClient.CreateChatCompletion(ctx, req)
	if err != nil {
		// Try to extract API error details
		if apiErr, ok := err.(*openai.APIError); ok {
			return "", 0, &APIError{
				Status:  apiErr.HTTPStatusCode,
				Message: apiErr.Message,
			}
		}
		return "", 0, fmt.Errorf("%w: %v", ErrAPIRequestFailed, err)
	}

	if len(resp.Choices) == 0 {
		return "", 0, fmt.Errorf("%w: no choices in response", ErrAPIRequestFailed)
	}

	text = resp.Choices[0].Message.Content

	// Check if GPT refused to process the image
	if c.isRefusalResponse(text) {
		return "", cost, fmt.Errorf("%w: %s", ErrRefusalResponse, text)
	}

	// Calculate cost based on GPT-4 Vision pricing
	// Input: $0.01 per 1K tokens, Output: $0.03 per 1K tokens
	// For simplicity, we'll use a rough estimate based on response tokens
	inputTokens := float64(resp.Usage.PromptTokens)
	outputTokens := float64(resp.Usage.CompletionTokens)
	cost = (inputTokens/1000.0)*0.01 + (outputTokens/1000.0)*0.03

	return text, cost, nil
}

// isRefusalResponse checks if the response indicates GPT refused to process the image
func (c *Client) isRefusalResponse(text string) bool {
	if text == "" {
		return false
	}

	textLower := strings.ToLower(strings.TrimSpace(text))

	// First, check for the most common refusal pattern: "sorry" + "can't/cannot" + "transcribe"
	// This catches variations like "I'm sorry, I can't transcribe the text from the image"
	if strings.Contains(textLower, "sorry") {
		if strings.Contains(textLower, "transcribe") {
			if strings.Contains(textLower, "can't") || strings.Contains(textLower, "cannot") || strings.Contains(textLower, "unable") {
				return true
			}
		}
	}

	// Check for very short responses that are likely refusals
	if len(text) < 100 {
		// Common refusal phrases in short responses
		shortRefusalPatterns := []string{
			"i'm sorry",
			"i can't",
			"i cannot",
			"unable to",
			"can't assist",
			"can't help",
			"can't transcribe",
			"cannot transcribe",
			"unable to transcribe",
			"i'm unable",
			"sorry, i can't",
		}
		for _, pattern := range shortRefusalPatterns {
			if strings.Contains(textLower, pattern) {
				return true
			}
		}
	}

	// Comprehensive refusal patterns
	refusalPatterns := []string{
		"i'm sorry, i can't",
		"i'm sorry, i cannot",
		"i can't assist",
		"i cannot assist",
		"i'm unable to assist",
		"i cannot help",
		"i can't help",
		"i'm sorry, i can't help",
		"i'm sorry, i can't assist",
		"i'm sorry, i cannot assist",
		"i'm sorry, i can't transcribe",
		"i'm sorry, i cannot transcribe",
		"i can't transcribe",
		"i cannot transcribe",
		"unable to transcribe",
		"can't transcribe",
		"cannot transcribe",
		"can't transcribe the text",
		"cannot transcribe the text",
		"unable to transcribe the text",
		"can't transcribe text from",
		"cannot transcribe text from",
		"unable to transcribe text from",
		"can't transcribe the text from the image",
		"cannot transcribe the text from the image",
		"unable to transcribe the text from the image",
		"can't transcribe the text from this image",
		"cannot transcribe the text from this image",
		"unable to transcribe the text from this image",
		"content policy",
		"against my usage policies",
		"against my policies",
		"inappropriate content",
		"violates my",
		"against my guidelines",
		"i'm not able to",
		"i am not able to",
		"not able to transcribe",
		"not able to assist",
		"not able to help",
	}

	for _, pattern := range refusalPatterns {
		if strings.Contains(textLower, pattern) {
			return true
		}
	}

	return false
}
