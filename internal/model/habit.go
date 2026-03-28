package model

// HabitTracker holds the habit definitions and completion grid for a month.
type HabitTracker struct {
	Habits []string       // habit names in order
	Done   map[string]map[int]bool // habit name -> day number -> done
}

// NewHabitTracker creates an empty tracker.
func NewHabitTracker() *HabitTracker {
	return &HabitTracker{
		Done: make(map[string]map[int]bool),
	}
}

// Toggle flips the done state for a habit on a given day.
func (ht *HabitTracker) Toggle(habit string, day int) {
	if ht.Done[habit] == nil {
		ht.Done[habit] = make(map[int]bool)
	}
	ht.Done[habit][day] = !ht.Done[habit][day]
}

// IsDone returns whether a habit is done on a given day.
func (ht *HabitTracker) IsDone(habit string, day int) bool {
	if m, ok := ht.Done[habit]; ok {
		return m[day]
	}
	return false
}

// Streak returns the current streak count for a habit ending at the given day.
func (ht *HabitTracker) Streak(habit string, maxDay int) int {
	count := 0
	for d := maxDay; d >= 1; d-- {
		if ht.IsDone(habit, d) {
			count++
		} else {
			break
		}
	}
	return count
}
