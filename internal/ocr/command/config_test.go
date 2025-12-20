package command

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigCollector(t *testing.T) {
	collector := newConfigCollector()
	assert.NotNil(t, collector)
}

func TestConfigModel_Init(t *testing.T) {
	model := newConfigModel()
	cmd := model.Init()
	assert.Nil(t, cmd)
}

func TestConfigModel_Update_Cancel(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyType
	}{
		{"ctrl+c", tea.KeyCtrlC},
		{"q", tea.KeyRunes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newConfigModel()
			msg := tea.KeyMsg{Type: tt.key}
			if tt.key == tea.KeyRunes {
				msg.Runes = []rune{'q'}
			}

			updatedModel, cmd := model.Update(msg)
			assert.NotNil(t, cmd) // tea.Quit is a function, so we just check it's not nil
			updated, ok := updatedModel.(*configModel)
			assert.True(t, ok)
			assert.True(t, updated.cancelled)
		})
	}
}

func TestConfigModel_Update_Enter(t *testing.T) {
	model := newConfigModel()
	model.inputDir = "/test/dir"
	model.outputFile = "output.txt"
	model.apiKey = "test-key"
	model.concurrency = "5"

	// Step 0: Input directory
	msg := tea.KeyMsg{Type: tea.KeyEnter, Runes: nil}
	updatedModel, cmd := model.Update(msg)
	assert.Nil(t, cmd)
	updated, ok := updatedModel.(*configModel)
	assert.True(t, ok)
	assert.Equal(t, 1, updated.step)

	// Step 1: Output file
	updatedModel, cmd = updated.Update(msg)
	assert.Nil(t, cmd)
	updated, ok = updatedModel.(*configModel)
	assert.True(t, ok)
	assert.Equal(t, 2, updated.step)

	// Step 2: API key
	updatedModel, cmd = updated.Update(msg)
	assert.Nil(t, cmd)
	updated, ok = updatedModel.(*configModel)
	assert.True(t, ok)
	assert.Equal(t, 3, updated.step)

	// Step 3: Concurrency
	updatedModel, cmd = updated.Update(msg)
	assert.Nil(t, cmd)
	updated, ok = updatedModel.(*configModel)
	assert.True(t, ok)
	assert.Equal(t, 4, updated.step)

	// Step 4: Start date (completes)
	updatedModel, cmd = updated.Update(msg)
	assert.NotNil(t, cmd) // tea.Quit is a function
	updated, ok = updatedModel.(*configModel)
	assert.True(t, ok)
	assert.True(t, updated.completed)
	assert.NotNil(t, updated.config)
	assert.Equal(t, "/test/dir", updated.config.InputDir)
	assert.Equal(t, "output.txt", updated.config.OutputFile)
	assert.Equal(t, "test-key", updated.config.APIKey)
	assert.Equal(t, 5, updated.config.Concurrency)
}

func TestConfigModel_Update_Enter_ValidationErrors(t *testing.T) {
	t.Run("empty input directory", func(t *testing.T) {
		model := newConfigModel()
		model.inputDir = ""

		msg := tea.KeyMsg{Type: tea.KeyEnter, Runes: nil}
		updatedModel, cmd := model.Update(msg)
		assert.Nil(t, cmd)
		updated, ok := updatedModel.(*configModel)
		assert.True(t, ok)
		assert.Equal(t, 0, updated.step)
		assert.Contains(t, updated.err, "cannot be empty")
	})

	t.Run("empty output file", func(t *testing.T) {
		model := newConfigModel()
		model.inputDir = "/test"
		model.outputFile = ""
		model.step = 1 // Set to step 1 (output file)

		msg := tea.KeyMsg{Type: tea.KeyEnter, Runes: nil}
		updatedModel, cmd := model.Update(msg)
		assert.Nil(t, cmd)
		updated, ok := updatedModel.(*configModel)
		assert.True(t, ok)
		assert.Equal(t, 1, updated.step)
		assert.Contains(t, updated.err, "cannot be empty")
	})

	t.Run("empty API key", func(t *testing.T) {
		model := newConfigModel()
		model.inputDir = "/test"
		model.outputFile = "output.txt"
		model.apiKey = ""
		model.step = 2 // Set to step 2 (API key)

		msg := tea.KeyMsg{Type: tea.KeyEnter, Runes: nil}
		updatedModel, cmd := model.Update(msg)
		assert.Nil(t, cmd)
		updated, ok := updatedModel.(*configModel)
		assert.True(t, ok)
		assert.Equal(t, 2, updated.step)
		assert.Contains(t, updated.err, "cannot be empty")
	})

	t.Run("invalid concurrency", func(t *testing.T) {
		model := newConfigModel()
		model.inputDir = "/test"
		model.outputFile = "output.txt"
		model.apiKey = "test-key"
		model.concurrency = "invalid"
		model.step = 3 // Set to step 3 (concurrency)

		msg := tea.KeyMsg{Type: tea.KeyEnter, Runes: nil}
		updatedModel, cmd := model.Update(msg)
		assert.Nil(t, cmd)
		updated, ok := updatedModel.(*configModel)
		assert.True(t, ok)
		assert.Equal(t, 3, updated.step)
		assert.Contains(t, updated.err, "positive integer")
	})

	t.Run("zero concurrency", func(t *testing.T) {
		model := newConfigModel()
		model.inputDir = "/test"
		model.outputFile = "output.txt"
		model.apiKey = "test-key"
		model.concurrency = "0"
		model.step = 3 // Set to step 3 (concurrency)

		msg := tea.KeyMsg{Type: tea.KeyEnter, Runes: nil}
		updatedModel, cmd := model.Update(msg)
		assert.Nil(t, cmd)
		updated, ok := updatedModel.(*configModel)
		assert.True(t, ok)
		assert.Equal(t, 3, updated.step)
		assert.Contains(t, updated.err, "positive integer")
	})
}

