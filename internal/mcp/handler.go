package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/studiowebux/bujotui/internal/service"
)

// Handler dispatches MCP tool calls to the EntryService and CollectionService.
type Handler struct {
	svc    *service.EntryService
	colSvc *service.CollectionService
}

func NewHandler(svc *service.EntryService, colSvc *service.CollectionService) *Handler {
	return &Handler{svc: svc, colSvc: colSvc}
}

func (h *Handler) HandleToolCall(name string, args json.RawMessage) ToolResult {
	switch name {
	case "add_entry":
		return h.addEntry(args)
	case "list_entries":
		return h.listEntries(args)
	case "edit_entry":
		return h.editEntry(args)
	case "transition_entry":
		return h.transitionEntry(args)
	case "delete_entry":
		return h.deleteEntry(args)
	case "set_note":
		return h.setNote(args)
	case "list_month":
		return h.listMonth(args)
	case "search":
		return h.search(args)
	case "list_collections":
		return h.listCollections(args)
	case "get_collection":
		return h.getCollection(args)
	case "create_collection":
		return h.createCollection(args)
	case "delete_collection":
		return h.deleteCollection(args)
	case "add_collection_item":
		return h.addCollectionItem(args)
	case "remove_collection_item":
		return h.removeCollectionItem(args)
	case "toggle_collection_item":
		return h.toggleCollectionItem(args)
	default:
		return ErrorResult("unknown tool: %s", name)
	}
}

