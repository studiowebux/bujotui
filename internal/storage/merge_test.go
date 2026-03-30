package storage

import (
	"testing"
	"time"

	"github.com/studiowebux/bujotui/internal/model"
)

func makeEntry(id, desc string) model.Entry {
	return model.Entry{
		ID:          id,
		Description: desc,
		UpdatedAt:   1000,
	}
}

func makeDay(date string, entries ...model.Entry) model.DayLog {
	t, _ := time.ParseInLocation("2006-01-02", date, time.Local)
	raw := make([]model.RawLine, len(entries))
	for i := range entries {
		raw[i] = model.RawLine{IsEntry: true, EntryIndex: i}
	}
	return model.DayLog{Date: t, Entries: entries, Raw: raw}
}

func entryIDs(day model.DayLog) []string {
	ids := make([]string, len(day.Entries))
	for i, e := range day.Entries {
		ids[i] = e.ID
	}
	return ids
}

func TestMergeMonths_EmptyBase(t *testing.T) {
	incoming := []model.DayLog{makeDay("2026-03-29", makeEntry("a", "task A"))}
	result := MergeMonths(nil, incoming, nil)
	if len(result) != 1 || len(result[0].Entries) != 1 {
		t.Fatalf("expected 1 day with 1 entry, got %v", result)
	}
}

func TestMergeMonths_NoConcurrentWrite(t *testing.T) {
	// base == incoming: no extra entries should appear
	e := makeEntry("a", "task A")
	base := []model.DayLog{makeDay("2026-03-29", e)}
	incoming := []model.DayLog{makeDay("2026-03-29", e)}
	result := MergeMonths(base, incoming, nil)
	if len(result[0].Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result[0].Entries))
	}
}

func TestMergeMonths_ConcurrentAdd(t *testing.T) {
	// Writer A loaded [a], added [b] → incoming=[a,b]
	// Writer B added [c] to disk while A was working → base=[a,c]
	// Merge should produce [a,b,c]
	eA := makeEntry("a", "original")
	eB := makeEntry("b", "added by A")
	eC := makeEntry("c", "added by B concurrently")

	base := []model.DayLog{makeDay("2026-03-29", eA, eC)}
	incoming := []model.DayLog{makeDay("2026-03-29", eA, eB)}

	result := MergeMonths(base, incoming, nil)
	ids := entryIDs(result[0])
	if len(ids) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(ids), ids)
	}
	idSet := map[string]bool{"a": false, "b": false, "c": false}
	for _, id := range ids {
		idSet[id] = true
	}
	for id, found := range idSet {
		if !found {
			t.Errorf("expected entry %q to be in result", id)
		}
	}
}

func TestMergeMonths_DeletedIDNotResurrected(t *testing.T) {
	// Writer deletes entry "a". Base still has "a" (from before delete).
	// Merge must not bring "a" back.
	eA := makeEntry("a", "to delete")
	eB := makeEntry("b", "keep")

	base := []model.DayLog{makeDay("2026-03-29", eA, eB)}
	incoming := []model.DayLog{makeDay("2026-03-29", eB)} // "a" removed

	result := MergeMonths(base, incoming, map[string]struct{}{"a": {}})
	ids := entryIDs(result[0])
	if len(ids) != 1 || ids[0] != "b" {
		t.Fatalf("expected only entry b, got %v", ids)
	}
}

func TestMergeMonths_LegacyEntriesIgnoredFromBase(t *testing.T) {
	// Legacy entries (no ID) in base must not be duplicated.
	legacy := model.Entry{Description: "legacy, no id"}
	eB := makeEntry("b", "new")

	base := []model.DayLog{makeDay("2026-03-29", legacy)}
	incoming := []model.DayLog{makeDay("2026-03-29", legacy, eB)}

	result := MergeMonths(base, incoming, nil)
	if len(result[0].Entries) != 2 {
		t.Fatalf("expected 2 entries (legacy+b), got %d", len(result[0].Entries))
	}
}

func TestMergeMonths_ConcurrentAddNewDay(t *testing.T) {
	// Writer B added an entry on a day that Writer A never touched.
	eA := makeEntry("a", "march 29")
	eB := makeEntry("b", "march 28 — added concurrently")

	base := []model.DayLog{
		makeDay("2026-03-29", eA),
		makeDay("2026-03-28", eB),
	}
	incoming := []model.DayLog{makeDay("2026-03-29", eA)}

	result := MergeMonths(base, incoming, nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 days, got %d", len(result))
	}
}

func TestMergeMonths_ConcurrentAddNewDayMultipleEntries(t *testing.T) {
	// Regression for incomingIdx bug: a base-only day with multiple entries
	// must have all of them appended to the correct day, not scattered.
	eA := makeEntry("a", "march 29")
	eB := makeEntry("b", "march 28 first")
	eC := makeEntry("c", "march 28 second")

	base := []model.DayLog{
		makeDay("2026-03-29", eA),
		makeDay("2026-03-28", eB, eC),
	}
	incoming := []model.DayLog{makeDay("2026-03-29", eA)}

	result := MergeMonths(base, incoming, nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 days, got %d", len(result))
	}
	// Find march 28
	var march28 *model.DayLog
	for i := range result {
		if result[i].Date.Format("2006-01-02") == "2026-03-28" {
			march28 = &result[i]
		}
	}
	if march28 == nil {
		t.Fatal("march 28 not found in result")
	}
	if len(march28.Entries) != 2 {
		t.Fatalf("expected 2 entries on march 28, got %d: %v", len(march28.Entries), entryIDs(*march28))
	}
}
