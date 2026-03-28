package service

import (
	"testing"

	"github.com/studiowebux/bujotui/internal/model"
)

// ---------- checkIndex ----------

func TestCheckIndex_Valid(t *testing.T) {
	for _, tc := range []struct {
		index, length int
	}{
		{0, 1},
		{0, 5},
		{4, 5},
		{2, 10},
	} {
		if err := checkIndex(tc.index, tc.length); err != nil {
			t.Errorf("checkIndex(%d, %d) unexpected error: %v", tc.index, tc.length, err)
		}
	}
}

func TestCheckIndex_OutOfRange(t *testing.T) {
	for _, tc := range []struct {
		index, length int
	}{
		{-1, 5},
		{5, 5},
		{0, 0},
		{10, 3},
	} {
		if err := checkIndex(tc.index, tc.length); err == nil {
			t.Errorf("checkIndex(%d, %d) expected error, got nil", tc.index, tc.length)
		}
	}
}

// ---------- containsFold ----------

func TestContainsFold_Match(t *testing.T) {
	tests := []struct {
		s, lower string
	}{
		{"Hello World", "hello"},
		{"Hello World", "world"},
		{"UPPER", "upper"},
		{"mixedCase", "mixedcase"},
		{"abc", "abc"},
	}
	for _, tc := range tests {
		if !containsFold(tc.s, tc.lower) {
			t.Errorf("containsFold(%q, %q) = false, want true", tc.s, tc.lower)
		}
	}
}

func TestContainsFold_NoMatch(t *testing.T) {
	tests := []struct {
		s, lower string
	}{
		{"Hello", "xyz"},
		{"abc", "abcd"},
		{"", "something"},
	}
	for _, tc := range tests {
		if containsFold(tc.s, tc.lower) {
			t.Errorf("containsFold(%q, %q) = true, want false", tc.s, tc.lower)
		}
	}
}

func TestContainsFold_EmptySubstr(t *testing.T) {
	// Empty substring is always contained.
	if !containsFold("anything", "") {
		t.Error("containsFold with empty substr should return true")
	}
}

// ---------- FilterEntries ----------

func makeEntries() []model.Entry {
	return []model.Entry{
		{
			Symbol:      model.Symbol{Name: "task", Char: "."},
			State:       "",
			Project:     "ProjectA",
			Person:      "Alice",
			Description: "Write unit tests",
		},
		{
			Symbol:      model.Symbol{Name: "event", Char: "o"},
			State:       "",
			Project:     "ProjectB",
			Person:      "Bob",
			Description: "Team standup meeting",
		},
		{
			Symbol:      model.Symbol{Name: "task", Char: "."},
			State:       "done",
			Project:     "ProjectA",
			Person:      "Alice",
			Description: "Review pull request",
		},
		{
			Symbol:      model.Symbol{Name: "note", Char: "-"},
			State:       "",
			Project:     "ProjectC",
			Person:      "Charlie",
			Description: "Remember to buy milk",
		},
	}
}

func TestFilterEntries_NoFilter(t *testing.T) {
	entries := makeEntries()
	result := FilterEntries(entries, "", "", "", "")
	if len(result) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(result))
	}
	// Verify it is a copy, not the same slice.
	if &result[0] == &entries[0] {
		t.Error("expected a copy of the slice, got the same backing array")
	}
}

func TestFilterEntries_ProjectFilter(t *testing.T) {
	entries := makeEntries()
	result := FilterEntries(entries, "projecta", "", "", "")
	if len(result) != 2 {
		t.Fatalf("expected 2 entries for ProjectA, got %d", len(result))
	}
	for _, e := range result {
		if e.Project != "ProjectA" {
			t.Errorf("unexpected project %q", e.Project)
		}
	}
}

func TestFilterEntries_PersonFilter(t *testing.T) {
	entries := makeEntries()
	result := FilterEntries(entries, "", "bob", "", "")
	if len(result) != 1 {
		t.Fatalf("expected 1 entry for Bob, got %d", len(result))
	}
	if result[0].Person != "Bob" {
		t.Errorf("expected Bob, got %q", result[0].Person)
	}
}

func TestFilterEntries_SymbolFilter(t *testing.T) {
	entries := makeEntries()
	result := FilterEntries(entries, "", "", "task", "")
	if len(result) != 2 {
		t.Fatalf("expected 2 task entries, got %d", len(result))
	}
	for _, e := range result {
		if e.Symbol.Name != "task" {
			t.Errorf("unexpected symbol %q", e.Symbol.Name)
		}
	}
}

func TestFilterEntries_TextSearchDescription(t *testing.T) {
	entries := makeEntries()
	result := FilterEntries(entries, "", "", "", "unit tests")
	if len(result) != 1 {
		t.Fatalf("expected 1 entry matching 'unit tests', got %d", len(result))
	}
	if result[0].Description != "Write unit tests" {
		t.Errorf("unexpected description %q", result[0].Description)
	}
}

func TestFilterEntries_TextSearchProject(t *testing.T) {
	entries := makeEntries()
	// Text search also matches against project.
	result := FilterEntries(entries, "", "", "", "projectb")
	if len(result) != 1 {
		t.Fatalf("expected 1 entry matching projectb in project field, got %d", len(result))
	}
}

func TestFilterEntries_TextSearchPerson(t *testing.T) {
	entries := makeEntries()
	result := FilterEntries(entries, "", "", "", "charlie")
	if len(result) != 1 {
		t.Fatalf("expected 1 entry matching charlie, got %d", len(result))
	}
}

func TestFilterEntries_TextSearchSymbol(t *testing.T) {
	entries := makeEntries()
	result := FilterEntries(entries, "", "", "", "note")
	if len(result) != 1 {
		t.Fatalf("expected 1 entry matching note symbol, got %d", len(result))
	}
}

func TestFilterEntries_TextSearchState(t *testing.T) {
	entries := makeEntries()
	result := FilterEntries(entries, "", "", "", "done")
	if len(result) != 1 {
		t.Fatalf("expected 1 entry with state 'done', got %d", len(result))
	}
}

func TestFilterEntries_CombinedFilters(t *testing.T) {
	entries := makeEntries()
	// Filter by project=ProjectA AND symbol=task -> 2 results.
	result := FilterEntries(entries, "ProjectA", "", "task", "")
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}

	// Add text filter to narrow further.
	result = FilterEntries(entries, "ProjectA", "", "task", "review")
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].Description != "Review pull request" {
		t.Errorf("unexpected entry: %q", result[0].Description)
	}
}

func TestFilterEntries_NoMatch(t *testing.T) {
	entries := makeEntries()
	result := FilterEntries(entries, "nonexistent", "", "", "")
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestFilterEntries_EmptySlice(t *testing.T) {
	result := FilterEntries(nil, "", "", "", "")
	if len(result) != 0 {
		t.Fatalf("expected 0 for nil input, got %d", len(result))
	}

	result = FilterEntries([]model.Entry{}, "proj", "", "", "")
	if len(result) != 0 {
		t.Fatalf("expected 0 for empty input, got %d", len(result))
	}
}
