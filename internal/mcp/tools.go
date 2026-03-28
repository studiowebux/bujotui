package mcp

import "encoding/json"

// ToolList returns all available MCP tools.
func ToolList() ToolsListResult {
	return ToolsListResult{
		Tools: []Tool{
			{
				Name:        "add_entry",
				Description: "Add a new journal entry for today.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"symbol":      {"type": "string", "description": "Entry type: task, event, note, idea, urgent, waiting, health"},
						"project":     {"type": "string", "description": "Project name (optional)"},
						"person":      {"type": "string", "description": "Assignee (optional)"},
						"description": {"type": "string", "description": "Entry description"}
					},
					"required": ["description"]
				}`),
			},
			{
				Name:        "list_entries",
				Description: "List journal entries for a specific date. Defaults to today.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"date": {"type": "string", "description": "Date in YYYY-MM-DD format (default: today)"}
					}
				}`),
			},
			{
				Name:        "edit_entry",
				Description: "Edit an existing journal entry by index (0-based).",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"date":        {"type": "string", "description": "Date in YYYY-MM-DD format (default: today)"},
						"index":       {"type": "integer", "description": "Entry index (0-based)"},
						"symbol":      {"type": "string", "description": "New entry type"},
						"state":       {"type": "string", "description": "New state (done, migrated, scheduled, cancelled, or empty)"},
						"project":     {"type": "string", "description": "New project name"},
						"person":      {"type": "string", "description": "New assignee"},
						"description": {"type": "string", "description": "New description"}
					},
					"required": ["index", "description"]
				}`),
			},
			{
				Name:        "transition_entry",
				Description: "Change the state of a journal entry (e.g., mark as done, migrated, scheduled, cancelled).",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"date":  {"type": "string", "description": "Date in YYYY-MM-DD format (default: today)"},
						"index": {"type": "integer", "description": "Entry index (0-based)"},
						"state": {"type": "string", "description": "Target state: done, migrated, scheduled, cancelled"}
					},
					"required": ["index", "state"]
				}`),
			},
			{
				Name:        "delete_entry",
				Description: "Delete a journal entry by index (0-based).",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"date":  {"type": "string", "description": "Date in YYYY-MM-DD format (default: today)"},
						"index": {"type": "integer", "description": "Entry index (0-based)"}
					},
					"required": ["index"]
				}`),
			},
			{
				Name:        "set_note",
				Description: "Set or update the daily note for a specific date.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"date": {"type": "string", "description": "Date in YYYY-MM-DD format (default: today)"},
						"note": {"type": "string", "description": "Note text (empty string to clear)"}
					},
					"required": ["note"]
				}`),
			},
			{
				Name:        "list_month",
				Description: "List all entries and notes for a month.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"month": {"type": "string", "description": "Month in YYYY-MM format (default: current month)"}
					}
				}`),
			},
			{
				Name:        "search",
				Description: "Search entries across all fields (description, project, person, symbol, state). Returns matching entries for a date range.",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"query":     {"type": "string", "description": "Search text (matches against all fields)"},
						"date_from": {"type": "string", "description": "Start date YYYY-MM-DD (default: first of current month)"},
						"date_to":   {"type": "string", "description": "End date YYYY-MM-DD (default: last of current month)"}
					},
					"required": ["query"]
				}`),
			},
		},
	}
}
