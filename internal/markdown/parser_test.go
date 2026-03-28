package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/studiowebux/bujotui/internal/model"
)

// testSymbols returns a SymbolSet with task, event, note, and done symbols.
func testSymbols() *model.SymbolSet {
	ss := model.NewSymbolSet()
	ss.Add("task", "•")
	ss.Add("event", "○")
	ss.Add("note", "–")
	ss.Add("done", "×")
	return ss
}

// localTime is a helper that builds a time.Time in the local timezone.
func localTime(year int, month time.Month, day, hour, min int) time.Time {
	return time.Date(year, month, day, hour, min, 0, 0, time.Local)
}

// localDate is a helper that builds a date-only time.Time in the local timezone.
func localDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

// ---------------------------------------------------------------------------
// ParseEntryLine
// ---------------------------------------------------------------------------

func TestParseEntryLine(t *testing.T) {
	symbols := testSymbols()

	tests := []struct {
		name    string
		line    string
		wantOK  bool
		wantEnt model.Entry
	}{
		{
			name:   "full entry with all fields",
			line:   "- • 2026-03-27T14:30 [myproject] @alice state:done ->2026-04-01 <-2026-03-20 implement parser",
			wantOK: true,
			wantEnt: model.Entry{
				Symbol:       model.Symbol{Name: "task", Char: "•"},
				DateTime:     localTime(2026, 3, 27, 14, 30),
				Project:      "myproject",
				Person:       "alice",
				State:        "done",
				MigratedTo:   "2026-04-01",
				MigratedFrom: "2026-03-20",
				Description:  "implement parser",
			},
		},
		{
			name:   "entry without project",
			line:   "- ○ 2026-03-27T09:00 @bob morning standup",
			wantOK: true,
			wantEnt: model.Entry{
				Symbol:      model.Symbol{Name: "event", Char: "○"},
				DateTime:    localTime(2026, 3, 27, 9, 0),
				Person:      "bob",
				Description: "morning standup",
			},
		},
		{
			name:   "entry without person",
			line:   "- – 2026-03-27T10:00 [notes] remember to check logs",
			wantOK: true,
			wantEnt: model.Entry{
				Symbol:      model.Symbol{Name: "note", Char: "–"},
				DateTime:    localTime(2026, 3, 27, 10, 0),
				Project:     "notes",
				Description: "remember to check logs",
			},
		},
		{
			name:   "entry with state only",
			line:   "- × 2026-03-27T11:00 [proj] @carol state:cancelled task was dropped",
			wantOK: true,
			wantEnt: model.Entry{
				Symbol:      model.Symbol{Name: "done", Char: "×"},
				DateTime:    localTime(2026, 3, 27, 11, 0),
				Project:     "proj",
				Person:      "carol",
				State:       "cancelled",
				Description: "task was dropped",
			},
		},
		{
			name:   "entry with migrated-to only",
			line:   "- • 2026-03-27T08:00 ->2026-04-01 deferred work",
			wantOK: true,
			wantEnt: model.Entry{
				Symbol:      model.Symbol{Name: "task", Char: "•"},
				DateTime:    localTime(2026, 3, 27, 8, 0),
				MigratedTo:  "2026-04-01",
				Description: "deferred work",
			},
		},
		{
			name:   "entry with migrated-from only",
			line:   "- • 2026-03-27T08:00 <-2026-03-20 carried over work",
			wantOK: true,
			wantEnt: model.Entry{
				Symbol:       model.Symbol{Name: "task", Char: "•"},
				DateTime:     localTime(2026, 3, 27, 8, 0),
				MigratedFrom: "2026-03-20",
				Description:  "carried over work",
			},
		},
		{
			name:   "minimal entry — description only",
			line:   "- • 2026-03-27T12:00 just a task",
			wantOK: true,
			wantEnt: model.Entry{
				Symbol:      model.Symbol{Name: "task", Char: "•"},
				DateTime:    localTime(2026, 3, 27, 12, 0),
				Description: "just a task",
			},
		},
		{
			name:   "person with no description",
			line:   "- ○ 2026-03-27T14:00 @dave",
			wantOK: true,
			wantEnt: model.Entry{
				Symbol:   model.Symbol{Name: "event", Char: "○"},
				DateTime: localTime(2026, 3, 27, 14, 0),
				Person:   "dave",
			},
		},
		{
			name:   "unknown symbol returns false",
			line:   "- ★ 2026-03-27T12:00 unknown symbol",
			wantOK: false,
		},
		{
			name:   "missing dash prefix returns false",
			line:   "• 2026-03-27T12:00 no prefix",
			wantOK: false,
		},
		{
			name:   "empty line returns false",
			line:   "",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseEntryLine(tc.line, symbols)
			if ok != tc.wantOK {
				t.Fatalf("ParseEntryLine ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			assertEntryEqual(t, tc.wantEnt, got)
		})
	}
}

// ---------------------------------------------------------------------------
// ParseFile
// ---------------------------------------------------------------------------

func TestParseFile(t *testing.T) {
	symbols := testSymbols()

	tests := []struct {
		name     string
		input    string
		wantDays int
		validate func(t *testing.T, days []model.DayLog)
	}{
		{
			name:     "empty file",
			input:    "",
			wantDays: 0,
		},
		{
			name:     "single day with one entry",
			input:    "# 2026-03-27\n- • 2026-03-27T10:00 [proj] @alice hello world\n",
			wantDays: 1,
			validate: func(t *testing.T, days []model.DayLog) {
				if !days[0].Date.Equal(localDate(2026, 3, 27)) {
					t.Errorf("date = %v, want 2026-03-27", days[0].Date)
				}
				if len(days[0].Entries) != 1 {
					t.Fatalf("entries = %d, want 1", len(days[0].Entries))
				}
				if days[0].Entries[0].Project != "proj" {
					t.Errorf("project = %q, want %q", days[0].Entries[0].Project, "proj")
				}
			},
		},
		{
			name: "single day with note and entries",
			input: "# 2026-03-27\n> daily note here\n- • 2026-03-27T10:00 task one\n- ○ 2026-03-27T11:00 event one\n",
			wantDays: 1,
			validate: func(t *testing.T, days []model.DayLog) {
				if days[0].Note != "daily note here" {
					t.Errorf("note = %q, want %q", days[0].Note, "daily note here")
				}
				if len(days[0].Entries) != 2 {
					t.Fatalf("entries = %d, want 2", len(days[0].Entries))
				}
			},
		},
		{
			name: "multiple days",
			input: "# 2026-03-27\n- • 2026-03-27T10:00 task A\n# 2026-03-28\n- ○ 2026-03-28T09:00 event B\n",
			wantDays: 2,
			validate: func(t *testing.T, days []model.DayLog) {
				if !days[0].Date.Equal(localDate(2026, 3, 27)) {
					t.Errorf("day[0] date = %v, want 2026-03-27", days[0].Date)
				}
				if !days[1].Date.Equal(localDate(2026, 3, 28)) {
					t.Errorf("day[1] date = %v, want 2026-03-28", days[1].Date)
				}
				if days[0].Entries[0].Description != "task A" {
					t.Errorf("day[0] entry desc = %q", days[0].Entries[0].Description)
				}
				if days[1].Entries[0].Description != "event B" {
					t.Errorf("day[1] entry desc = %q", days[1].Entries[0].Description)
				}
			},
		},
		{
			name:     "day heading with no entries",
			input:    "# 2026-03-27\n",
			wantDays: 1,
			validate: func(t *testing.T, days []model.DayLog) {
				if len(days[0].Entries) != 0 {
					t.Errorf("entries = %d, want 0", len(days[0].Entries))
				}
			},
		},
		{
			name: "raw lines preserved",
			input: "# 2026-03-27\n\nsome raw text\n- • 2026-03-27T10:00 task\n",
			wantDays: 1,
			validate: func(t *testing.T, days []model.DayLog) {
				if len(days[0].Raw) != 3 {
					t.Fatalf("raw = %d, want 3 (blank + raw text + entry)", len(days[0].Raw))
				}
				if days[0].Raw[0].IsEntry {
					t.Error("raw[0] should not be entry")
				}
				if days[0].Raw[2].IsEntry != true || days[0].Raw[2].EntryIndex != 0 {
					t.Error("raw[2] should be entry with index 0")
				}
			},
		},
		{
			name: "lines before first heading are ignored",
			input: "some preamble\n# 2026-03-27\n- – 2026-03-27T10:00 note\n",
			wantDays: 1,
			validate: func(t *testing.T, days []model.DayLog) {
				if len(days[0].Entries) != 1 {
					t.Fatalf("entries = %d, want 1", len(days[0].Entries))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			days, err := ParseFile(strings.NewReader(tc.input), symbols)
			if err != nil {
				t.Fatalf("ParseFile error: %v", err)
			}
			if len(days) != tc.wantDays {
				t.Fatalf("days = %d, want %d", len(days), tc.wantDays)
			}
			if tc.validate != nil {
				tc.validate(t, days)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FormatEntry roundtrip
// ---------------------------------------------------------------------------

func TestFormatEntry_Roundtrip(t *testing.T) {
	symbols := testSymbols()

	tests := []struct {
		name  string
		entry model.Entry
	}{
		{
			name: "full entry",
			entry: model.Entry{
				Symbol:       model.Symbol{Name: "task", Char: "•"},
				DateTime:     localTime(2026, 3, 27, 14, 30),
				Project:      "myproject",
				Person:       "alice",
				State:        "done",
				MigratedTo:   "2026-04-01",
				MigratedFrom: "2026-03-20",
				Description:  "implement parser",
			},
		},
		{
			name: "no project no person",
			entry: model.Entry{
				Symbol:      model.Symbol{Name: "event", Char: "○"},
				DateTime:    localTime(2026, 3, 27, 9, 0),
				Description: "standup meeting",
			},
		},
		{
			name: "with migration links only",
			entry: model.Entry{
				Symbol:      model.Symbol{Name: "task", Char: "•"},
				DateTime:    localTime(2026, 3, 27, 8, 0),
				MigratedTo:  "2026-04-01",
				Description: "deferred",
			},
		},
		{
			name: "state without migrations",
			entry: model.Entry{
				Symbol:      model.Symbol{Name: "done", Char: "×"},
				DateTime:    localTime(2026, 3, 27, 17, 0),
				Project:     "proj",
				Person:      "bob",
				State:       "cancelled",
				Description: "dropped",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			line := FormatEntry(tc.entry)
			got, ok := ParseEntryLine(line, symbols)
			if !ok {
				t.Fatalf("ParseEntryLine failed on formatted line: %q", line)
			}
			assertEntryEqual(t, tc.entry, got)
		})
	}
}

// ---------------------------------------------------------------------------
// FormatFile roundtrip
// ---------------------------------------------------------------------------

func TestFormatFile_Roundtrip(t *testing.T) {
	symbols := testSymbols()

	tests := []struct {
		name  string
		input string
	}{
		{
			name: "single day with note and entries",
			input: "# 2026-03-27\n> daily reflection\n- • 2026-03-27T10:00 [proj] @alice first task\n- ○ 2026-03-27T11:00 @bob team event\n",
		},
		{
			name: "multiple days",
			input: "# 2026-03-27\n- • 2026-03-27T10:00 task A\n\n# 2026-03-28\n- ○ 2026-03-28T09:00 event B\n",
		},
		{
			name:  "single day no entries",
			input: "# 2026-03-27\n",
		},
		{
			name: "entry with all optional fields",
			input: "# 2026-03-27\n- • 2026-03-27T14:30 [myproject] @alice state:done ->2026-04-01 <-2026-03-20 implement parser\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// First parse
			days1, err := ParseFile(strings.NewReader(tc.input), symbols)
			if err != nil {
				t.Fatalf("first ParseFile: %v", err)
			}

			// Format
			output := FormatFile(days1)

			// Second parse
			days2, err := ParseFile(strings.NewReader(string(output)), symbols)
			if err != nil {
				t.Fatalf("second ParseFile: %v", err)
			}

			// Compare day counts
			if len(days1) != len(days2) {
				t.Fatalf("day count mismatch: %d vs %d", len(days1), len(days2))
			}

			for i := range days1 {
				d1, d2 := days1[i], days2[i]
				if !d1.Date.Equal(d2.Date) {
					t.Errorf("day[%d] date: %v vs %v", i, d1.Date, d2.Date)
				}
				if d1.Note != d2.Note {
					t.Errorf("day[%d] note: %q vs %q", i, d1.Note, d2.Note)
				}
				if len(d1.Entries) != len(d2.Entries) {
					t.Fatalf("day[%d] entry count: %d vs %d", i, len(d1.Entries), len(d2.Entries))
				}
				for j := range d1.Entries {
					assertEntryEqual(t, d1.Entries[j], d2.Entries[j])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertEntryEqual(t *testing.T, want, got model.Entry) {
	t.Helper()
	if got.Symbol != want.Symbol {
		t.Errorf("Symbol = %+v, want %+v", got.Symbol, want.Symbol)
	}
	if !got.DateTime.Equal(want.DateTime) {
		t.Errorf("DateTime = %v, want %v", got.DateTime, want.DateTime)
	}
	if got.Project != want.Project {
		t.Errorf("Project = %q, want %q", got.Project, want.Project)
	}
	if got.Person != want.Person {
		t.Errorf("Person = %q, want %q", got.Person, want.Person)
	}
	if got.State != want.State {
		t.Errorf("State = %q, want %q", got.State, want.State)
	}
	if got.MigratedTo != want.MigratedTo {
		t.Errorf("MigratedTo = %q, want %q", got.MigratedTo, want.MigratedTo)
	}
	if got.MigratedFrom != want.MigratedFrom {
		t.Errorf("MigratedFrom = %q, want %q", got.MigratedFrom, want.MigratedFrom)
	}
	if got.Description != want.Description {
		t.Errorf("Description = %q, want %q", got.Description, want.Description)
	}
}
