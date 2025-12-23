package command

import (
	"context"
	"fmt"
	"sync"

	huhSpinner "github.com/charmbracelet/huh/spinner"
)

type spinner struct {
	cancelFunc context.CancelFunc
	mu         sync.Mutex
	title      string
	spinner    *huhSpinner.Spinner
}

func (s *spinner) Start(title string) {
	s.Stop()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.title = title

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel

	s.spinner = huhSpinner.New().
		Type(huhSpinner.Dots).
		Title(title).
		Context(ctx)

	go s.spinner.Run()
}

func (s *spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFunc == nil {
		return
	}

	s.cancelFunc()
	s.cancelFunc = nil
	s.spinner = nil
	s.title = ""
}

// UpdateProgress implements the ProgressUpdater interface
func (s *spinner) UpdateProgress(completed, total int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.spinner == nil {
		return
	}

	s.spinner.Title(fmt.Sprintf("%s [%d / %d]", s.title, completed, total))
}
