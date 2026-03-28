package model

// FutureEntry is a planned item for a future month, not tied to a specific date.
type FutureEntry struct {
	Symbol      Symbol // entry type
	Description string
}

// FutureMonth groups entries under a year-month.
type FutureMonth struct {
	Year    int
	Month   int // 1-12
	Entries []FutureEntry
}