func (h *Handler) addEntry(args json.RawMessage) ToolResult {
	var p struct {
		Symbol      string `json:"symbol"`
		Project     string `json:"project"`
		Person      string `json:"person"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	entry, err := h.svc.AddEntry(p.Symbol, "", p.Project, p.Person, p.Description)
	if err != nil {
		return ErrorResult("%v", err)
	}

	return TextResult(fmt.Sprintf("Added: %s %s %s @%s %s",
		entry.Symbol.Name, entry.DateTime.Format("15:04"), entry.Project, entry.Person, entry.Description))
}

func (h *Handler) listEntries(args json.RawMessage) ToolResult {
	var p struct {
		Date string `json:"date"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	date, err := parseDate(p.Date)
	if err != nil {
		return ErrorResult("%v", err)
	}

	entries, err := h.svc.LoadDay(date)
	if err != nil {
		return ErrorResult("%v", err)
	}

	if len(entries) == 0 {
		return TextResult(fmt.Sprintf("No entries for %s", date.Format("2006-01-02")))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Entries for %s:\n\n", date.Format("2006-01-02"))
	for i, e := range entries {
		state := ""
		if e.State != "" {
			state = "[" + e.State + "] "
		}
		fmt.Fprintf(&b, "%d. %s%s %s @%s %s\n",
			i, state, e.Symbol.Name, e.Project, e.Person, e.Description)
	}
	return TextResult(b.String())
}

func (h *Handler) editEntry(args json.RawMessage) ToolResult {
	var p struct {
		Date        string `json:"date"`
		Index       int    `json:"index"`
		Symbol      string `json:"symbol"`
		State       string `json:"state"`
		Project     string `json:"project"`
		Person      string `json:"person"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	date, err := parseDate(p.Date)
	if err != nil {
		return ErrorResult("%v", err)
	}

	if err := h.svc.EditEntry(date, p.Index, p.Symbol, p.State, p.Project, p.Person, p.Description); err != nil {
		return ErrorResult("%v", err)
	}

	return TextResult(fmt.Sprintf("Updated entry %d on %s", p.Index, date.Format("2006-01-02")))
}

func (h *Handler) transitionEntry(args json.RawMessage) ToolResult {
	var p struct {
		Date  string `json:"date"`
		Index int    `json:"index"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	date, err := parseDate(p.Date)
	if err != nil {
		return ErrorResult("%v", err)
	}

	if err := h.svc.TransitionEntry(date, p.Index, p.State); err != nil {
		return ErrorResult("%v", err)
	}

	return TextResult(fmt.Sprintf("Entry %d on %s -> %s", p.Index, date.Format("2006-01-02"), p.State))
}

func (h *Handler) deleteEntry(args json.RawMessage) ToolResult {
	var p struct {
		Date  string `json:"date"`
		Index int    `json:"index"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	date, err := parseDate(p.Date)
	if err != nil {
		return ErrorResult("%v", err)
	}

	if err := h.svc.DeleteEntry(date, p.Index); err != nil {
		return ErrorResult("%v", err)
	}

	return TextResult(fmt.Sprintf("Deleted entry %d on %s", p.Index, date.Format("2006-01-02")))
}

func (h *Handler) setNote(args json.RawMessage) ToolResult {
	var p struct {
		Date string `json:"date"`
		Note string `json:"note"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	date, err := parseDate(p.Date)
	if err != nil {
		return ErrorResult("%v", err)
	}

	if err := h.svc.SaveNote(date, p.Note); err != nil {
		return ErrorResult("%v", err)
	}

	if p.Note == "" {
		return TextResult(fmt.Sprintf("Cleared note for %s", date.Format("2006-01-02")))
	}
	return TextResult(fmt.Sprintf("Note set for %s: %s", date.Format("2006-01-02"), p.Note))
}

func (h *Handler) listMonth(args json.RawMessage) ToolResult {
	var p struct {
		Month string `json:"month"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	var month time.Time
	if p.Month == "" {
		now := time.Now()
		month = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	} else {
		t, err := time.ParseInLocation("2006-01", p.Month, time.Local)
		if err != nil {
			return ErrorResult("invalid month format (expected YYYY-MM): %v", err)
		}
		month = t
	}

	entries, err := h.svc.LoadMonth(month)
	if err != nil {
		return ErrorResult("%v", err)
	}
	notes, err := h.svc.LoadMonthNotes(month)
	if err != nil {
		return ErrorResult("%v", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Month: %s\n\n", month.Format("January 2006"))

	daysInMonth := time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, month.Location()).Day()
	for day := 1; day <= daysInMonth; day++ {
		dayEntries := entries[day]
		note := notes[day]
		if len(dayEntries) == 0 && note == "" {
			continue
		}
		date := time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, month.Location())
		fmt.Fprintf(&b, "## %s\n", date.Format("2006-01-02 Mon"))
		if note != "" {
			fmt.Fprintf(&b, "Note: %s\n", note)
		}
		for i, e := range dayEntries {
			state := ""
			if e.State != "" {
				state = "[" + e.State + "] "
			}
			fmt.Fprintf(&b, "  %d. %s%s %s @%s %s\n",
				i, state, e.Symbol.Name, e.Project, e.Person, e.Description)
		}
		b.WriteByte('\n')
	}

	if b.Len() < 30 {
		return TextResult(fmt.Sprintf("No entries for %s", month.Format("January 2006")))
	}
	return TextResult(b.String())
}

func (h *Handler) search(args json.RawMessage) ToolResult {
	var p struct {
		Query    string `json:"query"`
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	if p.Query == "" {
		return ErrorResult("query is required")
	}

	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 1, -1)

	if p.DateFrom != "" {
		t, err := time.ParseInLocation("2006-01-02", p.DateFrom, time.Local)
		if err != nil {
			return ErrorResult("invalid date_from: %v", err)
		}
		from = t
	}
	if p.DateTo != "" {
		t, err := time.ParseInLocation("2006-01-02", p.DateTo, time.Local)
		if err != nil {
			return ErrorResult("invalid date_to: %v", err)
		}
		to = t
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Search: %q (%s to %s)\n\n", p.Query, from.Format("2006-01-02"), to.Format("2006-01-02"))

	found := 0
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		entries, err := h.svc.LoadDay(d)
		if err != nil {
			continue
		}
		matches := service.FilterEntries(entries, "", "", "", p.Query)
		if len(matches) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s\n", d.Format("2006-01-02 Mon"))
		for _, e := range matches {
			state := ""
			if e.State != "" {
				state = "[" + e.State + "] "
			}
			fmt.Fprintf(&b, "  %s%s %s @%s %s\n",
				state, e.Symbol.Name, e.Project, e.Person, e.Description)
			found++
		}
		b.WriteByte('\n')
	}

	if found == 0 {
		return TextResult(fmt.Sprintf("No entries matching %q", p.Query))
	}
	fmt.Fprintf(&b, "Found %d matching entries.", found)
	return TextResult(b.String())
}

func (h *Handler) listCollections(_ json.RawMessage) ToolResult {
	names, err := h.colSvc.List()
	if err != nil {
		return ErrorResult("%v", err)
	}
	if len(names) == 0 {
		return TextResult("No collections.")
	}
	var b strings.Builder
	b.WriteString("Collections:\n\n")
	for _, n := range names {
		fmt.Fprintf(&b, "- %s\n", n)
	}
	return TextResult(b.String())
}

func (h *Handler) getCollection(args json.RawMessage) ToolResult {
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}
	if p.Name == "" {
		return ErrorResult("name is required")
	}

	col, err := h.colSvc.Get(p.Name)
	if err != nil {
		return ErrorResult("%v", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", col.Name)
	for i, item := range col.Items {
		check := "[ ]"
		if item.Done {
			check = "[x]"
		}
		fmt.Fprintf(&b, "%d. %s %s\n", i, check, item.Text)
	}
	if len(col.Items) == 0 {
		b.WriteString("(empty)\n")
	}
	return TextResult(b.String())
}

func (h *Handler) createCollection(args json.RawMessage) ToolResult {
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	col, err := h.colSvc.Create(p.Name)
	if err != nil {
		return ErrorResult("%v", err)
	}
	return TextResult(fmt.Sprintf("Created collection: %s", col.Name))
}

func (h *Handler) deleteCollection(args json.RawMessage) ToolResult {
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	if err := h.colSvc.Delete(p.Name); err != nil {
		return ErrorResult("%v", err)
	}
	return TextResult(fmt.Sprintf("Deleted collection: %s", p.Name))
}

func (h *Handler) addCollectionItem(args json.RawMessage) ToolResult {
	var p struct {
		Name string `json:"name"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	if err := h.colSvc.AddItem(p.Name, p.Text); err != nil {
		return ErrorResult("%v", err)
	}
	return TextResult(fmt.Sprintf("Added item to %s: %s", p.Name, p.Text))
}

func (h *Handler) removeCollectionItem(args json.RawMessage) ToolResult {
	var p struct {
		Name  string `json:"name"`
		Index int    `json:"index"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	if err := h.colSvc.RemoveItem(p.Name, p.Index); err != nil {
		return ErrorResult("%v", err)
	}
	return TextResult(fmt.Sprintf("Removed item %d from %s", p.Index, p.Name))
}

func (h *Handler) toggleCollectionItem(args json.RawMessage) ToolResult {
	var p struct {
		Name  string `json:"name"`
		Index int    `json:"index"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return ErrorResult("invalid arguments: %v", err)
	}

	if err := h.colSvc.ToggleItem(p.Name, p.Index); err != nil {
		return ErrorResult("%v", err)
	}
	return TextResult(fmt.Sprintf("Toggled item %d in %s", p.Index, p.Name))
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location()), nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (expected YYYY-MM-DD): %w", s, err)
	}
	return t.Add(12 * time.Hour), nil // noon to avoid timezone edge cases
}
