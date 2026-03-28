package model

// Collection is a persistent, topic-based page — not tied to dates.
// Examples: "Books to Read", "Trip Planning", "Project Ideas".
type Collection struct {
	Name  string           // unique identifier / title
	Items []CollectionItem // ordered list of items
}

// CollectionItem is a single entry within a collection.
type CollectionItem struct {
	Text string // free-text content
	Done bool   // checked off or not
}
