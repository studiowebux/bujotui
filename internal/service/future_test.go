package service

import (
	"strings"
	"testing"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/storage"
)

// newTestFutureLogService creates a FutureLogService backed by a temp directory.
func newTestFutureLogService(t *testing.T) *FutureLogService {
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

	return NewFutureLogService(store, cfg)
}

// ---------- LoadYear ----------

func TestLoadYear_Empty(t *testing.T) {
	svc := newTestFutureLogService(t)

	months, err := svc.LoadYear(2026)
	if err != nil {
		t.Fatalf("load year: %v", err)
	}
	if len(months) != 0 {
		t.Errorf("expected 0 months, got %d", len(months))
	}
}

func TestLoadYear_AfterAddingEntries(t *testing.T) {
	svc := newTestFutureLogService(t)

	if err := svc.AddEntry(2026, 3, "task", "Plan vacation"); err != nil {
		t.Fatalf("add entry: %v", err)
	}
	if err := svc.AddEntry(2026, 6, "task", "Submit report"); err != nil {
		t.Fatalf("add entry: %v", err)
	}

	months, err := svc.LoadYear(2026)
	if err != nil {
		t.Fatalf("load year: %v", err)
	}
	if len(months) != 2 {
		t.Fatalf("expected 2 months, got %d", len(months))
	}
	if months[0].Month != 3 {
		t.Errorf("first month = %d, want 3", months[0].Month)
	}
	if months[1].Month != 6 {
		t.Errorf("second month = %d, want 6", months[1].Month)
	}
	if months[0].Entries[0].Description != "Plan vacation" {
		t.Errorf("description = %q, want %q", months[0].Entries[0].Description, "Plan vacation")
	}
}

// ---------- AddEntry ----------

func TestFutureAddEntry_Valid(t *testing.T) {
	svc := newTestFutureLogService(t)

	if err := svc.AddEntry(2026, 1, "task", "New Year goals"); err != nil {
		t.Fatalf("add entry: %v", err)
	}

	months, _ := svc.LoadYear(2026)
	if len(months) != 1 {
		t.Fatalf("expected 1 month, got %d", len(months))
	}
	if months[0].Entries[0].Symbol.Name != "task" {
		t.Errorf("symbol = %q, want %q", months[0].Entries[0].Symbol.Name, "task")
	}
}

func TestFutureAddEntry_DefaultSymbol(t *testing.T) {
	svc := newTestFutureLogService(t)

	if err := svc.AddEntry(2026, 5, "", "Use default symbol"); err != nil {
		t.Fatalf("add entry: %v", err)
	}

	months, _ := svc.LoadYear(2026)
	if months[0].Entries[0].Symbol.Name != "task" {
		t.Errorf("symbol = %q, want %q", months[0].Entries[0].Symbol.Name, "task")
	}
}

