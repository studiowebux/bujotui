package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/studiowebux/bujotui/internal/model"
)

// Config holds the parsed bujo configuration.
type Config struct {
	Symbols  *model.SymbolSet
	Projects []string
	People   []string
	Defaults Defaults
	Colors   map[string]string // state/symbol name -> ANSI color name
	Dir      string            // config directory
	DataDir  string            // data directory (daily/ lives here)
}

// ValidColors maps user-facing color names to ANSI escape codes.
var ValidColors = map[string]string{
	"red":          "\x1b[31m",
	"green":        "\x1b[32m",
	"yellow":       "\x1b[33m",
	"blue":         "\x1b[34m",
	"magenta":      "\x1b[35m",
	"cyan":         "\x1b[36m",
	"white":        "\x1b[37m",
	"gray":         "\x1b[90m",
	"bright_white": "\x1b[97m",
	"bold_red":     "\x1b[1m\x1b[31m",
	"bold_green":   "\x1b[1m\x1b[32m",
	"bold_yellow":  "\x1b[1m\x1b[33m",
	"bold_blue":    "\x1b[1m\x1b[34m",
	"bold_cyan":    "\x1b[1m\x1b[36m",
	"bold_white":   "\x1b[1m\x1b[37m",
}

// LookupColor returns the ANSI code for a state/symbol name, with a fallback.
func (c *Config) LookupColor(name, fallback string) string {
	if colorName, ok := c.Colors[name]; ok {
		if ansi, ok := ValidColors[colorName]; ok {
			return ansi
		}
	}
	return fallback
}

// Defaults holds default values for new entries.
type Defaults struct {
	Project string
	Person  string
	Symbol  string // symbol name
}

// DefaultConfigDir returns the config directory following XDG.
// Checks: $BUJOTUI_CONFIG_DIR > $XDG_CONFIG_HOME/bujotui > ~/.config/bujotui
func DefaultConfigDir() string {
	if dir := os.Getenv("BUJOTUI_CONFIG_DIR"); dir != "" {
		return dir
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "bujotui")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bujotui"
	}
	return filepath.Join(home, ".config", "bujotui")
}

// DefaultDataDir returns the data directory following XDG.
// Checks: $BUJOTUI_DATA_DIR > $XDG_DATA_HOME/bujotui > ~/.local/share/bujotui
func DefaultDataDir() string {
	if dir := os.Getenv("BUJOTUI_DATA_DIR"); dir != "" {
		return dir
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "bujotui")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bujotui"
	}
	return filepath.Join(home, ".local", "share", "bujotui")
}

// DefaultDir returns the default bujotui config directory.
// Deprecated: use DefaultConfigDir and DefaultDataDir instead.
func DefaultDir() string {
	return DefaultConfigDir()
}

// Load reads and parses the config from the given config directory.
// dataDir specifies where journal data (daily/) is stored.
// If bujotui.conf does not exist, returns a config with built-in defaults.
func Load(configDir, dataDir string) (*Config, error) {
	confPath := filepath.Join(configDir, "bujotui.conf")
	f, err := os.Open(confPath)
	if err != nil {
		if os.IsNotExist(err) {
			return parseReader(strings.NewReader(DefaultConf), configDir, dataDir)
		}
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	return parseReader(f, configDir, dataDir)
}

// Init writes the default config file to the given directory.
func Init(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	confPath := filepath.Join(dir, "bujotui.conf")
	if _, err := os.Stat(confPath); err == nil {
		return fmt.Errorf("config already exists: %s", confPath)
	}
	return os.WriteFile(confPath, []byte(DefaultConf), 0o644)
}

func parseReader(r io.Reader, configDir, dataDir string) (*Config, error) {
	cfg := &Config{
		Symbols: model.NewSymbolSet(),
		Dir:     configDir,
		DataDir: dataDir,
		Colors:  make(map[string]string),
		Defaults: Defaults{
			Project: "inbox",
			Person:  "self",
			Symbol:  "task",
		},
	}

	scanner := bufio.NewScanner(r)
	var section string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}

		switch section {
		case "symbols":
			name, char, ok := parseKV(line)
			if !ok {
				continue
			}
			cfg.Symbols.Add(name, char)

		case "transitions":
			name, targets, ok := parseKV(line)
			if !ok {
				continue
			}
			var tlist []string
			if strings.TrimSpace(targets) != "" {
				for _, t := range strings.Split(targets, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						tlist = append(tlist, t)
					}
				}
			}
			cfg.Symbols.SetTransitions(name, tlist)

		case "projects":
			cfg.Projects = append(cfg.Projects, line)

		case "people":
			cfg.People = append(cfg.People, line)

		case "colors":
			name, color, ok := parseKV(line)
			if !ok {
				continue
			}
			cfg.Colors[name] = color

		case "defaults":
			key, val, ok := parseKV(line)
			if !ok {
				continue
			}
			switch key {
			case "project":
				cfg.Defaults.Project = val
			case "person":
				cfg.Defaults.Person = val
			case "symbol":
				cfg.Defaults.Symbol = val
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks config consistency after parsing.
func (c *Config) validate() error {
	// Check for duplicate symbols (Add silently overwrites — catch it here)
	seen := make(map[string]bool)
	for _, name := range c.Symbols.Order {
		if seen[name] {
			return fmt.Errorf("config: duplicate symbol %q", name)
		}
		seen[name] = true
	}

	// Validate transition targets reference defined symbols
	for from, targets := range c.Symbols.Transitions {
		if _, ok := c.Symbols.Symbols[from]; !ok {
			return fmt.Errorf("config: transition source %q is not a defined symbol", from)
		}
		for _, to := range targets {
			if _, ok := c.Symbols.Symbols[to]; !ok {
				return fmt.Errorf("config: transition target %q (from %q) is not a defined symbol", to, from)
			}
		}
	}

	// Validate default symbol exists
	if c.Defaults.Symbol != "" {
		if _, ok := c.Symbols.Symbols[c.Defaults.Symbol]; !ok {
			return fmt.Errorf("config: default symbol %q is not defined", c.Defaults.Symbol)
		}
	}

	// Validate color names
	for name, color := range c.Colors {
		if _, ok := ValidColors[color]; !ok {
			return fmt.Errorf("config: unknown color %q for %q (valid: red, green, yellow, blue, magenta, cyan, white, gray, bright_white, bold_red, bold_green, bold_yellow, bold_blue, bold_cyan, bold_white)", color, name)
		}
	}

	return nil
}

// parseKV splits "key = value" and trims spaces.
func parseKV(line string) (key, value string, ok bool) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}
