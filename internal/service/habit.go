package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/storage"
)

// HabitService encapsulates business logic for habit tracking.
type HabitService struct {
	store *storage.Store
}

// NewHabitService creates a HabitService.
func NewHabitService(store *storage.Store) *HabitService {
	return &HabitService{store: store}
}

// LoadMonth returns the habit tracker for a given month.
func (s *HabitService) LoadMonth(month time.Time) (*model.HabitTracker, error) {
	ht, err := s.store.LoadHabits(month)
	if err != nil {
		return nil, fmt.Errorf("load habits: %w", err)
	}
	return ht, nil
}

// AddHabit adds a new habit to the given month.
func (s *HabitService) AddHabit(month time.Time, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("habit name must not be empty")
	}

	ht, err := s.store.LoadHabits(month)
	if err != nil {
		return err
	}

	for _, h := range ht.Habits {
		if strings.EqualFold(h, name) {
			return fmt.Errorf("habit %q already exists", name)
		}
	}

	ht.Habits = append(ht.Habits, name)
	ht.Done[name] = make(map[int]bool)
	return s.store.SaveHabits(month, ht)
}

// RemoveHabit removes a habit from the given month.
func (s *HabitService) RemoveHabit(month time.Time, name string) error {
	ht, err := s.store.LoadHabits(month)
	if err != nil {
		return err
	}

	found := false
	for i, h := range ht.Habits {
		if h == name {
			ht.Habits = append(ht.Habits[:i], ht.Habits[i+1:]...)
			delete(ht.Done, name)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("habit %q not found", name)
	}

	return s.store.SaveHabits(month, ht)
}

// Toggle flips a habit's completion for a given day.
func (s *HabitService) Toggle(month time.Time, habit string, day int) error {
	ht, err := s.store.LoadHabits(month)
	if err != nil {
		return err
	}

	found := false
	for _, h := range ht.Habits {
		if h == habit {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("habit %q not found", habit)
	}

	ht.Toggle(habit, day)
	return s.store.SaveHabits(month, ht)
}
