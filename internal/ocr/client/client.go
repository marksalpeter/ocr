package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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

// OCRImage processes an image and returns the transcribed text and total cost from all attempts
func (c *Client) OCRImage(ctx context.Context, imageData []byte) (text string, totalCost float64, err error) {
	maxRetries := 5
	var lastErr error
	totalCost = 0

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, 8s, 16s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return "", totalCost, ctx.Err()
			case <-time.After(backoff):
			}
		}

		text, cost, err := c.ocrImageOnce(ctx, imageData)
		totalCost += cost // Accumulate cost from all attempts
		if err == nil {
			return text, totalCost, nil
		}

		lastErr = err
		// Don't retry on authentication errors
		if apiErr, ok := err.(*APIError); ok && apiErr.Status == http.StatusUnauthorized {
			return "", totalCost, err
		}
	}

	return "", totalCost, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
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
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: "Please transcribe all text from this image exactly as it appears, preserving all line breaks, punctuation, spacing, and wording. If there is a date at the top of the page, include it. Otherwise, just transcribe the text.",
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
		MaxTokens: 4096,
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

	// Calculate cost based on GPT-4 Vision pricing
	// Input: $0.01 per 1K tokens, Output: $0.03 per 1K tokens
	// For simplicity, we'll use a rough estimate based on response tokens
	inputTokens := float64(resp.Usage.PromptTokens)
	outputTokens := float64(resp.Usage.CompletionTokens)
	cost = (inputTokens/1000.0)*0.01 + (outputTokens/1000.0)*0.03

	return text, cost, nil
}

