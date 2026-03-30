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
	svc    *service.EntryService
	colSvc *service.CollectionService
	habSvc *service.HabitService
	futSvc *service.FutureLogService
	cfg    *config.Config
	completer *complete.Completer

	state         *ViewState
	date          time.Time
	entries       []model.Entry // filtered view of current day's entries
	allDay        []model.Entry // unfiltered entries for the current day
	entryIndexMap []int         // maps filtered index -> allDay index

	tty   *os.File
	ttyFd uintptr

	sizeMu sync.Mutex // protects sizeW/sizeH written by SIGWINCH goroutine
	sizeW  int
	sizeH  int
}

// New creates a new TUI App.
func New(svc *service.EntryService, colSvc *service.CollectionService, habSvc *service.HabitService, futSvc *service.FutureLogService, cfg *config.Config) *App {
	comp := complete.New(
		cfg.Symbols.SymbolNames(),
		cfg.Projects,
		cfg.People,
	)

	return &App{
		svc:    svc,
		colSvc: colSvc,
		habSvc: habSvc,
		futSvc: futSvc,
		cfg:    cfg,
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
			// SIGWINCH interrupts read with EINTR — re-render with new size
			if err == syscall.EINTR {
				a.render()
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
	case ModeCollections:
		return a.handleCollectionsKey(key)
	case ModeCollection:
		return a.handleCollectionKey(key)
	case ModeIndex:
		return a.handleIndexKey(key)
	case ModeHabit:
		return a.handleHabitKey(key)
	case ModeFuture:
		return a.handleFutureKey(key)
	}
	return false
}

// render delegates to the Render function.
// It renders into a buffer then writes the whole frame at once to avoid flicker.
func (a *App) render() {
	a.syncSize()
	var buf bytes.Buffer
	dateStr := a.date.Format("2006-01-02") + " " + a.date.Format("Mon")[:2]
	Render(&buf, a.entries, dateStr, a.state)
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

// applyFilter rebuilds the filtered entries slice from allDay.
func (a *App) applyFilter() {
	fp := a.state.FilterProject
	fn := a.state.FilterPerson
	fs := a.state.FilterSymbol
	ft := a.state.FilterText

	filtered := service.FilterEntries(a.allDay, fp, fn, fs, ft)

	// Build index map from filtered results back to allDay positions.
	a.entries = filtered
	a.entryIndexMap = nil
	if fp != "" || fn != "" || fs != "" || ft != "" {
		fi := 0
		for i := range a.allDay {
			if fi < len(filtered) && a.allDay[i] == filtered[fi] {
				a.entryIndexMap = append(a.entryIndexMap, i)
				fi++
			}
		}
	} else {
		a.entryIndexMap = make([]int, len(a.allDay))
		for i := range a.allDay {
			a.entryIndexMap[i] = i
		}
	}

	a.clampCursor()
}

// updateSize reads the current terminal dimensions with mutex protection.
// Called from the SIGWINCH goroutine — writes to sizeW/sizeH only.
func (a *App) updateSize() {
	rows, cols, err := term.GetSize(a.ttyFd)
	a.sizeMu.Lock()
	defer a.sizeMu.Unlock()
	if err == nil {
		a.sizeW = cols
		a.sizeH = rows
	} else if a.sizeW == 0 {
		a.sizeW = 80
		a.sizeH = 24
	}
}

// syncSize copies terminal dimensions from the goroutine-safe fields
// into ViewState. Called only from the main loop.
func (a *App) syncSize() {
	a.sizeMu.Lock()
	a.state.Width = a.sizeW
	a.state.Height = a.sizeH
	a.sizeMu.Unlock()
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