func TestFutureAddEntry_EmptyDescription(t *testing.T) {
	svc := newTestFutureLogService(t)

	err := svc.AddEntry(2026, 1, "task", "")
	if err == nil {
		t.Fatal("expected error for empty description")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFutureAddEntry_InvalidMonth_Zero(t *testing.T) {
	svc := newTestFutureLogService(t)

	err := svc.AddEntry(2026, 0, "task", "Bad month")
	if err == nil {
		t.Fatal("expected error for month 0")
	}
}

func TestFutureAddEntry_InvalidMonth_Thirteen(t *testing.T) {
	svc := newTestFutureLogService(t)

	err := svc.AddEntry(2026, 13, "task", "Bad month")
	if err == nil {
		t.Fatal("expected error for month 13")
	}
}

func TestFutureAddEntry_UnknownSymbol(t *testing.T) {
	svc := newTestFutureLogService(t)

	err := svc.AddEntry(2026, 1, "nonexistent", "Some entry")
	if err == nil {
		t.Fatal("expected error for unknown symbol")
	}
}

func TestFutureAddEntry_AddsToExistingMonth(t *testing.T) {
	svc := newTestFutureLogService(t)

	_ = svc.AddEntry(2026, 4, "task", "First")
	_ = svc.AddEntry(2026, 4, "task", "Second")

	months, _ := svc.LoadYear(2026)
	if len(months) != 1 {
		t.Fatalf("expected 1 month, got %d", len(months))
	}
	if len(months[0].Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(months[0].Entries))
	}
}

func TestFutureAddEntry_SortedOrder(t *testing.T) {
	svc := newTestFutureLogService(t)

	_ = svc.AddEntry(2026, 12, "task", "December")
	_ = svc.AddEntry(2026, 3, "task", "March")
	_ = svc.AddEntry(2026, 7, "task", "July")

	months, _ := svc.LoadYear(2026)
	want := []int{3, 7, 12}
	for i, m := range months {
		if m.Month != want[i] {
			t.Errorf("months[%d].Month = %d, want %d", i, m.Month, want[i])
		}
	}
}

// ---------- RemoveEntry ----------

func TestFutureRemoveEntry_Valid(t *testing.T) {
	svc := newTestFutureLogService(t)

	_ = svc.AddEntry(2026, 5, "task", "A")
	_ = svc.AddEntry(2026, 5, "task", "B")
	_ = svc.AddEntry(2026, 5, "task", "C")

	if err := svc.RemoveEntry(2026, 5, 1); err != nil {
		t.Fatalf("remove: %v", err)
	}

	months, _ := svc.LoadYear(2026)
	if len(months[0].Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(months[0].Entries))
	}
	if months[0].Entries[0].Description != "A" {
		t.Errorf("entry[0] = %q, want A", months[0].Entries[0].Description)
	}
	if months[0].Entries[1].Description != "C" {
		t.Errorf("entry[1] = %q, want C", months[0].Entries[1].Description)
	}
}

func TestFutureRemoveEntry_OutOfRange(t *testing.T) {
	svc := newTestFutureLogService(t)
	_ = svc.AddEntry(2026, 5, "task", "Only entry")

	err := svc.RemoveEntry(2026, 5, 5)
	if err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}

func TestFutureRemoveEntry_MonthNotFound(t *testing.T) {
	svc := newTestFutureLogService(t)
	_ = svc.AddEntry(2026, 3, "task", "March entry")

	err := svc.RemoveEntry(2026, 9, 0)
	if err == nil {
		t.Fatal("expected error for month not found")
	}
}

// ---------- Multi-year ----------

func TestFutureMultiYear(t *testing.T) {
	svc := newTestFutureLogService(t)

	_ = svc.AddEntry(2026, 1, "task", "Entry in 2026")
	_ = svc.AddEntry(2027, 1, "task", "Entry in 2027")

	m2026, _ := svc.LoadYear(2026)
	m2027, _ := svc.LoadYear(2027)

	if len(m2026) != 1 || len(m2027) != 1 {
		t.Fatalf("expected 1 month each, got %d and %d", len(m2026), len(m2027))
	}
	if m2026[0].Entries[0].Description != "Entry in 2026" {
		t.Errorf("2026: %q", m2026[0].Entries[0].Description)
	}
	if m2027[0].Entries[0].Description != "Entry in 2027" {
		t.Errorf("2027: %q", m2027[0].Entries[0].Description)
	}
}

// ---------- insertMonthSorted ----------

func TestInsertMonthSorted_EmptySlice(t *testing.T) {
	result := insertMonthSorted(nil, model.FutureMonth{Year: 2026, Month: 5})
	if len(result) != 1 || result[0].Month != 5 {
		t.Errorf("expected [5], got %v", monthNums(result))
	}
}

func TestInsertMonthSorted_Beginning(t *testing.T) {
	months := []model.FutureMonth{{Year: 2026, Month: 6}, {Year: 2026, Month: 9}}
	result := insertMonthSorted(months, model.FutureMonth{Year: 2026, Month: 2})
	want := []int{2, 6, 9}
	got := monthNums(result)
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}

func TestInsertMonthSorted_End(t *testing.T) {
	months := []model.FutureMonth{{Year: 2026, Month: 1}, {Year: 2026, Month: 3}}
	result := insertMonthSorted(months, model.FutureMonth{Year: 2026, Month: 11})
	want := []int{1, 3, 11}
	got := monthNums(result)
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}

func monthNums(months []model.FutureMonth) []int {
	nums := make([]int, len(months))
	for i, m := range months {
		nums[i] = m.Month
	}
	return nums
}
