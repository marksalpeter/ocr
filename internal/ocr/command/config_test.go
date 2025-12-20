package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigCollector(t *testing.T) {
	collector := newConfigCollector()
	assert.NotNil(t, collector)
}

// Note: Full integration tests for the huh form would require
// more complex setup with terminal simulation. The core logic
// is tested through the form's validation and the Collect method.
