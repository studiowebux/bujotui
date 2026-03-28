# bujotui

A terminal-based bullet journal with vim-style navigation, state transitions, and a form-based entry editor.

Zero external dependencies. Pure Go stdlib.

Bug tracker: https://github.com/studiowebux/bujotui/issues
<br>
Discord: https://discord.gg/BG5Erm9fNv

## Funding

[Buy Me a Coffee](https://buymeacoffee.com/studiowebux)
<br>
[GitHub Sponsors](https://github.com/sponsors/studiowebux)
<br>
[Patreon](https://patreon.com/studiowebux)

## Directory Structure

```
bujotui/
├── cmd/bujotui/        # CLI entry point
├── internal/
│   ├── cli/            # CLI command handlers
│   ├── complete/       # Autocomplete engine
│   ├── config/         # Configuration parsing (XDG-aware)
│   ├── markdown/       # Markdown file format (read/write)
│   ├── model/          # Domain types (Entry, Symbol, SymbolSet)
│   ├── service/        # Business logic (CRUD, transitions, filtering)
│   ├── storage/        # File-level persistence
│   ├── term/           # Raw terminal control (ANSI, termios)
│   └── tui/            # Terminal UI (rendering, key handling, forms)
├── .gitignore
├── go.mod
└── README.md
```

## Installation

### From source

```bash
git clone https://github.com/studiowebux/bujotui
cd bujotui
go build -o bujotui ./cmd/bujotui
```

## Usage

```bash
bujotui                          # Launch TUI
bujotui add "Buy milk"           # Add entry from CLI
bujotui add -s event -p work "Standup meeting"
bujotui list                     # List today's entries
bujotui list --week              # List current week
bujotui list --time              # Show timestamps
bujotui done 1                   # Mark entry #1 as done
bujotui migrate 2                # Mark entry #2 as migrated
bujotui schedule 3               # Mark entry #3 as scheduled
bujotui cancel 1                 # Mark entry #1 as cancelled
bujotui remove 1                 # Delete entry #1
bujotui projects                 # List known projects
bujotui people                   # List known people
bujotui config                   # Show current configuration
bujotui config init              # Create default config file
bujotui version                  # Show version
bujotui help                     # Show help
```

### CLI Flags (add)

```
-s symbol     Symbol name (default: task)
-p project    Project name
-a person     Assignee
-d datetime   Date/time as YYYY-MM-DDThh:mm (default: now)
```

## TUI Keybindings

Press `?` in the TUI to see all keybindings.

### Normal Mode

| Key | Action |
|-----|--------|
| `j/k` | Move up/down |
| `G` | Go to last entry |
| `g` | Go to first entry |
| `a` | Add new entry |
| `e` | Edit selected entry |
| `d` | Delete entry (confirm y/n) |
| `/` | Filter entries |
| `Esc` | Clear active filter |
| `[ ]` | Previous/next day |
| `t` | Toggle time column |
| `?` | Show help |
| `q` | Quit |

### State Transitions

| Key | Action |
|-----|--------|
| `x` | Mark done |
| `>` | Mark migrated |
| `<` | Mark scheduled |
| `c` | Mark cancelled |
| `r` | Reset state (clear) |

### Form (add/edit)

| Key | Action |
|-----|--------|
| `Tab` | Next field / cycle completions |
| `Shift+Tab` | Previous field |
| `Enter` | Accept completion or submit |
| `Esc` | Cancel |

### Filter Syntax

```
project:name    filter by project
@person         filter by assignee
symbol:name     filter by symbol type
free text       search all fields
```

## Configuration

Config file: `bujotui.conf`

### Directories

bujotui follows the XDG Base Directory specification:

| Purpose | Env Var | Fallback | Default |
|---------|---------|----------|---------|
| Config | `BUJOTUI_CONFIG_DIR` | `$XDG_CONFIG_HOME/bujotui` | `~/.config/bujotui` |
| Data | `BUJOTUI_DATA_DIR` | `$XDG_DATA_HOME/bujotui` | `~/.local/share/bujotui` |

`BUJOTUI_DIR` sets both config and data to the same directory.

`--dir /path` overrides both at runtime.

### Config Sections

```ini
[symbols]
task = .
event = o
note = -
idea = *
urgent = !
waiting = ~
health = +
done = x
migrated = >
scheduled = <
cancelled = X

[transitions]
task = done, migrated, scheduled, cancelled
event =
note =
idea = done, cancelled
urgent = done, migrated, cancelled
waiting = done, cancelled
health =
done =
migrated =
scheduled = done, cancelled
cancelled =

[colors]
done = green
cancelled = red
migrated = blue
scheduled = cyan

[projects]
inbox

[people]
self

[defaults]
project = inbox
person = self
symbol = task
```

### Available Colors

`red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `gray`, `bright_white`, `bold_red`, `bold_green`, `bold_yellow`, `bold_blue`, `bold_cyan`, `bold_white`

## Data Format

Entries are stored as markdown in monthly files under the data directory:

```
~/.local/share/bujotui/daily/2026-03.md
```

```markdown
# 2026-03-27

- . 2026-03-27T14:30 [project] @person Description text
- . 2026-03-27T15:00 [work] @self state:done Finished the report
```

## Contributions

1. Fork the repository
2. Create a branch: `git checkout -b feat/your-feature`
3. Commit your changes
4. Open a pull request

Open an issue before starting significant work.

## License

[MIT](LICENSE)

## Contact

[Studio Webux](https://studiowebux.com)
<br>
tommy@studiowebux.com
<br>
[Discord](https://discord.gg/BG5Erm9fNv)
