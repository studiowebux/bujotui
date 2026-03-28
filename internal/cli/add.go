package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/studiowebux/bujotui/internal/model"
	"github.com/studiowebux/bujotui/internal/storage"
)

func cmdAdd(args []string, store *storage.Store, stdout io.Writer) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	symbolName := fs.String("s", store.Config.Defaults.Symbol, "symbol name")
	project := fs.String("p", store.Config.Defaults.Project, "project name")
	person := fs.String("a", store.Config.Defaults.Person, "person name (@)")
	dateStr := fs.String("d", "", "date/time (YYYY-MM-DDThh:mm), default now")
	if err := fs.Parse(args); err != nil {
		return err
	}

	desc := strings.Join(fs.Args(), " ")
	if desc == "" {
		return fmt.Errorf("bujotui add: description required (usage: bujotui add [-s symbol] [-p project] [-a person] description)")
	}

	sym, ok := store.Config.Symbols.LookupByName(*symbolName)
	if !ok {
		return fmt.Errorf("bujotui add: unknown symbol %q (available: %s)", *symbolName, strings.Join(store.Config.Symbols.Names(), ", "))
	}

	var dt time.Time
	if *dateStr != "" {
		var err error
		dt, err = time.ParseInLocation("2006-01-02T15:04", *dateStr, time.Local)
		if err != nil {
			return fmt.Errorf("bujotui add: invalid datetime %q (expected YYYY-MM-DDThh:mm): %w", *dateStr, err)
		}
	} else {
		dt = time.Now()
	}

	entry := model.Entry{
		Symbol:      sym,
		Project:     *project,
		Person:      *person,
		Description: desc,
		DateTime:    dt,
	}

	if err := store.AddEntry(entry); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "%-10s %s  %-14s %-12s %s\n", sym.Name, dt.Format("15:04"), *project, "@"+*person, desc)
	return nil
}
