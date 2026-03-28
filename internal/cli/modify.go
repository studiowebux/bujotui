package cli

import (
	"fmt"
	"io"

	"github.com/studiowebux/bujotui/internal/storage"
)

func cmdModify(args []string, targetSymbol string, store *storage.Store, stdout io.Writer) error {
	today, idx, entries, err := parseEntryIndex(args, store)
	if err != nil {
		return fmt.Errorf("bujotui %s: %w", targetSymbol, err)
	}

	entry := entries[idx]
	if _, ok := store.Config.Symbols.LookupByName(targetSymbol); !ok {
		return fmt.Errorf("bujotui %s: unknown state %q", targetSymbol, targetSymbol)
	}

	if !store.Config.Symbols.CanTransition(entry.Symbol.Name, targetSymbol) {
		valid := store.Config.Symbols.ValidTransitions(entry.Symbol.Name)
		var names []string
		for _, v := range valid {
			names = append(names, v.Name)
		}
		return fmt.Errorf("bujotui %s: cannot transition %q to %q (valid: %v)",
			targetSymbol, entry.Symbol.Name, targetSymbol, names)
	}

	entry.State = targetSymbol
	if err := store.UpdateEntry(today, idx, entry); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%-12s %-10s %s  %-14s %-12s %s\n",
		targetSymbol, entry.Symbol.Name, entry.DateTime.Format("15:04"), entry.Project, "@"+entry.Person, entry.Description)
	return nil
}

func cmdRemove(args []string, store *storage.Store, stdout io.Writer) error {
	today, idx, entries, err := parseEntryIndex(args, store)
	if err != nil {
		return fmt.Errorf("bujotui remove: %w", err)
	}

	entry := entries[idx]
	if err := store.RemoveEntry(today, idx); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "removed: %s %s @%s %s\n",
		entry.Symbol.Name, entry.Project, entry.Person, entry.Description)
	return nil
}
