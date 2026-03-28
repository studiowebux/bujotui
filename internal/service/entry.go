package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/storage"
)

// EntryService encapsulates business logic for journal entries.
// It depends only on storage and config — never on the TUI layer.
type EntryService struct {
	store *storage.Store
	cfg   *config.Config
}

// NewEntryService creates an EntryService with the given dependencies.
func NewEntryService(store *storage.Store, cfg *config.Config) *EntryService {
	return &EntryService{store: store, cfg: cfg}
}

// AddEntry validates the symbol, builds a new entry timestamped with time.Now(),
// persists it via the store, and returns the created entry.
func (s *EntryService) AddEntry(symbol, state, project, person, description string) (model.Entry, error) {
	if description == "" {
		return model.Entry{}, fmt.Errorf("description must not be empty")
	}

	symName := symbol
	if symName == "" {
		symName = s.cfg.Defaults.Symbol
	}
	if person == "" {
		person = s.cfg.Defaults.Person
	}

	sym, ok := s.cfg.Symbols.LookupByName(symName)
	if !ok {
		return model.Entry{}, fmt.Errorf("unknown symbol %q: not defined in configuration", symName)
	}

	entry := model.Entry{
		Symbol:      sym,
		State:       state,
		Project:     project,
		Person:      person,
		Description: description,
		DateTime:    time.Now(),
	}

	if err := s.store.AddEntry(entry); err != nil {
		return model.Entry{}, fmt.Errorf("add entry: %w", err)
	}

	return entry, nil
}

// EditEntry validates inputs and updates the entry at the given index for the
// specified date. The entry keeps its original timestamp.
func (s *EntryService) EditEntry(date time.Time, index int, symbol, state, project, person, description string) error {
	if description == "" {
		return fmt.Errorf("description must not be empty")
	}

	symName := symbol
	if symName == "" {
		symName = s.cfg.Defaults.Symbol
	}
	if person == "" {
		person = s.cfg.Defaults.Person
	}

	sym, ok := s.cfg.Symbols.LookupByName(symName)
	if !ok {
		return fmt.Errorf("unknown symbol %q: not defined in configuration", symName)
	}

	// Load the existing entry so we can preserve its timestamp.
	entries, err := s.store.LoadDay(date)
	if err != nil {
		return fmt.Errorf("load day: %w", err)
	}
	if index < 0 || index >= len(entries) {
		return fmt.Errorf("entry index %d out of range (0-%d)", index, len(entries)-1)
	}

	updated := model.Entry{
		Symbol:      sym,
		State:       state,
		Project:     project,
		Person:      person,
		Description: description,
		DateTime:    entries[index].DateTime,
	}

	if err := s.store.UpdateEntry(date, index, updated); err != nil {
		return fmt.Errorf("update entry: %w", err)
	}

	return nil
}

// DeleteEntry removes the entry at the given index for the specified date.
func (s *EntryService) DeleteEntry(date time.Time, index int) error {
	if err := s.store.RemoveEntry(date, index); err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	return nil
}

// TransitionEntry validates that the transition from the entry's current symbol
// to targetState is allowed, resolves the target symbol, and persists the change.
func (s *EntryService) TransitionEntry(date time.Time, index int, targetState string) error {
	entries, err := s.store.LoadDay(date)
	if err != nil {
		return fmt.Errorf("load day: %w", err)
	}
	if index < 0 || index >= len(entries) {
		return fmt.Errorf("entry index %d out of range (0-%d)", index, len(entries)-1)
	}

	entry := entries[index]

	if !s.cfg.Symbols.CanTransition(entry.Symbol.Name, targetState) {
		return fmt.Errorf(
			"cannot transition from %q to %q: transition not allowed",
			entry.Symbol.Name, targetState,
		)
	}

	// Verify the target state is a known symbol/state name
	if _, ok := s.cfg.Symbols.LookupByName(targetState); !ok {
		return fmt.Errorf("unknown target state %q: not defined in configuration", targetState)
	}

	entry.State = targetState

	if err := s.store.UpdateEntry(date, index, entry); err != nil {
		return fmt.Errorf("transition entry: %w", err)
	}

	return nil
}

// ResetState clears the lifecycle state of an entry, returning it to active.
func (s *EntryService) ResetState(date time.Time, index int) error {
	entries, err := s.store.LoadDay(date)
	if err != nil {
		return fmt.Errorf("load day: %w", err)
	}
	if index < 0 || index >= len(entries) {
		return fmt.Errorf("entry index %d out of range (0-%d)", index, len(entries)-1)
	}

	entry := entries[index]
	entry.State = ""
	entry.MigratedTo = ""
	entry.MigratedFrom = ""

	if err := s.store.UpdateEntry(date, index, entry); err != nil {
		return fmt.Errorf("reset state: %w", err)
	}
	return nil
}

