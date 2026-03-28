package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/studiowebux/bujotui/internal/storage"
)

func printHelp(w io.Writer) {
	fmt.Fprint(w, `bujotui - bullet journal CLI

Usage:
  bujotui                          Launch TUI
  bujotui add [flags] "desc"       Add entry (now)
  bujotui list [flags]             List entries (today)
  bujotui done <n>                 Mark entry as done
  bujotui migrate <n>              Mark entry as migrated
  bujotui schedule <n>             Mark entry as scheduled
  bujotui cancel <n>               Mark entry as cancelled
  bujotui remove <n>               Remove entry
  bujotui projects                 List known projects
  bujotui people                   List known people
  bujotui config                   Show configuration
  bujotui config init              Create default config
  bujotui version                  Show version
  bujotui help                     Show this help

Add flags:
  -s symbol     Symbol name (default: from config)
  -p project    Project name (default: from config)
  -a person     Person name (default: from config)
  -d datetime   Date/time as YYYY-MM-DDThh:mm (default: now)

List flags:
  --project X   Filter by project
  --person X    Filter by person
  --symbol X    Filter by symbol name
  --date X      Specific date (YYYY-MM-DD)
  --week        Current week
  --month       Current month
  --time        Show entry timestamps

Global flags:
  --dir PATH    Override config and data directory

Shell completions:
  bujotui completion bash    Output bash completion script
  bujotui completion zsh     Output zsh completion script
`)
}

func cmdProjects(store *storage.Store, stdout io.Writer) error {
	now := time.Now()
	days, err := store.LoadMonth(now)
	if err != nil {
		return err
	}

	projects := store.AllProjects(days)

	// Merge with config projects
	seen := make(map[string]bool)
	for _, p := range projects {
		seen[p] = true
	}
	for _, p := range store.Config.Projects {
		if !seen[p] {
			projects = append(projects, p)
		}
	}

	if len(projects) == 0 {
		fmt.Fprintln(stdout, "No projects found.")
		return nil
	}

	for _, p := range projects {
		fmt.Fprintln(stdout, p)
	}
	return nil
}

func cmdPeople(store *storage.Store, stdout io.Writer) error {
	now := time.Now()
	days, err := store.LoadMonth(now)
	if err != nil {
		return err
	}

	people := store.AllPeople(days)

	seen := make(map[string]bool)
	for _, p := range people {
		seen[p] = true
	}
	for _, p := range store.Config.People {
		if !seen[p] {
			people = append(people, p)
		}
	}

	if len(people) == 0 {
		fmt.Fprintln(stdout, "No people found.")
		return nil
	}

	for _, p := range people {
		fmt.Fprintln(stdout, p)
	}
	return nil
}
