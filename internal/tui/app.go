package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/studiowebux/bujotui/internal/complete"
	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/service"
	"github.com/studiowebux/bujotui/internal/term"
)

// App is the main TUI application.
type App struct {
	svc       *service.EntryService
	cfg       *config.Config
	completer *complete.Completer

	state         *ViewState
	date          time.Time
	entries       []model.Entry // filtered view of current day's entries
	allDay        []model.Entry // unfiltered entries for the current day
	entryIndexMap []int         // maps filtered index -> allDay index

	tty   *os.File
	ttyFd uintptr

	sizeMu sync.Mutex // protects Width/Height from SIGWINCH goroutine
}

// New creates a new TUI App.
func New(svc *service.EntryService, cfg *config.Config) *App {
	comp := complete.New(
		cfg.Symbols.SymbolNames(),
		cfg.Projects,
		cfg.People,
	)

	return &App{
		svc:       svc,
		cfg:       cfg,
		completer: comp,
		state:     NewViewState(cfg),
		date:      time.Now(),
	}
}

// Run starts the TUI main loop.
func (a *App) Run() error {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open /dev/tty: %w", err)
	}
	defer tty.Close()

	a.tty = tty
	a.ttyFd = tty.Fd()

	restore, err := term.EnableRawMode(a.ttyFd)
	if err != nil {
		return fmt.Errorf("enable raw mode: %w", err)
	}
	defer restore()

	term.UseAlternateScreen(a.tty)
	defer term.UseMainScreen(a.tty)

	// Get terminal size
	a.updateSize()

	// Handle terminal resize
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	go func() {
		for range sigCh {
			a.updateSize()
			a.render()
		}
	}()

	// Load initial data
	if err := a.loadEntries(); err != nil {
		return err
	}

	a.render()

	buf := make([]byte, 64)
	for {
		n, err := syscall.Read(int(a.ttyFd), buf) // #nosec G115 -- ttyFd is a valid file descriptor
		if err != nil {
			// SIGWINCH interrupts read with EINTR, just retry
			if err == syscall.EINTR {
				continue
			}
			break
		}
		if n == 0 {
			continue
		}

		key := ParseKey(buf, n)
		quit := a.handleKey(key)
		if quit {
			break
		}

		a.render()
	}

	return nil
}

// handleKey dispatches key events to the appropriate mode handler.
func (a *App) handleKey(key Key) bool {
	a.state.StatusMsg = ""
	switch a.state.Mode {
	case ModeNormal:
		return a.handleNormalKey(key)
	case ModeFilter:
		return a.handleFilterKey(key)
	case ModeConfirm:
		return a.handleConfirmKey(key)
	case ModeHelp:
		return a.handleHelpKey(key)
	case ModeForm:
		return a.handleFormKey(key)
	case ModeMigrate:
		return a.handleMigrateKey(key)
	case ModeCalendar:
		return a.handleCalendarKey(key)
	}
	return false
}

// render delegates to the Render function.
// It renders into a buffer then writes the whole frame at once to avoid flicker.
func (a *App) render() {
	var buf bytes.Buffer
	dateStr := a.date.Format("2006-01-02") + " " + a.date.Format("Mon")[:2]
	a.sizeMu.Lock()
	Render(&buf, a.entries, dateStr, a.state)
	a.sizeMu.Unlock()
	a.tty.Write(buf.Bytes())
}

// loadEntries loads entries for the current date from the service,
// updates the completer, and applies the active filter.
func (a *App) loadEntries() error {
	entries, err := a.svc.LoadDay(a.date)
	if err != nil {
		return err
	}
	a.allDay = entries
	a.completer.DiscoverFromEntries(entries)
	a.applyFilter()
	return nil
}

// applyFilter uses service.FilterEntries to rebuild the filtered entries slice
// and populates entryIndexMap to map filtered indices back to allDay indices.
func (a *App) applyFilter() {
	a.entries = nil
	a.entryIndexMap = nil

	for i, e := range a.allDay {
		if !matchesFilter(e, a.state) {
			continue
		}
		a.entries = append(a.entries, e)
		a.entryIndexMap = append(a.entryIndexMap, i)
	}

	a.clampCursor()
}

// matchesFilter returns true if the entry passes all active filters.
func matchesFilter(e model.Entry, vs *ViewState) bool {
	filtered := service.FilterEntries(
		[]model.Entry{e},
		vs.FilterProject, vs.FilterPerson, vs.FilterSymbol, vs.FilterText,
	)
	return len(filtered) > 0
}

// updateSize reads the current terminal dimensions with mutex protection.
func (a *App) updateSize() {
	rows, cols, err := term.GetSize(a.ttyFd)
	a.sizeMu.Lock()
	defer a.sizeMu.Unlock()
	if err == nil {
		a.state.Width = cols
		a.state.Height = rows
	} else {
		// Fallback defaults when size cannot be determined.
		if a.state.Width == 0 {
			a.state.Width = 80
		}
		if a.state.Height == 0 {
			a.state.Height = 24
		}
	}
}

// clampCursor ensures the cursor stays within valid bounds for the
// current filtered entry list.
func (a *App) clampCursor() {
	if a.state.Cursor >= len(a.entries) {
		a.state.Cursor = len(a.entries) - 1
	}
	if a.state.Cursor < 0 {
		a.state.Cursor = 0
	}
}
