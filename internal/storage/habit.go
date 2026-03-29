package storage

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/studiowebux/bujotui/internal/model"
)

// habitsDir returns the path to the habits directory.
func (s *Store) habitsDir() string {
	return filepath.Join(s.Dir, "habits")
}

// ensureHabitsDir creates the habits directory if needed.
func (s *Store) ensureHabitsDir() error {
	return os.MkdirAll(s.habitsDir(), 0o700)
}

// habitFile returns the path for a month's habit file.
func (s *Store) habitFile(t time.Time) string {
	return filepath.Join(s.habitsDir(), t.Format("2006-01")+".md")
}

// LoadHabits reads the habit tracker for a given month.
// Format:
//
//	# Habits 2026-03
//
//	## Exercise
//	1,3,5,7,10
//
//	## Read
//	1,2,3,4,5,6,7
func (s *Store) LoadHabits(t time.Time) (*model.HabitTracker, error) {
	path := s.habitFile(t)
	data, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.NewHabitTracker(), nil
		}
		return nil, fmt.Errorf("load habits file: %w", err)
	}

	ht := model.NewHabitTracker()
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var currentHabit string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Habit heading: ## Name
		if strings.HasPrefix(line, "## ") {
			currentHabit = strings.TrimSpace(line[3:])
			ht.Habits = append(ht.Habits, currentHabit)
			ht.Done[currentHabit] = make(map[int]bool)
			continue
		}

		// Day list: 1,3,5,7
		if currentHabit != "" && line != "" && !strings.HasPrefix(line, "#") {
			for _, tok := range strings.Split(line, ",") {
				tok = strings.TrimSpace(tok)
				if d, err := strconv.Atoi(tok); err == nil && d >= 1 && d <= 31 {
					ht.Done[currentHabit][d] = true
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse habits: %w", err)
	}

	return ht, nil
}

// SaveHabits writes the habit tracker for a given month atomically.
func (s *Store) SaveHabits(t time.Time, ht *model.HabitTracker) error {
	if err := s.ensureHabitsDir(); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Habits %s\n", t.Format("2006-01"))

	for _, habit := range ht.Habits {
		fmt.Fprintf(&b, "\n## %s\n", habit)
		days := ht.Done[habit]
		var nums []string
		for d := 1; d <= 31; d++ {
			if days[d] {
				nums = append(nums, strconv.Itoa(d))
			}
		}
		if len(nums) > 0 {
			b.WriteString(strings.Join(nums, ","))
			b.WriteByte('\n')
		}
	}

	path := s.habitFile(t)
	return s.writeFile(path, []byte(b.String()))
}