func TestConfigModel_Update_Backspace(t *testing.T) {
	model := newConfigModel()
	model.inputDir = "test"
	model.outputFile = "test"
	model.apiKey = "test"
	model.concurrency = "123"
	model.startDate = "test"

	msg := tea.KeyMsg{Type: tea.KeyBackspace, Runes: nil}

	// Test backspace on input directory
	model.step = 0
	updatedModel, _ := model.Update(msg)
	updated, _ := updatedModel.(*configModel)
	assert.Equal(t, "tes", updated.inputDir)

	// Test backspace on output file
	model.step = 1
	updatedModel, _ = model.Update(msg)
	updated, _ = updatedModel.(*configModel)
	assert.Equal(t, "tes", updated.outputFile)

	// Test backspace on API key
	model.step = 2
	updatedModel, _ = model.Update(msg)
	updated, _ = updatedModel.(*configModel)
	assert.Equal(t, "tes", updated.apiKey)

	// Test backspace on concurrency
	model.step = 3
	updatedModel, _ = model.Update(msg)
	updated, _ = updatedModel.(*configModel)
	assert.Equal(t, "12", updated.concurrency)

	// Test backspace on start date
	model.step = 4
	updatedModel, _ = model.Update(msg)
	updated, _ = updatedModel.(*configModel)
	assert.Equal(t, "tes", updated.startDate)
}

func TestConfigModel_Update_Input(t *testing.T) {
	model := newConfigModel()

	// Test input on input directory
	model.step = 0
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updatedModel, _ := model.Update(msg)
	updated, _ := updatedModel.(*configModel)
	assert.Contains(t, updated.inputDir, "a")

	// Test input on output file
	model.step = 1
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	updatedModel, _ = model.Update(msg)
	updated, _ = updatedModel.(*configModel)
	assert.Contains(t, updated.outputFile, "b")

	// Test input on API key
	model.step = 2
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updatedModel, _ = model.Update(msg)
	updated, _ = updatedModel.(*configModel)
	assert.Contains(t, updated.apiKey, "c")

	// Test numeric input on concurrency
	model.step = 3
	model.concurrency = ""
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}}
	updatedModel, _ = model.Update(msg)
	updated, _ = updatedModel.(*configModel)
	assert.Equal(t, "5", updated.concurrency)

	// Test non-numeric input on concurrency (should be ignored)
	model.step = 3
	model.concurrency = ""
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updatedModel, _ = model.Update(msg)
	updated, _ = updatedModel.(*configModel)
	assert.Equal(t, "", updated.concurrency)

	// Test input on start date
	model.step = 4
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updatedModel, _ = model.Update(msg)
	updated, _ = updatedModel.(*configModel)
	assert.Contains(t, updated.startDate, "d")
}

func TestConfigModel_View(t *testing.T) {
	model := newConfigModel()
	view := model.View()

	assert.Contains(t, view, "OCR Configuration")
	assert.Contains(t, view, "Input directory")
	assert.Contains(t, view, "Output file")
	assert.Contains(t, view, "OpenAI API key")
	assert.Contains(t, view, "Concurrency level")
	assert.Contains(t, view, "Start date")
	assert.Contains(t, view, "Press Enter to continue")

	// Test cancelled state
	model.cancelled = true
	view = model.View()
	assert.Contains(t, view, "Configuration cancelled")

	// Test completed state
	model.cancelled = false
	model.completed = true
	view = model.View()
	assert.Contains(t, view, "Configuration completed")

	// Test error display
	model.completed = false
	model.err = "test error"
	view = model.View()
	assert.Contains(t, view, "test error")
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"short string", "abc", "***"},
		{"4 chars", "abcd", "****"},                             // 4 chars should be masked
		{"long string", "sk-test123456789", "sk-t************"}, // Shows first 4, then masks rest
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
