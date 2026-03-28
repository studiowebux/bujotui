package cli

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/studiowebux/bujotui/internal/storage"
)

func cmdList(args []string, store *storage.Store, stdout io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	project := fs.String("project", "", "filter by project")
	person := fs.String("person", "", "filter by person")
	symbol := fs.String("symbol", "", "filter by symbol name")
	dateStr := fs.String("date", "", "specific date (YYYY-MM-DD)")
	week := fs.Bool("week", false, "show current week")
	month := fs.Bool("month", false, "show current month")
	showTime := fs.Bool("time", false, "show entry timestamps")
	if err := fs.Parse(args); err != nil {
		return err
	}

	now := time.Now()
	var from, to time.Time

	switch {
	case *dateStr != "":
		d, err := time.Parse("2006-01-02", *dateStr)
		if err != nil {
			return fmt.Errorf("bujotui list: invalid date %q (expected YYYY-MM-DD): %w", *dateStr, err)
		}
		from = d
		to = d
	case *week:
		// Start from Monday
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		from = now.AddDate(0, 0, -int(weekday-time.Monday))
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 0, 6)
	case *month:
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 1, -1)
	default:
		// Today
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		to = from
	}

	opts := storage.FilterOpts{
		DateFrom: from,
		DateTo:   to,
		Project:  *project,
		Person:   *person,
		Symbol:   *symbol,
	}

	entries, err := store.Filter(opts)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Fprintln(stdout, "No entries found.")
		return nil
	}

	for i, e := range entries {
		stateCol := fmt.Sprintf("%-12s", e.State)
		symCol := fmt.Sprintf("%-10s", e.Symbol.Name)

		timeCol := ""
		if *showTime {
			timeCol = fmt.Sprintf("%-6s", e.DateTime.Format("15:04"))
		}

		fmt.Fprintf(stdout, "%3d  %s%s%s%-14s %-12s %s\n",
			i+1, stateCol, symCol, timeCol, e.Project, "@"+e.Person, e.Description)
	}

	return nil
}
