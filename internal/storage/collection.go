package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/studiowebux/bujotui/internal/model"
)

// collectionsDir returns the path to the collections directory.
func (s *Store) collectionsDir() string {
	return filepath.Join(s.Dir, "collections")
}

// ensureCollectionsDir creates the collections directory if it doesn't exist.
func (s *Store) ensureCollectionsDir() error {
	return os.MkdirAll(s.collectionsDir(), 0o750)
}

// collectionFile returns the path for a collection's markdown file.
func (s *Store) collectionFile(name string) string {
	return filepath.Join(s.collectionsDir(), sanitizeFilename(name)+".md")
}

// ListCollections returns the names of all collections.
func (s *Store) ListCollections() ([]string, error) {
	dir := s.collectionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read collections dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// Read the title from the file (first # heading)
		name, err := s.readCollectionName(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// readCollectionName reads the first heading from a collection file.
func (s *Store) readCollectionName(path string) (string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:]), nil
		}
	}
	return "", fmt.Errorf("no heading found")
}

// LoadCollection reads a collection from its markdown file.
func (s *Store) LoadCollection(name string) (model.Collection, error) {
	path := s.collectionFile(name)
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return model.Collection{}, fmt.Errorf("collection %q not found", name)
		}
		return model.Collection{}, fmt.Errorf("open collection: %w", err)
	}
	defer f.Close()

	return parseCollection(f)
}

// parseCollection parses a collection markdown file.
// Format:
//
//	# Collection Name
//
//	- [ ] unchecked item
//	- [x] checked item
func parseCollection(f *os.File) (model.Collection, error) {
	scanner := bufio.NewScanner(f)
	var col model.Collection

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Title
		if strings.HasPrefix(trimmed, "# ") {
			col.Name = strings.TrimSpace(trimmed[2:])
			continue
		}

		// Checked item: - [x] text
		if strings.HasPrefix(trimmed, "- [x] ") {
			col.Items = append(col.Items, model.CollectionItem{
				Text: trimmed[6:],
				Done: true,
			})
			continue
		}

		// Unchecked item: - [ ] text
		if strings.HasPrefix(trimmed, "- [ ] ") {
			col.Items = append(col.Items, model.CollectionItem{
				Text: trimmed[6:],
				Done: false,
			})
			continue
		}

		// Plain list item: - text (treated as unchecked)
		if strings.HasPrefix(trimmed, "- ") {
			col.Items = append(col.Items, model.CollectionItem{
				Text: trimmed[2:],
				Done: false,
			})
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return model.Collection{}, fmt.Errorf("parse collection: %w", err)
	}

	return col, nil
}

// SaveCollection writes a collection to its markdown file atomically.
func (s *Store) SaveCollection(col model.Collection) error {
	if err := s.ensureCollectionsDir(); err != nil {
		return err
	}

	path := s.collectionFile(col.Name)
	data := formatCollection(col)
	return AtomicWriteFile(path, data, 0o644)
}

// formatCollection renders a collection to markdown bytes.
func formatCollection(col model.Collection) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", col.Name)
	for _, item := range col.Items {
		if item.Done {
			fmt.Fprintf(&b, "- [x] %s\n", item.Text)
		} else {
			fmt.Fprintf(&b, "- [ ] %s\n", item.Text)
		}
	}
	return []byte(b.String())
}

// DeleteCollection removes a collection's markdown file.
func (s *Store) DeleteCollection(name string) error {
	path := s.collectionFile(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("collection %q not found", name)
		}
		return fmt.Errorf("delete collection: %w", err)
	}
	return nil
}

// sanitizeFilename converts a collection name to a safe filename.
// Lowercases and replaces non-alphanumeric characters with hyphens.
func sanitizeFilename(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	prev := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prev = false
		} else if !prev {
			b.WriteByte('-')
			prev = true
		}
	}
	return strings.Trim(b.String(), "-")
}
