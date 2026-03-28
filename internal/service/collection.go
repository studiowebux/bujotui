package service

import (
	"fmt"
	"strings"

	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/storage"
)

// hasAlphanumeric returns true if s contains at least one letter or digit.
func hasAlphanumeric(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return true
		}
	}
	return false
}

// CollectionService encapsulates business logic for collections.
type CollectionService struct {
	store *storage.Store
}

// NewCollectionService creates a CollectionService.
func NewCollectionService(store *storage.Store) *CollectionService {
	return &CollectionService{store: store}
}

// List returns all collection names.
func (s *CollectionService) List() ([]string, error) {
	names, err := s.store.ListCollections()
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	return names, nil
}

// Get loads a collection by name.
func (s *CollectionService) Get(name string) (model.Collection, error) {
	col, err := s.store.LoadCollection(name)
	if err != nil {
		return model.Collection{}, fmt.Errorf("get collection: %w", err)
	}
	return col, nil
}

// Create creates a new empty collection.
func (s *CollectionService) Create(name string) (model.Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Collection{}, fmt.Errorf("collection name must not be empty")
	}
	if len(name) > 100 {
		return model.Collection{}, fmt.Errorf("collection name too long (max 100 characters)")
	}
	if !hasAlphanumeric(name) {
		return model.Collection{}, fmt.Errorf("collection name must contain at least one letter or digit")
	}

	// Check if already exists
	if _, err := s.store.LoadCollection(name); err == nil {
		return model.Collection{}, fmt.Errorf("collection %q already exists", name)
	}

	col := model.Collection{Name: name}
	if err := s.store.SaveCollection(col); err != nil {
		return model.Collection{}, fmt.Errorf("create collection: %w", err)
	}
	return col, nil
}

// Delete removes a collection.
func (s *CollectionService) Delete(name string) error {
	if err := s.store.DeleteCollection(name); err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}
	return nil
}

// AddItem appends an item to a collection.
func (s *CollectionService) AddItem(name, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("item text must not be empty")
	}

	col, err := s.store.LoadCollection(name)
	if err != nil {
		return err
	}

	col.Items = append(col.Items, model.CollectionItem{Text: text})
	return s.store.SaveCollection(col)
}

// RemoveItem removes an item by index from a collection.
func (s *CollectionService) RemoveItem(name string, index int) error {
	col, err := s.store.LoadCollection(name)
	if err != nil {
		return err
	}

	if err := checkIndex(index, len(col.Items)); err != nil {
		return err
	}

	col.Items = append(col.Items[:index], col.Items[index+1:]...)
	return s.store.SaveCollection(col)
}

// ToggleItem toggles the done state of an item.
func (s *CollectionService) ToggleItem(name string, index int) error {
	col, err := s.store.LoadCollection(name)
	if err != nil {
		return err
	}

	if err := checkIndex(index, len(col.Items)); err != nil {
		return err
	}

	col.Items[index].Done = !col.Items[index].Done
	return s.store.SaveCollection(col)
}

// EditItem updates the text of an item.
func (s *CollectionService) EditItem(name string, index int, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("item text must not be empty")
	}

	col, err := s.store.LoadCollection(name)
	if err != nil {
		return err
	}

	if err := checkIndex(index, len(col.Items)); err != nil {
		return err
	}

	col.Items[index].Text = text
	return s.store.SaveCollection(col)
}

// MoveItem moves an item from one index to another.
func (s *CollectionService) MoveItem(name string, from, to int) error {
	col, err := s.store.LoadCollection(name)
	if err != nil {
		return err
	}

	if err := checkIndex(from, len(col.Items)); err != nil {
		return err
	}
	if err := checkIndex(to, len(col.Items)); err != nil {
		return err
	}

	item := col.Items[from]
	col.Items = append(col.Items[:from], col.Items[from+1:]...)

	// Adjust target index since the slice shrank by one
	if from < to {
		to--
	}

	// Insert at target position
	result := make([]model.CollectionItem, 0, len(col.Items)+1)
	result = append(result, col.Items[:to]...)
	result = append(result, item)
	result = append(result, col.Items[to:]...)
	col.Items = result

	return s.store.SaveCollection(col)
}
