package service

import (
	"strings"
	"testing"
	"time"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/storage"
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

// ---------- EntryService CRUD ----------

func setupTestService(t *testing.T) (*EntryService, *storage.Store) {
	t.Helper()
	dir := t.TempDir()
	cfg, err := config.Load(dir, dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	store, err := storage.NewStore(cfg)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return NewEntryService(store, cfg), store
}

func TestAddEntry_Valid(t *testing.T) {
	svc, _ := setupTestService(t)
	entry, err := svc.AddEntry("task", "", "work", "self", "Buy milk")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if entry.Description != "Buy milk" {
		t.Errorf("desc = %q, want %q", entry.Description, "Buy milk")
	}
	if entry.Symbol.Name != "task" {
		t.Errorf("symbol = %q, want task", entry.Symbol.Name)
	}

	entries, _ := svc.LoadDay(time.Now())
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestAddEntry_EmptyDescription(t *testing.T) {
	svc, _ := setupTestService(t)
	_, err := svc.AddEntry("task", "", "", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddEntry_TooLong(t *testing.T) {
	svc, _ := setupTestService(t)
	_, err := svc.AddEntry("task", "", "", "", strings.Repeat("x", 1001))
	if err == nil {
		t.Fatal("expected error for too long description")
	}
}

func TestAddEntry_UnknownSymbol(t *testing.T) {
	svc, _ := setupTestService(t)
	_, err := svc.AddEntry("nonexistent", "", "", "", "test")
	if err == nil {
		t.Fatal("expected error for unknown symbol")
	}
}

func TestAddEntry_StampsID(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("task", "", "", "", "needs an id")

	entries, _ := svc.LoadDay(time.Now())
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID == "" {
		t.Error("expected entry to have a non-empty ID after AddEntry")
	}
	if entries[0].UpdatedAt == 0 {
		t.Error("expected entry to have a non-zero UpdatedAt after AddEntry")
	}
}

func TestEditEntry_Valid(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("task", "", "work", "self", "Original")

	err := svc.EditEntry(time.Now(), 0, "task", "", "work", "self", "Updated")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}

	entries, _ := svc.LoadDay(time.Now())
	if entries[0].Description != "Updated" {
		t.Errorf("desc = %q, want Updated", entries[0].Description)
	}
}

func TestEditEntry_PreservesID(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("task", "", "", "", "Original")

	before, _ := svc.LoadDay(time.Now())
	if before[0].ID == "" {
		t.Fatal("entry has no ID before edit")
	}
	originalID := before[0].ID

	svc.EditEntry(time.Now(), 0, "task", "", "", "", "Updated")

	after, _ := svc.LoadDay(time.Now())
	if after[0].ID != originalID {
		t.Errorf("ID changed after edit: got %q, want %q", after[0].ID, originalID)
	}
}

func TestEditEntry_OutOfRange(t *testing.T) {
	svc, _ := setupTestService(t)
	err := svc.EditEntry(time.Now(), 0, "task", "", "", "", "test")
	if err == nil {
		t.Fatal("expected error for out of range")
	}
}

func TestDeleteEntry_Valid(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("task", "", "", "", "Delete me")

	err := svc.DeleteEntry(time.Now(), 0)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	entries, _ := svc.LoadDay(time.Now())
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestDeleteEntry_OutOfRange(t *testing.T) {
	svc, _ := setupTestService(t)
	err := svc.DeleteEntry(time.Now(), 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTransitionEntry_TaskToDone(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("task", "", "", "", "Finish this")

	err := svc.TransitionEntry(time.Now(), 0, "done")
	if err != nil {
		t.Fatalf("transition: %v", err)
	}

	entries, _ := svc.LoadDay(time.Now())
	if entries[0].State != "done" {
		t.Errorf("state = %q, want done", entries[0].State)
	}
}

func TestTransitionEntry_InvalidTransition(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("event", "", "", "", "Meeting")

	err := svc.TransitionEntry(time.Now(), 0, "done")
	if err == nil {
		t.Fatal("expected error — event has no transitions")
	}
}

func TestResetState_ClearsStateAndLinks(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("task", "", "", "", "Migrate me")

	tomorrow := time.Now().AddDate(0, 0, 1)
	svc.MigrateEntry(time.Now(), 0, tomorrow)

	// Original should be migrated with link
	entries, _ := svc.LoadDay(time.Now())
	if entries[0].State != "migrated" {
		t.Fatalf("state = %q, want migrated", entries[0].State)
	}
	if entries[0].MigratedTo == "" {
		t.Fatal("MigratedTo should be set")
	}

	// Reset
	err := svc.ResetState(time.Now(), 0)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}

	entries, _ = svc.LoadDay(time.Now())
	if entries[0].State != "" {
		t.Errorf("state = %q, want empty", entries[0].State)
	}
	if entries[0].MigratedTo != "" {
		t.Errorf("MigratedTo = %q, want empty", entries[0].MigratedTo)
	}
}

func TestMigrateEntry_Valid(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("task", "", "work", "self", "Move this")

	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)

	err := svc.MigrateEntry(today, 0, tomorrow)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Check original
	origEntries, _ := svc.LoadDay(today)
	if origEntries[0].State != "migrated" {
		t.Errorf("original state = %q, want migrated", origEntries[0].State)
	}
	if origEntries[0].MigratedTo != tomorrow.Format("2006-01-02") {
		t.Errorf("MigratedTo = %q", origEntries[0].MigratedTo)
	}

	// Check copy
	copyEntries, _ := svc.LoadDay(tomorrow)
	if len(copyEntries) != 1 {
		t.Fatalf("expected 1 copy, got %d", len(copyEntries))
	}
	if copyEntries[0].Description != "Move this" {
		t.Errorf("copy desc = %q", copyEntries[0].Description)
	}
	if copyEntries[0].MigratedFrom != today.Format("2006-01-02") {
		t.Errorf("MigratedFrom = %q", copyEntries[0].MigratedFrom)
	}
	if copyEntries[0].State != "" {
		t.Errorf("copy state = %q, want empty", copyEntries[0].State)
	}
}

func TestMigrateEntry_SameDay(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("task", "", "", "", "test")

	err := svc.MigrateEntry(time.Now(), 0, time.Now())
	if err == nil {
		t.Fatal("expected error for same day migration")
	}
}

func TestMigrateEntry_AlreadyMigrated(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.AddEntry("task", "", "", "", "test")

	tomorrow := time.Now().AddDate(0, 0, 1)
	svc.MigrateEntry(time.Now(), 0, tomorrow)

	err := svc.MigrateEntry(time.Now(), 0, tomorrow.AddDate(0, 0, 1))
	if err == nil {
		t.Fatal("expected error for already migrated")
	}
}

func TestLoadDay_Empty(t *testing.T) {
	svc, _ := setupTestService(t)
	entries, err := svc.LoadDay(time.Now())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil, got %d entries", len(entries))
	}
}
