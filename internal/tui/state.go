package tui

import (
	"time"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/model"
)

// Mode represents the current TUI interaction mode.
type Mode int

const (
	ModeNormal  Mode = iota
	ModeFilter       // typing a filter
	ModeConfirm      // confirming a delete
	ModeHelp         // showing help screen
	ModeForm         // multi-field form for add/edit
	ModeMigrate      // picking a target date for migration
	ModeCalendar     // monthly calendar view
	ModeCollections  // list of collections
	ModeCollection   // viewing/editing a single collection
	ModeIndex        // searchable index of collections/projects
	ModeHabit        // habit tracker grid
	ModeFuture       // future log view
)

// ViewState holds all view-layer state. This never leaks into model or data.
type ViewState struct {
	Cursor   int // selected entry index
	Mode     Mode
	ShowTime bool // toggle: show time column

	// Input mode state
	Input       EditBuffer
	InputPrompt string // e.g. "filter> "

	// Filter state
	FilterProject string
	FilterPerson  string
	FilterSymbol  string
	FilterText    string // free-text filter on description

	// Autocomplete state
	Completions    []string
	CompletionIdx  int
	CompletionType string // "symbol", "project", "person"

	// Form state (add/edit modal)
	Form *Form

	// Confirm state
	ConfirmMsg   string
	ConfirmIndex int // entry index to act on

	// Migrate state
	MigrateIndex int        // real allDay index of entry to migrate
	MigrateDate  EditBuffer // target date input

	// Calendar state
	CalMonth    time.Time             // first day of displayed month
	CalCursor   int                   // selected day (0-based, 0 = day 1)
	CalEntries  map[int][]model.Entry // day number -> entries for that day
	CalNotes    map[int]string        // day number -> daily note
	CalEditing  bool                  // true when editing the note
	CalNoteBuf  EditBuffer            // note edit buffer

	// Collections state
	ColNames      []string // list of collection names
	ColCursor     int      // cursor in collection list
	ColScroll     int      // scroll offset for collection list
	ColName       string   // name of currently viewed collection
	ColItems      []ColViewItem // items in current collection
	ColItemCursor int      // cursor in item list
	ColItemScroll int      // scroll offset for item list
	ColEditing    bool     // true when adding/editing an item
	ColEditBuf    EditBuffer
	ColEditIdx    int  // -1 for add, >=0 for edit
	ColAdding     bool // true when creating a new collection
	ColConfirm    bool // true when confirming a delete

	// Habit state
	HabMonth    time.Time           // first day of displayed month
	HabTracker  *HabitViewData      // loaded habit data
	HabRow      int                 // cursor row (habit index)
	HabCol      int                 // cursor col (day, 0-based = day 1)
	HabAdding   bool                // adding a new habit
	HabEditing  bool                // editing a habit name
	HabEditBuf  EditBuffer
	HabConfirm  bool                // confirming a delete

	// Future log state
	FutYear     int                // displayed year
	FutMonths   []FutureViewMonth  // loaded months (6 months from current)
	FutMonthIdx int                // selected month index
	FutItemIdx  int                // selected entry within month
	FutAdding   bool               // adding a new entry
	FutEditBuf  EditBuffer
	FutConfirm  bool               // confirming a delete

	// Index state
	IdxEntries    []IndexEntry // all index entries
	IdxFiltered   []int        // indices into IdxEntries matching filter
	IdxCursor     int
	IdxScroll     int
	IdxFilterBuf  EditBuffer
	IdxFiltering  bool // true when typing in filter

	// Status message (shown for one frame after an action)
	StatusMsg string

	// Scroll
	ScrollOffset int

	// Terminal size
	Width  int
	Height int

	// Config references (for rendering)
	Symbols *model.SymbolSet
	Cfg     *config.Config
}

// NewViewState returns a ViewState with sensible defaults.
func NewViewState(cfg *config.Config) *ViewState {
	return &ViewState{
		CompletionIdx: -1,
		Symbols:       cfg.Symbols,
		Cfg:           cfg,
	}
}

// InputString returns the current input buffer as a string.
func (vs *ViewState) InputString() string {
	return vs.Input.String()
}

// ClearInput resets the input buffer.
func (vs *ViewState) ClearInput() {
	vs.Input.Clear()
	vs.ClearCompletions()
}

// ClearCompletions resets autocomplete state.
func (vs *ViewState) ClearCompletions() {
	vs.Completions = nil
	vs.CompletionIdx = -1
	vs.CompletionType = ""
}

