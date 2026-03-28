package model

import "testing"

func TestNewHabitTracker(t *testing.T) {
	ht := NewHabitTracker()
	if ht.Done == nil {
		t.Fatal("expected Done map to be initialized")
	}
	if len(ht.Habits) != 0 {
		t.Errorf("expected empty Habits slice")
	}
}

func TestToggle(t *testing.T) {
	ht := NewHabitTracker()

	// First toggle: false -> true
	ht.Toggle("exercise", 1)
	if !ht.Done["exercise"][1] {
		t.Error("expected exercise day 1 to be true after first toggle")
	}

	// Second toggle: true -> false
	ht.Toggle("exercise", 1)
	if ht.Done["exercise"][1] {
		t.Error("expected exercise day 1 to be false after second toggle")
	}

	// Third toggle: false -> true again
	ht.Toggle("exercise", 1)
	if !ht.Done["exercise"][1] {
		t.Error("expected exercise day 1 to be true after third toggle")
	}
}

func TestToggleMultipleHabitsAndDays(t *testing.T) {
	ht := NewHabitTracker()

	ht.Toggle("read", 5)
	ht.Toggle("meditate", 5)
	ht.Toggle("read", 10)

	if !ht.Done["read"][5] {
		t.Error("read day 5 should be true")
	}
	if !ht.Done["meditate"][5] {
		t.Error("meditate day 5 should be true")
	}
	if !ht.Done["read"][10] {
		t.Error("read day 10 should be true")
	}
	// Unset days should default to false.
	if ht.Done["read"][1] {
		t.Error("read day 1 should be false (never toggled)")
	}
}

func TestIsDone(t *testing.T) {
	ht := NewHabitTracker()
	ht.Toggle("exercise", 3)

	tests := []struct {
		habit string
		day   int
		want  bool
	}{
		{"exercise", 3, true},
		{"exercise", 4, false},   // day not toggled
		{"unknown", 3, false},    // habit never used
	}
	for _, tc := range tests {
		got := ht.IsDone(tc.habit, tc.day)
		if got != tc.want {
			t.Errorf("IsDone(%q, %d) = %v, want %v", tc.habit, tc.day, got, tc.want)
		}
	}
}

func TestStreak(t *testing.T) {
	tests := []struct {
		name       string
		doneDays   []int
		maxDay     int
		wantStreak int
	}{
		{
			name:       "no completions",
			doneDays:   nil,
			maxDay:     10,
			wantStreak: 0,
		},
		{
			name:       "streak from day 1",
			doneDays:   []int{1, 2, 3, 4, 5},
			maxDay:     5,
			wantStreak: 5,
		},
		{
			name:       "single day streak",
			doneDays:   []int{7},
			maxDay:     7,
			wantStreak: 1,
		},
		{
			name:       "gap in middle breaks streak",
			doneDays:   []int{1, 2, 3, 5, 6, 7},
			maxDay:     7,
			wantStreak: 3, // days 5, 6, 7
		},
		{
			name:       "maxDay not done means zero streak",
			doneDays:   []int{1, 2, 3},
			maxDay:     4,
			wantStreak: 0,
		},
		{
			name:       "streak ending before maxDay",
			doneDays:   []int{1, 2, 3, 8, 9, 10},
			maxDay:     10,
			wantStreak: 3, // days 8, 9, 10
		},
		{
			name:       "maxDay is 1 and done",
			doneDays:   []int{1},
			maxDay:     1,
			wantStreak: 1,
		},
		{
			name:       "maxDay is 1 and not done",
			doneDays:   nil,
			maxDay:     1,
			wantStreak: 0,
		},
		{
			name:       "all 31 days done",
			doneDays:   func() []int { d := make([]int, 31); for i := range d { d[i] = i + 1 }; return d }(),
			maxDay:     31,
			wantStreak: 31,
		},
		{
			name:       "gap at day 1 only",
			doneDays:   []int{2, 3, 4, 5},
			maxDay:     5,
			wantStreak: 4, // days 2-5
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ht := NewHabitTracker()
			for _, d := range tc.doneDays {
				ht.Toggle("habit", d)
			}
			got := ht.Streak("habit", tc.maxDay)
			if got != tc.wantStreak {
				t.Errorf("Streak(%q, %d) = %d, want %d", "habit", tc.maxDay, got, tc.wantStreak)
			}
		})
	}
}

func TestStreakUnknownHabit(t *testing.T) {
	ht := NewHabitTracker()
	if got := ht.Streak("nonexistent", 15); got != 0 {
		t.Errorf("Streak for unknown habit should be 0, got %d", got)
	}
}
