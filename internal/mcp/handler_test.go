package mcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/service"
	"github.com/studiowebux/bujotui/internal/storage"
)

// newTestHandler creates a Handler backed by a temporary directory.
// It returns the handler and the current date string (YYYY-MM-DD) for convenience.
func newTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()

	cfg, err := config.Load(dir, dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	store, err := storage.NewStore(cfg)
	if err != nil {
		t.Fatalf("storage.NewStore: %v", err)
	}

	entrySvc := service.NewEntryService(store, cfg)
	colSvc := service.NewCollectionService(store)
	habSvc := service.NewHabitService(store)
	futSvc := service.NewFutureLogService(store, cfg)

	h := NewHandler(entrySvc, colSvc, habSvc, futSvc)
	today := time.Now().Format("2006-01-02")
	return h, today
}

// mustJSON marshals v to json.RawMessage; panics on error.
func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return json.RawMessage(b)
}

// resultText returns the text content of the first content element.
func resultText(r ToolResult) string {
	if len(r.Content) == 0 {
		return ""
	}
	return r.Content[0].Text
}

// ---------- parseDate ----------

func TestParseDate_Empty(t *testing.T) {
	d, err := parseDate("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	now := time.Now()
	if d.Year() != now.Year() || d.Month() != now.Month() || d.Day() != now.Day() {
		t.Errorf("expected today's date, got %s", d.Format("2006-01-02"))
	}
}

func TestParseDate_Valid(t *testing.T) {
	d, err := parseDate("2025-06-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Format("2006-01-02") != "2025-06-15" {
		t.Errorf("got %s, want 2025-06-15", d.Format("2006-01-02"))
	}
}

func TestParseDate_InvalidFormat(t *testing.T) {
	_, err := parseDate("15/06/2025")
	if err == nil {
		t.Fatal("expected error for invalid date format")
	}
	if !strings.Contains(err.Error(), "invalid date") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---------- parseMonth ----------

func TestParseMonth_Empty(t *testing.T) {
	m, err := parseMonth("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	now := time.Now()
	if m.Year() != now.Year() || m.Month() != now.Month() || m.Day() != 1 {
		t.Errorf("expected first of current month, got %s", m.Format("2006-01-02"))
	}
}

func TestParseMonth_Valid(t *testing.T) {
	m, err := parseMonth("2025-03")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Format("2006-01") != "2025-03" {
		t.Errorf("got %s, want 2025-03", m.Format("2006-01"))
	}
	if m.Day() != 1 {
		t.Errorf("day should be 1, got %d", m.Day())
	}
}

func TestParseMonth_InvalidFormat(t *testing.T) {
	_, err := parseMonth("March 2025")
	if err == nil {
		t.Fatal("expected error for invalid month format")
	}
	if !strings.Contains(err.Error(), "invalid month format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---------- HandleToolCall unknown tool ----------

func TestHandleToolCall_UnknownTool(t *testing.T) {
	h, _ := newTestHandler(t)
	r := h.HandleToolCall("nonexistent_tool", json.RawMessage(`{}`))
	if !r.IsError {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(resultText(r), "unknown tool") {
		t.Errorf("unexpected error text: %s", resultText(r))
	}
}

// ---------- add_entry ----------

func TestAddEntry_Valid(t *testing.T) {
	h, _ := newTestHandler(t)
	args := mustJSON(t, map[string]string{
		"description": "Write unit tests",
		"symbol":      "task",
		"project":     "bujotui",
	})
	r := h.HandleToolCall("add_entry", args)
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "Added") {
		t.Errorf("expected 'Added' in result, got: %s", text)
	}
	if !strings.Contains(text, "Write unit tests") {
		t.Errorf("expected description in result, got: %s", text)
	}
}

func TestAddEntry_MissingDescription(t *testing.T) {
	h, _ := newTestHandler(t)
	args := mustJSON(t, map[string]string{
		"symbol": "task",
	})
	r := h.HandleToolCall("add_entry", args)
	if !r.IsError {
		t.Fatal("expected error for missing description")
	}
	if !strings.Contains(resultText(r), "description") {
		t.Errorf("expected 'description' in error, got: %s", resultText(r))
	}
}

// ---------- list_entries ----------

func TestListEntries_WithEntries(t *testing.T) {
	h, today := newTestHandler(t)

	// Add an entry first.
	addArgs := mustJSON(t, map[string]string{
		"description": "Test entry",
		"symbol":      "task",
	})
	r := h.HandleToolCall("add_entry", addArgs)
	if r.IsError {
		t.Fatalf("add_entry failed: %s", resultText(r))
	}

	// List entries for today.
	listArgs := mustJSON(t, map[string]string{"date": today})
	r = h.HandleToolCall("list_entries", listArgs)
	if r.IsError {
		t.Fatalf("list_entries failed: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "Test entry") {
		t.Errorf("expected 'Test entry' in result, got: %s", text)
	}
	if !strings.Contains(text, "Entries for") {
		t.Errorf("expected header in result, got: %s", text)
	}
}

func TestListEntries_EmptyDay(t *testing.T) {
	h, _ := newTestHandler(t)
	args := mustJSON(t, map[string]string{"date": "2020-01-01"})
	r := h.HandleToolCall("list_entries", args)
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "No entries") {
		t.Errorf("expected 'No entries', got: %s", resultText(r))
	}
}

// ---------- edit_entry ----------

func TestEditEntry_Valid(t *testing.T) {
	h, today := newTestHandler(t)

	// Add an entry to edit.
	addArgs := mustJSON(t, map[string]string{
		"description": "Original text",
		"symbol":      "task",
	})
	r := h.HandleToolCall("add_entry", addArgs)
	if r.IsError {
		t.Fatalf("add_entry failed: %s", resultText(r))
	}

	// Edit the entry.
	editArgs := mustJSON(t, map[string]any{
		"date":        today,
		"index":       0,
		"description": "Updated text",
		"symbol":      "note",
	})
	r = h.HandleToolCall("edit_entry", editArgs)
	if r.IsError {
		t.Fatalf("edit_entry failed: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "Updated entry 0") {
		t.Errorf("expected confirmation, got: %s", text)
	}

	// Verify the edit by listing.
	listArgs := mustJSON(t, map[string]string{"date": today})
	r = h.HandleToolCall("list_entries", listArgs)
	if r.IsError {
		t.Fatalf("list_entries failed: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "Updated text") {
		t.Errorf("expected edited text in list, got: %s", resultText(r))
	}
}

// ---------- transition_entry ----------

func TestTransitionEntry_Valid(t *testing.T) {
	h, today := newTestHandler(t)

	// Add a task entry.
	addArgs := mustJSON(t, map[string]string{
		"description": "Transition me",
		"symbol":      "task",
	})
	r := h.HandleToolCall("add_entry", addArgs)
	if r.IsError {
		t.Fatalf("add_entry failed: %s", resultText(r))
	}

	// Transition to done. The default config defines transitions for task.
	transArgs := mustJSON(t, map[string]any{
		"date":  today,
		"index": 0,
		"state": "done",
	})
	r = h.HandleToolCall("transition_entry", transArgs)
	if r.IsError {
		t.Fatalf("transition_entry failed: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "done") {
		t.Errorf("expected 'done' in result, got: %s", text)
	}
}

// ---------- delete_entry ----------

func TestDeleteEntry_Valid(t *testing.T) {
	h, today := newTestHandler(t)

	// Add an entry to delete.
	addArgs := mustJSON(t, map[string]string{
		"description": "Delete me",
		"symbol":      "task",
	})
	r := h.HandleToolCall("add_entry", addArgs)
	if r.IsError {
		t.Fatalf("add_entry failed: %s", resultText(r))
	}

	// Delete the entry.
	delArgs := mustJSON(t, map[string]any{
		"date":  today,
		"index": 0,
	})
	r = h.HandleToolCall("delete_entry", delArgs)
	if r.IsError {
		t.Fatalf("delete_entry failed: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "Deleted entry 0") {
		t.Errorf("expected confirmation, got: %s", text)
	}

	// Verify deletion.
	listArgs := mustJSON(t, map[string]string{"date": today})
	r = h.HandleToolCall("list_entries", listArgs)
	if r.IsError {
		t.Fatalf("list_entries failed: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "No entries") {
		t.Errorf("expected no entries after delete, got: %s", resultText(r))
	}
}

// ---------- set_note ----------

func TestSetNote_SetAndClear(t *testing.T) {
	h, today := newTestHandler(t)

	// Set a note.
	setArgs := mustJSON(t, map[string]string{
		"date": today,
		"note": "Important reminder",
	})
	r := h.HandleToolCall("set_note", setArgs)
	if r.IsError {
		t.Fatalf("set_note failed: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "Note set") {
		t.Errorf("expected 'Note set', got: %s", text)
	}
	if !strings.Contains(text, "Important reminder") {
		t.Errorf("expected note text in result, got: %s", text)
	}

	// Clear the note.
	clearArgs := mustJSON(t, map[string]string{
		"date": today,
		"note": "",
	})
	r = h.HandleToolCall("set_note", clearArgs)
	if r.IsError {
		t.Fatalf("set_note (clear) failed: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "Cleared note") {
		t.Errorf("expected 'Cleared note', got: %s", resultText(r))
	}
}

// ---------- list_collections ----------

func TestListCollections_Empty(t *testing.T) {
	h, _ := newTestHandler(t)
	r := h.HandleToolCall("list_collections", json.RawMessage(`{}`))
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "No collections") {
		t.Errorf("expected 'No collections', got: %s", resultText(r))
	}
}

func TestListCollections_WithCollections(t *testing.T) {
	h, _ := newTestHandler(t)

	// Create a collection first.
	createArgs := mustJSON(t, map[string]string{"name": "Books"})
	r := h.HandleToolCall("create_collection", createArgs)
	if r.IsError {
		t.Fatalf("create_collection failed: %s", resultText(r))
	}

	// List collections.
	r = h.HandleToolCall("list_collections", json.RawMessage(`{}`))
	if r.IsError {
		t.Fatalf("list_collections failed: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "Books") {
		t.Errorf("expected 'Books' in list, got: %s", text)
	}
	if !strings.Contains(text, "Collections:") {
		t.Errorf("expected header, got: %s", text)
	}
}

// ---------- create_collection ----------

func TestCreateCollection_Valid(t *testing.T) {
	h, _ := newTestHandler(t)
	args := mustJSON(t, map[string]string{"name": "Movies"})
	r := h.HandleToolCall("create_collection", args)
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "Created collection: Movies") {
		t.Errorf("unexpected result: %s", resultText(r))
	}
}

func TestCreateCollection_Duplicate(t *testing.T) {
	h, _ := newTestHandler(t)
	args := mustJSON(t, map[string]string{"name": "Music"})

	// Create first.
	r := h.HandleToolCall("create_collection", args)
	if r.IsError {
		t.Fatalf("first create failed: %s", resultText(r))
	}

	// Create duplicate.
	r = h.HandleToolCall("create_collection", args)
	if !r.IsError {
		t.Fatal("expected error for duplicate collection")
	}
	if !strings.Contains(resultText(r), "already exists") {
		t.Errorf("expected 'already exists' error, got: %s", resultText(r))
	}
}

// ---------- add_collection_item ----------

func TestAddCollectionItem_Valid(t *testing.T) {
	h, _ := newTestHandler(t)

	// Create collection first.
	createArgs := mustJSON(t, map[string]string{"name": "Groceries"})
	r := h.HandleToolCall("create_collection", createArgs)
	if r.IsError {
		t.Fatalf("create_collection failed: %s", resultText(r))
	}

	// Add item.
	addArgs := mustJSON(t, map[string]string{
		"name": "Groceries",
		"text": "Milk",
	})
	r = h.HandleToolCall("add_collection_item", addArgs)
	if r.IsError {
		t.Fatalf("add_collection_item failed: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "Added item") {
		t.Errorf("expected 'Added item', got: %s", text)
	}
	if !strings.Contains(text, "Milk") {
		t.Errorf("expected 'Milk' in result, got: %s", text)
	}
}

// ---------- list_habits ----------

func TestListHabits_Empty(t *testing.T) {
	h, _ := newTestHandler(t)
	month := time.Now().Format("2006-01")
	args := mustJSON(t, map[string]string{"month": month})
	r := h.HandleToolCall("list_habits", args)
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "No habits") {
		t.Errorf("expected 'No habits', got: %s", resultText(r))
	}
}

// ---------- add_habit ----------

func TestAddHabit_Valid(t *testing.T) {
	h, _ := newTestHandler(t)
	month := time.Now().Format("2006-01")
	args := mustJSON(t, map[string]string{
		"month": month,
		"name":  "Exercise",
	})
	r := h.HandleToolCall("add_habit", args)
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "Added habit: Exercise") {
		t.Errorf("unexpected result: %s", resultText(r))
	}

	// Verify it appears in the list.
	listArgs := mustJSON(t, map[string]string{"month": month})
	r = h.HandleToolCall("list_habits", listArgs)
	if r.IsError {
		t.Fatalf("list_habits failed: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "Exercise") {
		t.Errorf("expected 'Exercise' in habit list, got: %s", resultText(r))
	}
}

// ---------- toggle_habit ----------

func TestToggleHabit_Valid(t *testing.T) {
	h, _ := newTestHandler(t)
	month := time.Now().Format("2006-01")

	// Add habit first.
	addArgs := mustJSON(t, map[string]string{
		"month": month,
		"name":  "Read",
	})
	r := h.HandleToolCall("add_habit", addArgs)
	if r.IsError {
		t.Fatalf("add_habit failed: %s", resultText(r))
	}

	// Toggle day 5.
	toggleArgs := mustJSON(t, map[string]any{
		"month": month,
		"name":  "Read",
		"day":   5,
	})
	r = h.HandleToolCall("toggle_habit", toggleArgs)
	if r.IsError {
		t.Fatalf("toggle_habit failed: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "Toggled Read on day 5") {
		t.Errorf("unexpected result: %s", text)
	}
}

// ---------- list_future ----------

func TestListFuture_Empty(t *testing.T) {
	h, _ := newTestHandler(t)
	args := mustJSON(t, map[string]any{"year": 2099})
	r := h.HandleToolCall("list_future", args)
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "No future log entries") {
		t.Errorf("expected 'No future log entries', got: %s", resultText(r))
	}
}

// ---------- add_future_entry ----------

func TestAddFutureEntry_Valid(t *testing.T) {
	h, _ := newTestHandler(t)
	year := time.Now().Year()
	args := mustJSON(t, map[string]any{
		"year":        year,
		"month":       6,
		"symbol":      "event",
		"description": "Summer conference",
	})
	r := h.HandleToolCall("add_future_entry", args)
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}
	text := resultText(r)
	if !strings.Contains(text, "Summer conference") {
		t.Errorf("expected description in result, got: %s", text)
	}
	if !strings.Contains(text, "June") {
		t.Errorf("expected 'June' in result, got: %s", text)
	}

	// Verify via list_future.
	listArgs := mustJSON(t, map[string]any{"year": year})
	r = h.HandleToolCall("list_future", listArgs)
	if r.IsError {
		t.Fatalf("list_future failed: %s", resultText(r))
	}
	if !strings.Contains(resultText(r), "Summer conference") {
		t.Errorf("expected entry in future log, got: %s", resultText(r))
	}
}