// FormField represents a single field in the add/edit form.
type FormField struct {
	Label string // display label
	Buf   EditBuffer
	Type  string // "symbol", "project", "person", "text"
}

// Form holds the state for the multi-field add/edit form.
type Form struct {
	Fields  []FormField
	Active  int  // index of focused field
	IsEdit  bool // true = editing existing entry
	EditIdx int  // index of entry being edited (-1 for new)
}

// ActiveField returns the currently focused field.
func (f *Form) ActiveField() *FormField {
	if f.Active >= 0 && f.Active < len(f.Fields) {
		return &f.Fields[f.Active]
	}
	return nil
}

// NextField moves focus to the next field, wrapping around.
func (f *Form) NextField() {
	f.Active = (f.Active + 1) % len(f.Fields)
}

// PrevField moves focus to the previous field, wrapping around.
func (f *Form) PrevField() {
	f.Active = (f.Active - 1 + len(f.Fields)) % len(f.Fields)
}

// FieldInsertChar inserts a character in the active field.
func (f *Form) FieldInsertChar(c byte) {
	if field := f.ActiveField(); field != nil {
		field.Buf.InsertChar(c)
	}
}

// FieldDeleteChar removes the character before the cursor in the active field.
func (f *Form) FieldDeleteChar() {
	if field := f.ActiveField(); field != nil {
		field.Buf.DeleteChar()
	}
}

// FieldDeleteCharForward removes the character at the cursor in the active field.
func (f *Form) FieldDeleteCharForward() {
	if field := f.ActiveField(); field != nil {
		field.Buf.DeleteCharForward()
	}
}

// FieldWordLeft moves cursor to the start of the previous word.
func (f *Form) FieldWordLeft() {
	if field := f.ActiveField(); field != nil {
		field.Buf.WordLeft()
	}
}

// FieldWordRight moves cursor to the end of the next word.
func (f *Form) FieldWordRight() {
	if field := f.ActiveField(); field != nil {
		field.Buf.WordRight()
	}
}

// FieldDeleteWord removes the word before the cursor.
func (f *Form) FieldDeleteWord() {
	if field := f.ActiveField(); field != nil {
		field.Buf.DeleteWord()
	}
}

// FieldValue returns the string value of a field by type.
func (f *Form) FieldValue(fieldType string) string {
	for _, field := range f.Fields {
		if field.Type == fieldType {
			return field.Buf.String()
		}
	}
	return ""
}

// ColViewItem wraps a collection item for display.
type ColViewItem struct {
	Text string
	Done bool
}

// IndexEntry represents one item in the index view.
type IndexEntry struct {
	Kind string // "collection" or "project"
	Name string
}

// FutureViewMonth holds one month's data for display.
type FutureViewMonth struct {
	Year    int
	Month   int    // 1-12
	Label   string // "January 2026"
	Entries []FutureViewEntry
}

// FutureViewEntry is a single future log item for display.
type FutureViewEntry struct {
	Symbol string
	Desc   string
}

// HabitViewData holds habit data for TUI display.
type HabitViewData struct {
	Habits  []string              // habit names
	Done    map[string]map[int]bool // habit -> day -> done
	NumDays int                   // days in month
	Streaks map[string]int        // habit -> current streak
}

// AcceptCompletion replaces the current token with the selected completion.
func (vs *ViewState) AcceptCompletion() {
	if vs.CompletionIdx < 0 || vs.CompletionIdx >= len(vs.Completions) {
		return
	}
	completion := vs.Completions[vs.CompletionIdx]

	// Find the start of the current token
	tokenStart := vs.Input.Cursor
	for tokenStart > 0 && vs.Input.Data[tokenStart-1] != ' ' {
		tokenStart--
	}

	// Handle @ prefix for people
	prefix := ""
	if tokenStart < len(vs.Input.Data) && vs.Input.Data[tokenStart] == '@' {
		prefix = "@"
		tokenStart++
	}

	// Replace token
	after := make([]byte, len(vs.Input.Data[vs.Input.Cursor:]))
	copy(after, vs.Input.Data[vs.Input.Cursor:])

	vs.Input.Data = append(vs.Input.Data[:tokenStart], []byte(prefix+completion)...)
	vs.Input.Cursor = len(vs.Input.Data)
	vs.Input.Data = append(vs.Input.Data, after...)

	vs.ClearCompletions()
}
