package client

import (
	"testing"
)

func TestIsRefusalResponse(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "exact refusal message",
			text:     "I'm sorry, I can't transcribe the text from the image.",
			expected: true,
		},
		{
			name:     "refusal with start date prefix",
			text:     "Start Date\nI'm sorry, I can't transcribe the text from the image.",
			expected: true,
		},
		{
			name:     "refusal with newlines",
			text:     "I'm sorry, I can't transcribe the text from the image.\n",
			expected: true,
		},
		{
			name:     "refusal without period",
			text:     "I'm sorry, I can't transcribe the text from the image",
			expected: true,
		},
		{
			name:     "refusal with this",
			text:     "I'm sorry, I can't transcribe the text from this image",
			expected: true,
		},
		{
			name:     "short refusal",
			text:     "I can't transcribe",
			expected: true,
		},
		{
			name:     "valid transcription",
			text:     "The day before my birthday my sister asked me to choose the lighting scheme.",
			expected: false,
		},
		{
			name:     "valid short text",
			text:     "E3",
			expected: false,
		},
		{
			name:     "refusal with sorry and can't transcribe",
			text:     "I'm sorry, I can't transcribe text from the image.",
			expected: true,
		},
	}

	c := New("test-key")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.isRefusalResponse(tt.text)
			if result != tt.expected {
				t.Errorf("isRefusalResponse(%q) = %v, want %v", tt.text, result, tt.expected)
			}
		})
	}
}
