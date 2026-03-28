package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/studiowebux/bujotui/internal/model"
)

// futureDir returns the path to the future log directory.
func (s *Store) futureDir() string {
	return filepath.Join(s.Dir, "future")
}

// ensureFutureDir creates the future directory if needed.
func (s *Store) ensureFutureDir() error {
	return os.MkdirAll(s.futureDir(), 0o700)
}

// futureFile returns the path for a year's future log.
func (s *Store) futureFile(year int) string {
	return filepath.Join(s.futureDir(), fmt.Sprintf("%d.md", year))
}

// LoadFuture reads the future log for a given year.
// Format:
//
//	# Future Log 2026
//
//	## January
//	- . Doctor appointment
//	- o Conference
//
//	## February
//	- . Tax deadline
func (s *Store) LoadFuture(year int) ([]model.FutureMonth, error) {
	path := filepath.Clean(s.futureFile(year))
	f, err := os.Open(path) // #nosec G304 -- path from user-configured data dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open future file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var months []model.FutureMonth
	var current *model.FutureMonth

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Month heading: ## January
		if strings.HasPrefix(line, "## ") {
			monthName := strings.TrimSpace(line[3:])
			monthNum := parseMonthName(monthName)
			if monthNum > 0 {
				if current != nil {
					months = append(months, *current)
				}
				current = &model.FutureMonth{Year: year, Month: monthNum}
			}
			continue
		}

		// Entry: - symbol description
		if current != nil && strings.HasPrefix(line, "- ") {
			rest := line[2:]
			symChar, size := utf8.DecodeRuneInString(rest)
			if symChar != utf8.RuneError && size > 0 {
				sym, ok := s.Config.Symbols.LookupByChar(string(symChar))
				if ok {
					desc := strings.TrimLeft(rest[size:], " ")
					current.Entries = append(current.Entries, model.FutureEntry{
						Symbol:      sym,
						Description: desc,
					})
				}
			}
		}
	}

	if current != nil {
		months = append(months, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse future log: %w", err)
	}

	return months, nil
}

// SaveFuture writes the future log for a year atomically.
func (s *Store) SaveFuture(year int, months []model.FutureMonth) error {
	if err := s.ensureFutureDir(); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Future Log %d\n", year)

	for _, m := range months {
		monthName := time.Month(m.Month).String()
		fmt.Fprintf(&b, "\n## %s\n", monthName)
		for _, e := range m.Entries {
			fmt.Fprintf(&b, "- %s %s\n", e.Symbol.Char, e.Description)
		}
	}

	path := s.futureFile(year)
	return AtomicWriteFile(path, []byte(b.String()), 0o644)
}

// parseMonthName converts a month name to its number (1-12). Returns 0 if unknown.
func parseMonthName(name string) int {
	name = strings.ToLower(strings.TrimSpace(name))
	for m := time.January; m <= time.December; m++ {
		if strings.ToLower(m.String()) == name {
			return int(m)
		}
	}
	return 0
}
