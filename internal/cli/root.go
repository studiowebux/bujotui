package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/studiowebux/bujotui/internal/config"
	"github.com/studiowebux/bujotui/internal/service"
	"github.com/studiowebux/bujotui/internal/storage"
	"github.com/studiowebux/bujotui/internal/tui"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// Run dispatches CLI subcommands. Returns an exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	configDir := config.DefaultConfigDir()
	dataDir := config.DefaultDataDir()

	// Legacy: BUJOTUI_DIR sets both config and data dir
	if dir := os.Getenv("BUJOTUI_DIR"); dir != "" {
		configDir = dir
		dataDir = dir
	}

	// Check for --dir global flag (overrides both)
	for i, arg := range args {
		if arg == "--dir" && i+1 < len(args) {
			configDir = args[i+1]
			dataDir = args[i+1]
			args = append(args[:i], args[i+2:]...)
			break
		}
	}

	if len(args) == 0 {
		return runTUI(configDir, dataDir, stdout, stderr)
	}

	switch args[0] {
	case "config":
		return runConfig(args[1:], configDir, dataDir, stdout, stderr)
	case "add":
		return runWithStore(configDir, dataDir, stderr, func(s *storage.Store) error {
			return cmdAdd(args[1:], s, stdout)
		})
	case "list", "ls":
		return runWithStore(configDir, dataDir, stderr, func(s *storage.Store) error {
			return cmdList(args[1:], s, stdout)
		})
	case "done":
		return runWithStore(configDir, dataDir, stderr, func(s *storage.Store) error {
			return cmdModify(args[1:], "done", s, stdout)
		})
	case "migrate":
		return runWithStore(configDir, dataDir, stderr, func(s *storage.Store) error {
			return cmdModify(args[1:], "migrated", s, stdout)
		})
	case "schedule":
		return runWithStore(configDir, dataDir, stderr, func(s *storage.Store) error {
			return cmdModify(args[1:], "scheduled", s, stdout)
		})
	case "cancel":
		return runWithStore(configDir, dataDir, stderr, func(s *storage.Store) error {
			return cmdModify(args[1:], "cancelled", s, stdout)
		})
	case "remove", "rm":
		return runWithStore(configDir, dataDir, stderr, func(s *storage.Store) error {
			return cmdRemove(args[1:], s, stdout)
		})
	case "projects":
		return runWithStore(configDir, dataDir, stderr, func(s *storage.Store) error {
			return cmdProjects(s, stdout)
		})
	case "people":
		return runWithStore(configDir, dataDir, stderr, func(s *storage.Store) error {
			return cmdPeople(s, stdout)
		})
	case "version":
		fmt.Fprintf(stdout, "bujotui %s\n", Version)
		return 0
	case "completion":
		return cmdCompletion(args[1:], stdout)
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printHelp(stderr)
		return 1
	}
}

func runWithStore(configDir, dataDir string, stderr io.Writer, fn func(*storage.Store) error) int {
	cfg, err := config.Load(configDir, dataDir)
	if err != nil {
		fmt.Fprintf(stderr, "error loading config: %v\n", err)
		return 1
	}
	store, err := storage.NewStore(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "error initializing store: %v\n", err)
		return 1
	}
	if err := fn(store); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func runConfig(args []string, configDir, dataDir string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		cfg, err := config.Load(configDir, dataDir)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Config dir: %s\n", cfg.Dir)
		fmt.Fprintf(stdout, "Data dir:   %s\n\n", cfg.DataDir)
		fmt.Fprintln(stdout, "Symbols (entry types):")
		for _, name := range cfg.Symbols.SymbolNames() {
			fmt.Fprintf(stdout, "  %s\n", name)
		}
		fmt.Fprintln(stdout, "\nStates (lifecycle):")
		for _, name := range cfg.Symbols.StateNames() {
			fmt.Fprintf(stdout, "  %s\n", name)
		}
		fmt.Fprintf(stdout, "\nProjects: %v\n", cfg.Projects)
		fmt.Fprintf(stdout, "People:   %v\n", cfg.People)
		fmt.Fprintf(stdout, "\nDefaults: symbol=%s project=%s person=%s\n",
			cfg.Defaults.Symbol, cfg.Defaults.Project, cfg.Defaults.Person)
		return 0
	}

	if args[0] == "init" {
		if err := config.Init(configDir); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Config created at %s/bujotui.conf\n", configDir)
		return 0
	}

	fmt.Fprintf(stderr, "unknown config command: %s\n", args[0])
	return 1
}

func runTUI(configDir, dataDir string, _, stderr io.Writer) int {
	cfg, err := config.Load(configDir, dataDir)
	if err != nil {
		fmt.Fprintf(stderr, "error loading config: %v\n", err)
		return 1
	}
	store, err := storage.NewStore(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "error initializing store: %v\n", err)
		return 1
	}
	svc := service.NewEntryService(store, cfg)
	app := tui.New(svc, cfg)
	if err := app.Run(); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}
