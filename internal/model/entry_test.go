package model

import (
	"testing"
)

// helper builds a SymbolSet resembling a bullet journal:
//
//	task (bullet), event (*), note (-), done (x), migrated (>), scheduled (<), cancelled (~)
//
// Transitions: task -> done, migrated, scheduled, cancelled
func newTestSymbolSet() *SymbolSet {
	ss := NewSymbolSet()
	ss.Add("task", "•")
	ss.Add("event", "*")
	ss.Add("note", "-")
	ss.Add("done", "×")
	ss.Add("migrated", ">")
	ss.Add("scheduled", "<")
	ss.Add("cancelled", "~")

	ss.SetTransitions("task", []string{"done", "migrated", "scheduled", "cancelled"})
	return ss
}

func TestAdd(t *testing.T) {
	ss := NewSymbolSet()
	ss.Add("task", "•")
	ss.Add("event", "*")

	if len(ss.Symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(ss.Symbols))
	}
	if ss.Symbols["task"].Char != "•" {
		t.Errorf("expected task char '•', got %q", ss.Symbols["task"].Char)
	}
	if ss.CharToName["*"] != "event" {
		t.Errorf("expected char '*' to map to 'event', got %q", ss.CharToName["*"])
	}
	if len(ss.Order) != 2 || ss.Order[0] != "task" || ss.Order[1] != "event" {
		t.Errorf("unexpected order: %v", ss.Order)
	}
}

func TestLookupByChar(t *testing.T) {
	ss := newTestSymbolSet()

	tests := []struct {
		char    string
		wantOK  bool
		wantName string
	}{
		{"•", true, "task"},
		{"×", true, "done"},
		{"?", false, ""},
	}
	for _, tc := range tests {
		sym, ok := ss.LookupByChar(tc.char)
		if ok != tc.wantOK {
			t.Errorf("LookupByChar(%q): got ok=%v, want %v", tc.char, ok, tc.wantOK)
		}
		if ok && sym.Name != tc.wantName {
			t.Errorf("LookupByChar(%q): got name=%q, want %q", tc.char, sym.Name, tc.wantName)
		}
	}
}

func TestLookupByName(t *testing.T) {
	ss := newTestSymbolSet()

	tests := []struct {
		name    string
		wantOK  bool
		wantChar string
	}{
		{"task", true, "•"},
		{"migrated", true, ">"},
		{"bogus", false, ""},
	}
	for _, tc := range tests {
		sym, ok := ss.LookupByName(tc.name)
		if ok != tc.wantOK {
			t.Errorf("LookupByName(%q): got ok=%v, want %v", tc.name, ok, tc.wantOK)
		}
		if ok && sym.Char != tc.wantChar {
			t.Errorf("LookupByName(%q): got char=%q, want %q", tc.name, sym.Char, tc.wantChar)
		}
	}
}

func TestCanTransition(t *testing.T) {
	ss := newTestSymbolSet()

	tests := []struct {
		from, to string
		want     bool
	}{
		{"task", "done", true},
		{"task", "migrated", true},
		{"task", "scheduled", true},
		{"task", "cancelled", true},
		{"task", "event", false},      // not a valid target
		{"task", "task", false},       // self-transition not defined
		{"event", "done", false},      // event has no transitions
		{"note", "done", false},       // note has no transitions
		{"nonexistent", "done", false}, // unknown source
	}
	for _, tc := range tests {
		got := ss.CanTransition(tc.from, tc.to)
		if got != tc.want {
			t.Errorf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestValidTransitions(t *testing.T) {
	ss := newTestSymbolSet()

	syms := ss.ValidTransitions("task")
	names := make([]string, len(syms))
	for i, s := range syms {
		names[i] = s.Name
	}
	want := []string{"done", "migrated", "scheduled", "cancelled"}
	if len(names) != len(want) {
		t.Fatalf("ValidTransitions(task): got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("ValidTransitions(task)[%d] = %q, want %q", i, names[i], want[i])
		}
	}

	// Symbol with no transitions returns nil/empty.
	if got := ss.ValidTransitions("event"); len(got) != 0 {
		t.Errorf("ValidTransitions(event): expected empty, got %v", got)
	}
}

func TestIsState(t *testing.T) {
	ss := newTestSymbolSet()

	tests := []struct {
		name string
		want bool
	}{
		{"task", false},      // entry type, not a target
		{"event", false},     // entry type
		{"note", false},      // entry type
		{"done", true},       // transition target of task
		{"migrated", true},   // transition target of task
		{"scheduled", true},  // transition target of task
		{"cancelled", true},  // transition target of task
	}
	for _, tc := range tests {
		if got := ss.IsState(tc.name); got != tc.want {
			t.Errorf("IsState(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestSymbolNames(t *testing.T) {
	ss := newTestSymbolSet()

	got := ss.SymbolNames()
	want := []string{"task", "event", "note"}
	if len(got) != len(want) {
		t.Fatalf("SymbolNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SymbolNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStateNames(t *testing.T) {
	ss := newTestSymbolSet()

	got := ss.StateNames()
	want := []string{"done", "migrated", "scheduled", "cancelled"}
	if len(got) != len(want) {
		t.Fatalf("StateNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("StateNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNames(t *testing.T) {
	ss := newTestSymbolSet()

	got := ss.Names()
	want := []string{"task", "event", "note", "done", "migrated", "scheduled", "cancelled"}
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNewSymbolSetEmpty(t *testing.T) {
	ss := NewSymbolSet()
	if len(ss.Symbols) != 0 {
		t.Errorf("expected empty Symbols map")
	}
	if len(ss.CharToName) != 0 {
		t.Errorf("expected empty CharToName map")
	}
	if len(ss.Transitions) != 0 {
		t.Errorf("expected empty Transitions map")
	}
	if len(ss.Order) != 0 {
		t.Errorf("expected empty Order slice")
	}
}

func TestValidTransitionsSkipsUnknownTargets(t *testing.T) {
	ss := NewSymbolSet()
	ss.Add("task", "•")
	// Set a transition to a symbol that was never added.
	ss.SetTransitions("task", []string{"ghost"})

	got := ss.ValidTransitions("task")
	if len(got) != 0 {
		t.Errorf("expected no symbols for unknown target, got %v", got)
	}
}
