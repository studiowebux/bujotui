package complete

import (
	"strings"

	"github.com/studiowebux/bujotui/internal/model"
)

// Completer provides autocomplete suggestions for symbols, projects, and people.
type Completer struct {
	Symbols  []string // entry type names (not states) from config
	Projects []string // from config + discovered
	People   []string // from config + discovered
}

// New creates a Completer from config lists.
func New(symbolNames, projects, people []string) *Completer {
	return &Completer{
		Symbols:  symbolNames,
		Projects: dedupe(projects),
		People:   dedupe(people),
	}
}

// DiscoverFromEntries adds projects and people found in entries.
func (c *Completer) DiscoverFromEntries(entries []model.Entry) {
	for _, e := range entries {
		c.Projects = addUnique(c.Projects, e.Project)
		c.People = addUnique(c.People, e.Person)
	}
}

// CompleteSymbol returns symbol names matching the prefix.
func (c *Completer) CompleteSymbol(prefix string) []string {
	return matchPrefix(c.Symbols, prefix)
}

// CompleteProject returns project names matching the prefix.
func (c *Completer) CompleteProject(prefix string) []string {
	return matchPrefix(c.Projects, prefix)
}

// CompletePerson returns person names matching the prefix.
func (c *Completer) CompletePerson(prefix string) []string {
	return matchPrefix(c.People, prefix)
}

func matchPrefix(items []string, prefix string) []string {
	if prefix == "" {
		return items
	}
	lower := strings.ToLower(prefix)
	var matches []string
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item), lower) {
			matches = append(matches, item)
		}
	}
	return matches
}

func addUnique(slice []string, val string) []string {
	if val == "" {
		return slice
	}
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func dedupe(items []string) []string {
	seen := make(map[string]bool, len(items))
	var result []string
	for _, item := range items {
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
