package service

import (
	"strings"
	"testing"
	"time"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/storage"
)

// testMonth is a fixed month used across habit tests.
var testMonth = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

// newTestHabitService creates a HabitService backed by a temp directory.
func newTestHabitService(t *testing.T) *HabitService {
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

	return NewHabitService(store)
}

// ---------- AddHabit ----------

func TestHabitAdd(t *testing.T) {
	svc := newTestHabitService(t)

	if err := svc.AddHabit(testMonth, "Exercise"); err != nil {
		t.Fatalf("add habit: %v", err)
	}

	ht, err := svc.LoadMonth(testMonth)
	if err != nil {
		t.Fatalf("load month: %v", err)
	}
	if len(ht.Habits) != 1 {
		t.Fatalf("expected 1 habit, got %d", len(ht.Habits))
	}
	if ht.Habits[0] != "Exercise" {
		t.Errorf("habit = %q, want %q", ht.Habits[0], "Exercise")
	}
}

func TestHabitAdd_Multiple(t *testing.T) {
	svc := newTestHabitService(t)

	for _, name := range []string{"Exercise", "Reading", "Meditation"} {
		if err := svc.AddHabit(testMonth, name); err != nil {
			t.Fatalf("add %q: %v", name, err)
		}
	}

	ht, _ := svc.LoadMonth(testMonth)
	if len(ht.Habits) != 3 {
		t.Fatalf("expected 3 habits, got %d", len(ht.Habits))
	}
}

func TestHabitAdd_EmptyName(t *testing.T) {
	svc := newTestHabitService(t)

	if err := svc.AddHabit(testMonth, ""); err == nil {
		t.Fatal("expected error for empty habit name")
	}
}

func TestHabitAdd_WhitespaceOnlyName(t *testing.T) {
	svc := newTestHabitService(t)

	if err := svc.AddHabit(testMonth, "   "); err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestHabitAdd_TooLongName(t *testing.T) {
	svc := newTestHabitService(t)

	longName := strings.Repeat("x", 101)
	if err := svc.AddHabit(testMonth, longName); err == nil {
		t.Fatal("expected error for name > 100 chars")
	}
	if err := svc.AddHabit(testMonth, longName); !strings.Contains(err.Error(), "too long") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHabitAdd_Duplicate(t *testing.T) {
	svc := newTestHabitService(t)

	if err := svc.AddHabit(testMonth, "Run"); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := svc.AddHabit(testMonth, "Run"); err == nil {
		t.Fatal("expected error for duplicate habit")
	}
}

func TestHabitAdd_DuplicateCaseInsensitive(t *testing.T) {
	svc := newTestHabitService(t)

	if err := svc.AddHabit(testMonth, "Run"); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := svc.AddHabit(testMonth, "run"); err == nil {
		t.Fatal("expected error for case-insensitive duplicate")
	}
}

// ---------- Toggle ----------

func TestHabitToggle(t *testing.T) {
	svc := newTestHabitService(t)

	_ = svc.AddHabit(testMonth, "Water")

	// Toggle on.
	if err := svc.Toggle(testMonth, "Water", 5); err != nil {
		t.Fatalf("toggle on: %v", err)
	}
	ht, _ := svc.LoadMonth(testMonth)
	if !ht.IsDone("Water", 5) {
		t.Error("expected day 5 to be done")
	}

	// Toggle off.
	if err := svc.Toggle(testMonth, "Water", 5); err != nil {
		t.Fatalf("toggle off: %v", err)
	}
	ht, _ = svc.LoadMonth(testMonth)
	if ht.IsDone("Water", 5) {
		t.Error("expected day 5 to be undone after second toggle")
	}
}

func TestHabitToggle_MultipleDays(t *testing.T) {
	svc := newTestHabitService(t)

	_ = svc.AddHabit(testMonth, "Study")

	for _, day := range []int{1, 2, 3, 10, 20} {
		if err := svc.Toggle(testMonth, "Study", day); err != nil {
			t.Fatalf("toggle day %d: %v", day, err)
		}
	}

	ht, _ := svc.LoadMonth(testMonth)
	for _, day := range []int{1, 2, 3, 10, 20} {
		if !ht.IsDone("Study", day) {
			t.Errorf("day %d should be done", day)
		}
	}
	if ht.IsDone("Study", 4) {
		t.Error("day 4 should not be done")
	}
}

func TestHabitToggle_NotFound(t *testing.T) {
	svc := newTestHabitService(t)

	if err := svc.Toggle(testMonth, "Ghost", 1); err == nil {
		t.Fatal("expected error for toggling nonexistent habit")
	}
}

// ---------- LoadMonth ----------

func TestHabitLoadMonth_Empty(t *testing.T) {
	svc := newTestHabitService(t)

	ht, err := svc.LoadMonth(testMonth)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(ht.Habits) != 0 {
		t.Errorf("expected 0 habits, got %d", len(ht.Habits))
	}
}

func TestHabitLoadMonth_Persistence(t *testing.T) {
	svc := newTestHabitService(t)

	_ = svc.AddHabit(testMonth, "Run")
	_ = svc.Toggle(testMonth, "Run", 15)

	// Load again to verify persistence.
	ht, err := svc.LoadMonth(testMonth)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(ht.Habits) != 1 || ht.Habits[0] != "Run" {
		t.Errorf("unexpected habits: %v", ht.Habits)
	}
	if !ht.IsDone("Run", 15) {
		t.Error("day 15 should be done after reload")
	}
}

func TestHabitLoadMonth_DifferentMonths(t *testing.T) {
	svc := newTestHabitService(t)

	march := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	april := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	_ = svc.AddHabit(march, "A")
	_ = svc.AddHabit(april, "B")

	htMar, _ := svc.LoadMonth(march)
	htApr, _ := svc.LoadMonth(april)

	if len(htMar.Habits) != 1 || htMar.Habits[0] != "A" {
		t.Errorf("march habits: %v", htMar.Habits)
	}
	if len(htApr.Habits) != 1 || htApr.Habits[0] != "B" {
		t.Errorf("april habits: %v", htApr.Habits)
	}
}

// ---------- RemoveHabit ----------

func TestHabitRemove(t *testing.T) {
	svc := newTestHabitService(t)

	_ = svc.AddHabit(testMonth, "A")
	_ = svc.AddHabit(testMonth, "B")
	_ = svc.AddHabit(testMonth, "C")

	if err := svc.RemoveHabit(testMonth, "B"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	ht, _ := svc.LoadMonth(testMonth)
	if len(ht.Habits) != 2 {
		t.Fatalf("expected 2 habits, got %d", len(ht.Habits))
	}
	for _, h := range ht.Habits {
		if h == "B" {
			t.Error("habit B should have been removed")
		}
	}
}

func TestHabitRemove_NotFound(t *testing.T) {
	svc := newTestHabitService(t)

	if err := svc.RemoveHabit(testMonth, "Ghost"); err == nil {
		t.Fatal("expected error removing nonexistent habit")
	}
	if err := svc.RemoveHabit(testMonth, "Ghost"); !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHabitRemove_ClearsCompletionData(t *testing.T) {
	svc := newTestHabitService(t)

	_ = svc.AddHabit(testMonth, "Temp")
	_ = svc.Toggle(testMonth, "Temp", 1)
	_ = svc.Toggle(testMonth, "Temp", 2)

	if err := svc.RemoveHabit(testMonth, "Temp"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	ht, _ := svc.LoadMonth(testMonth)
	if _, ok := ht.Done["Temp"]; ok {
		t.Error("completion data for removed habit should be gone")
	}
}
