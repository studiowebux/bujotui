package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/storage"
)

// parseEntryIndex parses a 1-based entry number from args, loads today's entries,
// and validates the index is in bounds. It returns the date, the 0-based index,
// and the loaded entries.
func parseEntryIndex(args []string, store *storage.Store) (time.Time, int, []model.Entry, error) {
	if len(args) < 1 {
		return time.Time{}, 0, nil, fmt.Errorf("entry number required")
	}

	idx, err := strconv.Atoi(args[0])
	if err != nil {
		return time.Time{}, 0, nil, fmt.Errorf("invalid entry number: %q", args[0])
	}
	idx-- // 1-based to 0-based

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	entries, err := store.LoadDay(today)
	if err != nil {
		return time.Time{}, 0, nil, err
	}

	if idx < 0 || idx >= len(entries) {
		return time.Time{}, 0, nil, fmt.Errorf("entry %d not found (have %d entries)", idx+1, len(entries))
	}

	return today, idx, entries, nil
}
