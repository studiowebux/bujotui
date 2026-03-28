package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir: dir,
		Symbols: model.NewSymbolSet(),
	}
	s, err := NewStore(cfg)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

// --- sanitizeFilename tests ---

func TestSanitizeFilename_Normal(t *testing.T) {
	got := sanitizeFilename("MyProject")
	if got != "myproject" {
		t.Errorf("sanitizeFilename('MyProject') = %q, want %q", got, "myproject")
	}
}

func TestSanitizeFilename_SpecialChars(t *testing.T) {
	got := sanitizeFilename("hello world! foo/bar")
	if got != "hello-world-foo-bar" {
		t.Errorf("sanitizeFilename('hello world! foo/bar') = %q, want %q", got, "hello-world-foo-bar")
	}
}

func TestSanitizeFilename_AllSpecialChars(t *testing.T) {
	got := sanitizeFilename("!@#$%")
	if got != "" {
		t.Errorf("sanitizeFilename('!@#$%%') = %q, want empty string", got)
	}
}

func TestSanitizeFilename_Unicode(t *testing.T) {
	got := sanitizeFilename("cafe\u0301")
	// Non-ASCII letters are not in a-z/0-9, so they become hyphens.
	// "cafe" + combining accent -> "caf" + hyphen trimmed or "cafe-"
	// Actually: c, a, f, e are ASCII, then \u0301 is non-ASCII -> "cafe-" -> trimmed to "cafe"
	if got != "cafe" {
		t.Errorf("sanitizeFilename('cafe\\u0301') = %q, want %q", got, "cafe")
	}
}

func TestSanitizeFilename_NumbersPreserved(t *testing.T) {
	got := sanitizeFilename("item123")
	if got != "item123" {
		t.Errorf("sanitizeFilename('item123') = %q, want %q", got, "item123")
	}
}

// --- LoadCollection / SaveCollection roundtrip ---

func TestSaveAndLoadCollection(t *testing.T) {
	s := newTestStore(t)

	col := model.Collection{
		Name: "groceries",
		Items: []model.CollectionItem{
			{Text: "milk", Done: false},
			{Text: "eggs", Done: true},
			{Text: "bread", Done: false},
		},
	}

	if err := s.SaveCollection(col); err != nil {
		t.Fatalf("SaveCollection: %v", err)
	}

	loaded, err := s.LoadCollection("groceries")
	if err != nil {
		t.Fatalf("LoadCollection: %v", err)
	}

	if loaded.Name != col.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, col.Name)
	}
	if len(loaded.Items) != len(col.Items) {
		t.Fatalf("got %d items, want %d", len(loaded.Items), len(col.Items))
	}
	for i, item := range loaded.Items {
		if item.Text != col.Items[i].Text {
			t.Errorf("item[%d].Text = %q, want %q", i, item.Text, col.Items[i].Text)
		}
		if item.Done != col.Items[i].Done {
			t.Errorf("item[%d].Done = %v, want %v", i, item.Done, col.Items[i].Done)
		}
	}
}

func TestLoadCollection_NotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.LoadCollection("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing collection")
	}
}

// --- ListCollections ---

func TestListCollections_Multiple(t *testing.T) {
	s := newTestStore(t)

	names := []string{"alpha", "bravo", "charlie"}
	for _, name := range names {
		col := model.Collection{
			Name:  name,
			Items: []model.CollectionItem{{Text: "item", Done: false}},
		}
		if err := s.SaveCollection(col); err != nil {
			t.Fatalf("SaveCollection(%s): %v", name, err)
		}
	}

	listed, err := s.ListCollections()
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}

	if len(listed) != 3 {
		t.Fatalf("got %d collections, want 3", len(listed))
	}

	// Build a set from listed names to check all are present
	got := make(map[string]bool)
	for _, n := range listed {
		got[n] = true
	}
	for _, name := range names {
		if !got[name] {
			t.Errorf("missing collection %q in list", name)
		}
	}
}

func TestListCollections_EmptyDir(t *testing.T) {
	s := newTestStore(t)

	listed, err := s.ListCollections()
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("got %d collections, want 0", len(listed))
	}
}

func TestListCollections_IgnoresNonMD(t *testing.T) {
	s := newTestStore(t)

	// Create collections dir and put a non-.md file in it
	colDir := s.collectionsDir()
	if err := os.MkdirAll(colDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(colDir, "notes.txt"), []byte("not a collection"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Also save a real collection
	col := model.Collection{Name: "real", Items: []model.CollectionItem{{Text: "x", Done: false}}}
	if err := s.SaveCollection(col); err != nil {
		t.Fatal(err)
	}

	listed, err := s.ListCollections()
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(listed) != 1 {
		t.Errorf("got %d collections, want 1", len(listed))
	}
}

// --- DeleteCollection ---

func TestDeleteCollection(t *testing.T) {
	s := newTestStore(t)

	col := model.Collection{
		Name:  "tobedeleted",
		Items: []model.CollectionItem{{Text: "bye", Done: false}},
	}
	if err := s.SaveCollection(col); err != nil {
		t.Fatalf("SaveCollection: %v", err)
	}

	if err := s.DeleteCollection("tobedeleted"); err != nil {
		t.Fatalf("DeleteCollection: %v", err)
	}

	// Verify it's gone
	_, err := s.LoadCollection("tobedeleted")
	if err == nil {
		t.Error("expected error loading deleted collection")
	}
}

func TestDeleteCollection_NotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.DeleteCollection("ghost")
	if err == nil {
		t.Fatal("expected error deleting non-existent collection")
	}
}
