package service

import (
	"strings"
	"testing"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/storage"
)

// newTestCollectionService creates a CollectionService backed by a temp directory.
func newTestCollectionService(t *testing.T) *CollectionService {
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

	return NewCollectionService(store)
}

// ---------- Create ----------

func TestCollectionCreate(t *testing.T) {
	svc := newTestCollectionService(t)

	col, err := svc.Create("Books to Read")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if col.Name != "Books to Read" {
		t.Errorf("name = %q, want %q", col.Name, "Books to Read")
	}
	if len(col.Items) != 0 {
		t.Errorf("items = %d, want 0", len(col.Items))
	}
}

func TestCollectionCreate_EmptyName(t *testing.T) {
	svc := newTestCollectionService(t)

	_, err := svc.Create("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCollectionCreate_WhitespaceOnlyName(t *testing.T) {
	svc := newTestCollectionService(t)

	_, err := svc.Create("   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestCollectionCreate_TooLongName(t *testing.T) {
	svc := newTestCollectionService(t)

	longName := strings.Repeat("a", 101)
	_, err := svc.Create(longName)
	if err == nil {
		t.Fatal("expected error for name > 100 chars")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCollectionCreate_Duplicate(t *testing.T) {
	svc := newTestCollectionService(t)

	_, err := svc.Create("MyList")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = svc.Create("MyList")
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------- List ----------

func TestCollectionList_Empty(t *testing.T) {
	svc := newTestCollectionService(t)

	names, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 collections, got %d", len(names))
	}
}

func TestCollectionList_Multiple(t *testing.T) {
	svc := newTestCollectionService(t)

	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		if _, err := svc.Create(name); err != nil {
			t.Fatalf("create %q: %v", name, err)
		}
	}

	names, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("expected 3 collections, got %d", len(names))
	}
}

// ---------- Get ----------

func TestCollectionGet(t *testing.T) {
	svc := newTestCollectionService(t)

	_, err := svc.Create("Todo")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	col, err := svc.Get("Todo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if col.Name != "Todo" {
		t.Errorf("name = %q, want %q", col.Name, "Todo")
	}
}

func TestCollectionGet_NotFound(t *testing.T) {
	svc := newTestCollectionService(t)

	_, err := svc.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent collection")
	}
}

// ---------- AddItem ----------

func TestCollectionAddItem(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Groceries"); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.AddItem("Groceries", "Milk"); err != nil {
		t.Fatalf("add item: %v", err)
	}
	if err := svc.AddItem("Groceries", "Bread"); err != nil {
		t.Fatalf("add item: %v", err)
	}

	col, err := svc.Get("Groceries")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(col.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(col.Items))
	}
	if col.Items[0].Text != "Milk" {
		t.Errorf("item[0] = %q, want %q", col.Items[0].Text, "Milk")
	}
	if col.Items[1].Text != "Bread" {
		t.Errorf("item[1] = %q, want %q", col.Items[1].Text, "Bread")
	}
}

