package service

import (
	"fmt"
	"strings"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/storage"
)

// FutureLogService encapsulates business logic for the future log.
type FutureLogService struct {
	store *storage.Store
	cfg   *config.Config
}

// NewFutureLogService creates a FutureLogService.
func NewFutureLogService(store *storage.Store, cfg *config.Config) *FutureLogService {
	return &FutureLogService{store: store, cfg: cfg}
}

// LoadYear returns all future months for a given year.
func (s *FutureLogService) LoadYear(year int) ([]model.FutureMonth, error) {
	months, err := s.store.LoadFuture(year)
	if err != nil {
		return nil, fmt.Errorf("load future log: %w", err)
	}
	return months, nil
}

// AddEntry adds an entry to a specific month in the future log.
func (s *FutureLogService) AddEntry(year, month int, symbolName, description string) error {
	description = strings.TrimSpace(description)
	if description == "" {
		return fmt.Errorf("description must not be empty")
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("month must be 1-12")
	}

	symName := symbolName
	if symName == "" {
		symName = s.cfg.Defaults.Symbol
	}
	sym, ok := s.cfg.Symbols.LookupByName(symName)
	if !ok {
		return fmt.Errorf("unknown symbol %q", symName)
	}

	months, err := s.store.LoadFuture(year)
	if err != nil {
		return err
	}

	// Find or create the month
	found := false
	for i, m := range months {
		if m.Month == month {
			months[i].Entries = append(months[i].Entries, model.FutureEntry{
				Symbol:      sym,
				Description: description,
			})
			found = true
			break
		}
	}
	if !found {
		fm := model.FutureMonth{
			Year:  year,
			Month: month,
			Entries: []model.FutureEntry{
				{Symbol: sym, Description: description},
			},
		}
		months = insertMonthSorted(months, fm)
	}

	return s.store.SaveFuture(year, months)
}

// RemoveEntry removes an entry by index from a specific month.
func (s *FutureLogService) RemoveEntry(year, month, index int) error {
	if month < 1 || month > 12 {
		return fmt.Errorf("month must be 1-12")
	}

	months, err := s.store.LoadFuture(year)
	if err != nil {
		return err
	}

	for i, m := range months {
		if m.Month == month {
			if err := checkIndex(index, len(m.Entries)); err != nil {
				return err
			}
			months[i].Entries = append(m.Entries[:index], m.Entries[index+1:]...)
			return s.store.SaveFuture(year, months)
		}
	}

	return fmt.Errorf("no entries for month %d", month)
}

// insertMonthSorted inserts a FutureMonth in chronological order.
func insertMonthSorted(months []model.FutureMonth, fm model.FutureMonth) []model.FutureMonth {
	for i, m := range months {
		if fm.Month < m.Month {
			result := make([]model.FutureMonth, 0, len(months)+1)
			result = append(result, months[:i]...)
			result = append(result, fm)
			result = append(result, months[i:]...)
			return result
		}
	}
	return append(months, fm)
}
