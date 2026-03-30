package storage

import (
	"time"

	"github.com/studiowebux/bujotui/internal/model"
)

// MergeMonths merges base (current disk state) into incoming (what this writer wants to save).
//
// Rules:
//   - All entries from incoming are kept as-is (writer's intent).
//   - Entries in base that have an ID not present in incoming are appended
//     to the appropriate day (added by a concurrent writer — must not be lost).
//   - Entries in base without an ID cannot be deduplicated and are ignored
//     (they will already be present in incoming if this writer loaded them first).
//   - Entries whose ID appears in deletedIDs are never brought back from base —
//     they were explicitly removed by this writer.
//   - For entries present in both with the same ID: incoming wins.
func MergeMonths(base, incoming []model.DayLog, deletedIDs map[string]struct{}) []model.DayLog {
	if len(base) == 0 {
		return incoming
	}

	// Build set of IDs already in incoming, keyed by date string.
	type dayKey = string // "YYYY-MM-DD"
	incomingIDs := make(map[dayKey]map[string]struct{})
	for _, day := range incoming {
		dk := day.Date.Format("2006-01-02")
		ids := make(map[string]struct{}, len(day.Entries))
		for _, e := range day.Entries {
			if e.ID != "" {
				ids[e.ID] = struct{}{}
			}
		}
		incomingIDs[dk] = ids
	}

	// Index incoming days by date for fast lookup and mutation.
	incomingIdx := make(map[dayKey]int, len(incoming))
	result := make([]model.DayLog, len(incoming))
	copy(result, incoming)
	for i, day := range result {
		incomingIdx[day.Date.Format("2006-01-02")] = i
	}

	// Walk base: for each entry with an ID not in incoming, add it.
	for _, baseDay := range base {
		dk := baseDay.Date.Format("2006-01-02")
		knownIDs := incomingIDs[dk] // may be nil if this day has no incoming entries

		for _, e := range baseDay.Entries {
			if e.ID == "" {
				continue // legacy entry — cannot dedup, skip
			}
			if _, deleted := deletedIDs[e.ID]; deleted {
				continue // explicitly removed by this writer — do not resurrect
			}
			if _, found := knownIDs[e.ID]; found {
				continue // already in incoming
			}
			// This entry was added by another concurrent writer — append it.
			if idx, ok := incomingIdx[dk]; ok {
				result[idx].Entries = append(result[idx].Entries, e)
				result[idx].Raw = append(result[idx].Raw, model.RawLine{
					IsEntry:    true,
					EntryIndex: len(result[idx].Entries) - 1,
				})
			} else {
				// Day doesn't exist in incoming at all — add it.
				newDay := model.DayLog{
					Date:    truncateToDay(baseDay.Date),
					Entries: []model.Entry{e},
					Raw: []model.RawLine{
						{IsEntry: false, Text: ""},
						{IsEntry: true, EntryIndex: 0},
					},
				}
				result = insertDaySorted(result, newDay)
				// Update index — insertDaySorted may have shifted positions,
				// but we only need the new day's index going forward.
				incomingIdx[dk] = len(result) - 1
				if incomingIDs[dk] == nil {
					incomingIDs[dk] = make(map[string]struct{})
				}
			}
			if incomingIDs[dk] == nil {
				incomingIDs[dk] = make(map[string]struct{})
			}
			incomingIDs[dk][e.ID] = struct{}{} // prevent double-add within same base
		}
	}

	return result
}

// now returns the current unix timestamp. Isolated for testability.
var now = func() int64 { return time.Now().Unix() }
