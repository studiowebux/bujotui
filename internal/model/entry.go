package model

import "time"

// Symbol represents a bullet journal symbol with a human-readable name and display character.
type Symbol struct {
	Name string // e.g. "task", "done", "cancelled"
	Char string // e.g. "•", "×", "✘"
}

// Entry is a single bullet journal log entry.
//
// State holds the lifecycle state of the entry. Valid values:
//   - "" (empty string) — active, no state applied
//   - "done" — completed
//   - "migrated" — moved to another date (see MigratedTo)
//   - "scheduled" — deferred to a future date
//   - "cancelled" — no longer relevant
type Entry struct {
	Symbol       Symbol // entry type: task, event, note, idea, urgent, waiting
	State        string // lifecycle state (see Entry doc)
	Project      string
	Person       string
	Description  string
	DateTime     time.Time
	MigratedTo   string // "YYYY-MM-DD" — where this entry was migrated to
	MigratedFrom string // "YYYY-MM-DD" — where this entry was migrated from
	ID           string // 16-char hex; set on first write, never changes — used for merge dedup
	UpdatedAt    int64  // unix timestamp; updated on every write — used for last-write-wins merge
}

// DayLog groups entries under a single date heading.
type DayLog struct {
	Date    time.Time // truncated to day
	Note    string    // free-text daily note (stored as "> note" in markdown)
	Entries []Entry
	// Raw holds non-entry lines (blanks, comments) for roundtrip preservation.
	// Each element is either a raw line string or nil (entry placeholder).
	Raw []RawLine
}

// RawLine represents either a parsed entry or a verbatim line.
type RawLine struct {
	IsEntry    bool
	EntryIndex int    // index into DayLog.Entries when IsEntry is true
	Text       string // original line text when IsEntry is false
}

// SymbolSet holds the full set of user-defined symbols and their transition rules.
type SymbolSet struct {
	Symbols     map[string]Symbol   // name -> Symbol
	CharToName  map[string]string   // char -> name (reverse lookup)
	Transitions map[string][]string // symbol name -> valid target symbol names
	Order       []string            // insertion-ordered symbol names for display
}

// NewSymbolSet creates an empty SymbolSet.
func NewSymbolSet() *SymbolSet {
	return &SymbolSet{
		Symbols:     make(map[string]Symbol),
		CharToName:  make(map[string]string),
		Transitions: make(map[string][]string),
	}
}

// Add registers a symbol.
func (ss *SymbolSet) Add(name, char string) {
	s := Symbol{Name: name, Char: char}
	ss.Symbols[name] = s
	ss.CharToName[char] = name
	ss.Order = append(ss.Order, name)
}

// SetTransitions sets the valid transition targets for a symbol.
func (ss *SymbolSet) SetTransitions(name string, targets []string) {
	ss.Transitions[name] = targets
}

// LookupByChar finds a symbol by its display character.
func (ss *SymbolSet) LookupByChar(char string) (Symbol, bool) {
	name, ok := ss.CharToName[char]
	if !ok {
		return Symbol{}, false
	}
	s, ok := ss.Symbols[name]
	return s, ok
}

// LookupByName finds a symbol by its name.
func (ss *SymbolSet) LookupByName(name string) (Symbol, bool) {
	s, ok := ss.Symbols[name]
	return s, ok
}

// CanTransition checks whether transitioning from one symbol to another is allowed.
func (ss *SymbolSet) CanTransition(from, to string) bool {
	targets, ok := ss.Transitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// ValidTransitions returns the list of symbols the given symbol can transition to.
func (ss *SymbolSet) ValidTransitions(name string) []Symbol {
	targets := ss.Transitions[name]
	var result []Symbol
	for _, t := range targets {
		if s, ok := ss.Symbols[t]; ok {
			result = append(result, s)
		}
	}
	return result
}

// Names returns all symbol names in definition order.
func (ss *SymbolSet) Names() []string {
	return ss.Order
}

// IsState returns true if a symbol is a lifecycle state — meaning it appears
// as a transition target of another symbol. Entry types like event/note that
// simply have no transitions are NOT states.
func (ss *SymbolSet) IsState(name string) bool {
	for src, targets := range ss.Transitions {
		if src == name {
			continue
		}
		for _, t := range targets {
			if t == name {
				return true
			}
		}
	}
	return false
}

// SymbolNames returns names that are entry types (not states) in definition order.
func (ss *SymbolSet) SymbolNames() []string {
	var result []string
	for _, name := range ss.Order {
		if !ss.IsState(name) {
			result = append(result, name)
		}
	}
	return result
}

// StateNames returns names that are states (transition targets only) in definition order.
func (ss *SymbolSet) StateNames() []string {
	var result []string
	for _, name := range ss.Order {
		if ss.IsState(name) {
			result = append(result, name)
		}
	}
	return result
}