// MigrateEntry copies the entry at index on sourceDate to targetDate,
// then marks the original as migrated.
func (s *EntryService) MigrateEntry(sourceDate time.Time, index int, targetDate time.Time) error {
	entries, err := s.store.LoadDay(sourceDate)
	if err != nil {
		return fmt.Errorf("load source day: %w", err)
	}
	if index < 0 || index >= len(entries) {
		return fmt.Errorf("entry index %d out of range (0-%d)", index, len(entries)-1)
	}

	entry := entries[index]

	if entry.State == "migrated" {
		return fmt.Errorf("entry is already migrated")
	}

	src := time.Date(sourceDate.Year(), sourceDate.Month(), sourceDate.Day(), 0, 0, 0, 0, sourceDate.Location())
	tgt := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	if src.Equal(tgt) {
		return fmt.Errorf("cannot migrate to the same day")
	}

	if !s.cfg.Symbols.CanTransition(entry.Symbol.Name, "migrated") {
		return fmt.Errorf("cannot migrate %q entries", entry.Symbol.Name)
	}

	srcDateStr := sourceDate.Format("2006-01-02")
	tgtDateStr := targetDate.Format("2006-01-02")

	// Create copy on target date with back-link
	entryCopy := model.Entry{
		Symbol:       entry.Symbol,
		State:        "",
		Project:      entry.Project,
		Person:       entry.Person,
		Description:  entry.Description,
		DateTime:     targetDate,
		MigratedFrom: srcDateStr,
	}
	if err := s.store.AddEntry(entryCopy); err != nil {
		return fmt.Errorf("add migrated copy: %w", err)
	}

	// Mark original as migrated with forward-link
	entry.State = "migrated"
	entry.MigratedTo = tgtDateStr
	if err := s.store.UpdateEntry(sourceDate, index, entry); err != nil {
		return fmt.Errorf("mark original as migrated: %w", err)
	}

	return nil
}

// SaveNote sets the daily note for a given date.
func (s *EntryService) SaveNote(date time.Time, note string) error {
	days, err := s.store.LoadMonth(date)
	if err != nil {
		return fmt.Errorf("load month: %w", err)
	}

	dateStr := date.Format("2006-01-02")
	found := false
	for i, d := range days {
		if d.Date.Format("2006-01-02") == dateStr {
			days[i].Note = note
			// Update raw lines: find and replace existing note line, or insert one
			updated := false
			for j, rl := range days[i].Raw {
				if !rl.IsEntry && len(rl.Text) > 1 && strings.HasPrefix(strings.TrimSpace(rl.Text), "> ") {
					if note != "" {
						days[i].Raw[j].Text = "> " + note
					} else {
						// Remove the note line
						days[i].Raw = append(days[i].Raw[:j], days[i].Raw[j+1:]...)
					}
					updated = true
					break
				}
			}
			if !updated && note != "" {
				// Insert note line at the beginning of raw lines
				noteLine := model.RawLine{IsEntry: false, Text: "> " + note}
				days[i].Raw = append([]model.RawLine{
					{IsEntry: false, Text: ""},
					noteLine,
				}, days[i].Raw...)
			}
			found = true
			break
		}
	}

	if !found && note != "" {
		// Create a new day with just the note
		days = append(days, model.DayLog{
			Date: date,
			Note: note,
			Raw: []model.RawLine{
				{IsEntry: false, Text: ""},
				{IsEntry: false, Text: "> " + note},
			},
		})
	}

	if err := s.store.SaveMonth(date, days); err != nil {
		return fmt.Errorf("save note: %w", err)
	}
	return nil
}

// LoadMonthNotes returns a map of day number -> note for the given month.
func (s *EntryService) LoadMonthNotes(month time.Time) (map[int]string, error) {
	days, err := s.store.LoadMonth(month)
	if err != nil {
		return nil, fmt.Errorf("load month: %w", err)
	}
	result := make(map[int]string)
	for _, d := range days {
		if d.Note != "" {
			result[d.Date.Day()] = d.Note
		}
	}
	return result, nil
}

// LoadMonth returns a map of day number -> entries for the given month.
func (s *EntryService) LoadMonth(month time.Time) (map[int][]model.Entry, error) {
	days, err := s.store.LoadMonth(month)
	if err != nil {
		return nil, fmt.Errorf("load month: %w", err)
	}
	result := make(map[int][]model.Entry)
	for _, d := range days {
		result[d.Date.Day()] = d.Entries
	}
	return result, nil
}

// LoadDay returns all entries for the given date, delegating to the store.
func (s *EntryService) LoadDay(date time.Time) ([]model.Entry, error) {
	entries, err := s.store.LoadDay(date)
	if err != nil {
		return nil, fmt.Errorf("load day: %w", err)
	}
	return entries, nil
}

// FilterEntries applies in-memory filtering to a slice of entries.
// All filter parameters are optional — an empty string means "no filter" for that field.
// Matching is case-insensitive. Text search matches against the description.
func FilterEntries(entries []model.Entry, project, person, symbol, text string) []model.Entry {
	if project == "" && person == "" && symbol == "" && text == "" {
		// No filters active — return a copy to avoid aliasing.
		result := make([]model.Entry, len(entries))
		copy(result, entries)
		return result
	}

	lowerText := strings.ToLower(text)
	var result []model.Entry

	for _, e := range entries {
		if project != "" && !strings.EqualFold(e.Project, project) {
			continue
		}
		if person != "" && !strings.EqualFold(e.Person, person) {
			continue
		}
		if symbol != "" && !strings.EqualFold(e.Symbol.Name, symbol) {
			continue
		}
		if text != "" {
			haystack := strings.ToLower(e.Description + " " + e.Project + " " + e.Person + " " + e.Symbol.Name + " " + e.State)
			if !strings.Contains(haystack, lowerText) {
				continue
			}
		}
		result = append(result, e)
	}

	return result
}
