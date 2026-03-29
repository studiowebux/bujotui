package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/studiowebux/bujotui/internal/config"
	bujocrypto "github.com/studiowebux/bujotui/internal/crypto"
	"github.com/studiowebux/bujotui/internal/markdown"
	"github.com/studiowebux/bujotui/internal/model"
)

// Store handles file-level persistence for bullet journal entries.
type Store struct {
	Dir    string
	Config *config.Config

	// Vault provides optional encryption at rest. If nil, files are stored as plaintext.
	Vault    *bujocrypto.Vault
	KeySlots []*bujocrypto.KeySlot // key slots for encrypted file headers
}

// NewStore creates a Store and ensures required directories exist.
func NewStore(cfg *config.Config) (*Store, error) {
	dailyDir := filepath.Join(cfg.DataDir, "daily")
	if err := os.MkdirAll(dailyDir, 0o700); err != nil {
		return nil, fmt.Errorf("create daily dir: %w", err)
	}
	return &Store{Dir: cfg.DataDir, Config: cfg}, nil
}

// MonthFile returns the path to the monthly markdown file for the given time.
func (s *Store) MonthFile(t time.Time) string {
	return filepath.Join(s.Dir, "daily", t.Format("2006-01")+".md")
}

// readFile reads a file, decrypting if the vault is set and the file is encrypted.
// Returns plaintext bytes.
func (s *Store) readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- path from user-configured data dir
	if err != nil {
		return nil, err
	}

	if s.Vault != nil && bujocrypto.IsEncrypted(data) {
		_, _, slots, ciphertext, err := bujocrypto.ParseFileRaw(data)
		if err != nil {
			return nil, fmt.Errorf("parse encrypted file: %w", err)
		}
		_ = slots
		plaintext, err := s.Vault.Decrypt(ciphertext)
		if err != nil {
			return nil, fmt.Errorf("decrypt file: %w", err)
		}
		return plaintext, nil
	}

	return data, nil
}

// writeFile versions the existing file then atomically writes new data.
// Encrypts if the vault is set.
func (s *Store) writeFile(path string, data []byte) error {
	if err := saveVersion(path); err != nil {
		return fmt.Errorf("save version: %w", err)
	}

	if s.Vault != nil {
		encrypted, err := bujocrypto.EncryptFile(s.Vault, s.KeySlots, data)
		if err != nil {
			return fmt.Errorf("encrypt file: %w", err)
		}
		data = encrypted
	}
	return AtomicWriteFile(path, data, 0o644)
}

// LoadMonth reads and parses the monthly file.
// Returns (nil, nil) if the file does not exist — callers should treat nil as empty.
func (s *Store) LoadMonth(t time.Time) ([]model.DayLog, error) {
	path := s.MonthFile(t)
	data, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("load month file: %w", err)
	}
	return markdown.ParseBytes(data, s.Config.Symbols)
}

// SaveMonth writes DayLogs to the monthly file atomically.
// It re-reads the current disk state and merges before writing so that
// concurrent writers (multiple MCP agents, the TUI) do not clobber each other.
func (s *Store) SaveMonth(t time.Time, days []model.DayLog) error {
	return s.saveMonth(t, days, nil)
}

// saveMonth is the internal implementation; deletedIDs prevents explicitly
// removed entries from being resurrected by the merge.
func (s *Store) saveMonth(t time.Time, days []model.DayLog, deletedIDs map[string]struct{}) error {
	path := s.MonthFile(t)
	// LoadMonth already returns (nil, nil) for missing files, so any
	// error here is a real I/O failure.
	current, err := s.LoadMonth(t)
	if err != nil {
		return fmt.Errorf("re-read for merge: %w", err)
	}
	merged := MergeMonths(current, days, deletedIDs)
	data := markdown.FormatFile(merged)
	return s.writeFile(path, data)
}

// LoadDay returns entries for a specific date.
func (s *Store) LoadDay(t time.Time) ([]model.Entry, error) {
	days, err := s.LoadMonth(t)
	if err != nil {
		return nil, err
	}
	dateStr := t.Format("2006-01-02")
	for _, d := range days {
		if d.Date.Format("2006-01-02") == dateStr {
			return d.Entries, nil
		}
	}
	return nil, nil
}

// AddEntry appends an entry to the appropriate date section in the month file.
func (s *Store) AddEntry(e model.Entry) error {
	if e.ID == "" {
		e.ID = newEntryID()
	}
	if e.UpdatedAt == 0 {
		e.UpdatedAt = now()
	}

	days, err := s.LoadMonth(e.DateTime)
	if err != nil {
		return err
	}

	dateStr := e.DateTime.Format("2006-01-02")
	found := false
	for i, d := range days {
		if d.Date.Format("2006-01-02") == dateStr {
			days[i].Entries = append(days[i].Entries, e)
			days[i].Raw = append(days[i].Raw, model.RawLine{
				IsEntry:    true,
				EntryIndex: len(days[i].Entries) - 1,
			})
			found = true
			break
		}
	}

	if !found {
		day := model.DayLog{
			Date:    truncateToDay(e.DateTime),
			Entries: []model.Entry{e},
			Raw: []model.RawLine{
				{IsEntry: false, Text: ""},
				{IsEntry: true, EntryIndex: 0},
			},
		}
		days = insertDaySorted(days, day)
	}

	return s.SaveMonth(e.DateTime, days)
}