func TestCollectionAddItem_EmptyText(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("List"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.AddItem("List", ""); err == nil {
		t.Fatal("expected error for empty item text")
	}
}

// ---------- RemoveItem ----------

func TestCollectionRemoveItem(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("List"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = svc.AddItem("List", "A")
	_ = svc.AddItem("List", "B")
	_ = svc.AddItem("List", "C")

	if err := svc.RemoveItem("List", 1); err != nil {
		t.Fatalf("remove: %v", err)
	}

	col, _ := svc.Get("List")
	if len(col.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(col.Items))
	}
	if col.Items[0].Text != "A" || col.Items[1].Text != "C" {
		t.Errorf("unexpected items: %v", col.Items)
	}
}

func TestCollectionRemoveItem_OutOfRange(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("List"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = svc.AddItem("List", "Only")

	if err := svc.RemoveItem("List", 5); err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}

// ---------- ToggleItem ----------

func TestCollectionToggleItem(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Tasks"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = svc.AddItem("Tasks", "Do laundry")

	// Toggle on.
	if err := svc.ToggleItem("Tasks", 0); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	col, _ := svc.Get("Tasks")
	if !col.Items[0].Done {
		t.Error("expected item to be done after toggle")
	}

	// Toggle off.
	if err := svc.ToggleItem("Tasks", 0); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	col, _ = svc.Get("Tasks")
	if col.Items[0].Done {
		t.Error("expected item to be undone after second toggle")
	}
}

func TestCollectionToggleItem_OutOfRange(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("T"); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.ToggleItem("T", 0); err == nil {
		t.Fatal("expected error for out-of-range toggle on empty collection")
	}
}

// ---------- EditItem ----------

func TestCollectionEditItem(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Notes"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = svc.AddItem("Notes", "Original")

	if err := svc.EditItem("Notes", 0, "Updated"); err != nil {
		t.Fatalf("edit: %v", err)
	}
	col, _ := svc.Get("Notes")
	if col.Items[0].Text != "Updated" {
		t.Errorf("text = %q, want %q", col.Items[0].Text, "Updated")
	}
}

func TestCollectionEditItem_EmptyText(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Notes"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = svc.AddItem("Notes", "Something")

	if err := svc.EditItem("Notes", 0, ""); err == nil {
		t.Fatal("expected error for empty text on edit")
	}
}

func TestCollectionEditItem_OutOfRange(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Notes"); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.EditItem("Notes", 0, "text"); err == nil {
		t.Fatal("expected error for out-of-range edit on empty collection")
	}
}

// ---------- MoveItem ----------

func TestCollectionMoveItem_Forward(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Order"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = svc.AddItem("Order", "A")
	_ = svc.AddItem("Order", "B")
	_ = svc.AddItem("Order", "C")
	_ = svc.AddItem("Order", "D")

	// Move index 0 -> 2 (A should end up after B, before C after adjustment).
	if err := svc.MoveItem("Order", 0, 2); err != nil {
		t.Fatalf("move: %v", err)
	}

	col, _ := svc.Get("Order")
	got := make([]string, len(col.Items))
	for i, item := range col.Items {
		got[i] = item.Text
	}
	// After removing A from position 0: [B, C, D]
	// from(0) < to(2), so to becomes 1
	// Insert A at position 1: [B, A, C, D]
	expected := []string{"B", "A", "C", "D"}
	for i, want := range expected {
		if got[i] != want {
			t.Errorf("position %d: got %q, want %q (full: %v)", i, got[i], want, got)
			break
		}
	}
}

func TestCollectionMoveItem_Backward(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Order"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = svc.AddItem("Order", "A")
	_ = svc.AddItem("Order", "B")
	_ = svc.AddItem("Order", "C")

	// Move index 2 -> 0 (C moves to front).
	if err := svc.MoveItem("Order", 2, 0); err != nil {
		t.Fatalf("move: %v", err)
	}

	col, _ := svc.Get("Order")
	got := make([]string, len(col.Items))
	for i, item := range col.Items {
		got[i] = item.Text
	}
	// After removing C from position 2: [A, B]
	// from(2) > to(0), so to stays 0
	// Insert C at position 0: [C, A, B]
	expected := []string{"C", "A", "B"}
	for i, want := range expected {
		if got[i] != want {
			t.Errorf("position %d: got %q, want %q (full: %v)", i, got[i], want, got)
			break
		}
	}
}

func TestCollectionMoveItem_SameIndex(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Order"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = svc.AddItem("Order", "A")
	_ = svc.AddItem("Order", "B")

	// Moving to the same position should be a no-op.
	if err := svc.MoveItem("Order", 1, 1); err != nil {
		t.Fatalf("move same index: %v", err)
	}

	col, _ := svc.Get("Order")
	if col.Items[0].Text != "A" || col.Items[1].Text != "B" {
		t.Errorf("unexpected order after same-index move: %v", col.Items)
	}
}

func TestCollectionMoveItem_OutOfRange(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Order"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = svc.AddItem("Order", "A")

	if err := svc.MoveItem("Order", 0, 5); err == nil {
		t.Fatal("expected error for out-of-range 'to' index")
	}
	if err := svc.MoveItem("Order", 5, 0); err == nil {
		t.Fatal("expected error for out-of-range 'from' index")
	}
}

// ---------- Delete ----------

func TestCollectionDelete(t *testing.T) {
	svc := newTestCollectionService(t)

	if _, err := svc.Create("Temp"); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := svc.Delete("Temp"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := svc.Get("Temp")
	if err == nil {
		t.Fatal("expected error getting deleted collection")
	}
}

func TestCollectionDelete_NotFound(t *testing.T) {
	svc := newTestCollectionService(t)

	if err := svc.Delete("ghost"); err == nil {
		t.Fatal("expected error deleting nonexistent collection")
	}
}