// UpdateEntry replaces the entry at the given index for the given date.
func (s *Store) UpdateEntry(date time.Time, index int, e model.Entry) error {
	e.UpdatedAt = now()

	days, err := s.LoadMonth(date)
	if err != nil {
		return err
	}

	dateStr := date.Format("2006-01-02")
	for i, d := range days {
		if d.Date.Format("2006-01-02") == dateStr {
			if index < 0 || index >= len(d.Entries) {
				return fmt.Errorf("entry index %d out of range (0-%d)", index, len(d.Entries)-1)
			}
			days[i].Entries[index] = e
			return s.SaveMonth(date, days)
		}
	}

	return fmt.Errorf("no entries for date %s", dateStr)
}

// RemoveEntry deletes the entry at the given index for the given date.
func (s *Store) RemoveEntry(date time.Time, index int) error {
	days, err := s.LoadMonth(date)
	if err != nil {
		return err
	}

	dateStr := date.Format("2006-01-02")
	for i, d := range days {
		if d.Date.Format("2006-01-02") == dateStr {
			if index < 0 || index >= len(d.Entries) {
				return fmt.Errorf("entry index %d out of range (0-%d)", index, len(d.Entries)-1)
			}
			// Capture ID before append shifts the underlying slice.
			deletedID := d.Entries[index].ID
			// Remove from entries
			days[i].Entries = append(d.Entries[:index], d.Entries[index+1:]...)
			// Rebuild raw lines: remove the entry's raw line and adjust indices
			var newRaw []model.RawLine
			for _, rl := range d.Raw {
				if rl.IsEntry && rl.EntryIndex == index {
					continue
				}
				if rl.IsEntry && rl.EntryIndex > index {
					rl.EntryIndex--
				}
				newRaw = append(newRaw, rl)
			}
			days[i].Raw = newRaw
			var deletedIDs map[string]struct{}
			if deletedID != "" {
				deletedIDs = map[string]struct{}{deletedID: {}}
			}
			return s.saveMonth(date, days, deletedIDs)
		}
	}

	return fmt.Errorf("no entries for date %s", dateStr)
}

// FilterOpts specifies criteria for filtering entries.
type FilterOpts struct {
	DateFrom time.Time
	DateTo   time.Time
	Project  string
	Person   string
	Symbol   string // symbol name
}

// Filter returns entries matching the given criteria across month files.
func (s *Store) Filter(opts FilterOpts) ([]model.Entry, error) {
	// Determine which months to scan
	start := opts.DateFrom
	end := opts.DateTo

	var results []model.Entry
	current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())

	for !current.After(end) {
		days, err := s.LoadMonth(current)
		if err != nil {
			return nil, err
		}

		for _, d := range days {
			if d.Date.Before(truncateToDay(start)) || d.Date.After(truncateToDay(end)) {
				continue
			}
			for _, e := range d.Entries {
				if opts.Project != "" && !strings.EqualFold(e.Project, opts.Project) {
					continue
				}
				if opts.Person != "" && !strings.EqualFold(e.Person, opts.Person) {
					continue
				}
				if opts.Symbol != "" && !strings.EqualFold(e.Symbol.Name, opts.Symbol) {
					continue
				}
				results = append(results, e)
			}
		}

		current = current.AddDate(0, 1, 0)
	}

	return results, nil
}

// AllProjects scans loaded entries and returns unique project names.
func (s *Store) AllProjects(days []model.DayLog) []string {
	seen := make(map[string]bool)
	var list []string
	for _, d := range days {
		for _, e := range d.Entries {
			if e.Project != "" && !seen[e.Project] {
				seen[e.Project] = true
				list = append(list, e.Project)
			}
		}
	}
	return list
}

// AllPeople scans loaded entries and returns unique person names.
func (s *Store) AllPeople(days []model.DayLog) []string {
	seen := make(map[string]bool)
	var list []string
	for _, d := range days {
		for _, e := range d.Entries {
			if e.Person != "" && !seen[e.Person] {
				seen[e.Person] = true
				list = append(list, e.Person)
			}
		}
	}
	return list
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func insertDaySorted(days []model.DayLog, day model.DayLog) []model.DayLog {
	// Insert in reverse chronological order (newest first)
	for i, d := range days {
		if day.Date.After(d.Date) {
			result := make([]model.DayLog, 0, len(days)+1)
			result = append(result, days[:i]...)
			result = append(result, day)
			result = append(result, days[i:]...)
			return result
		}
	}
	return append(days, day)
}
