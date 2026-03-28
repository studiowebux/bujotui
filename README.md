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
├── cmd/bujotui/        # TUI + CLI entry point
├── cmd/bujotui-mcp/    # MCP server entry point
├── internal/
│   ├── cli/            # CLI command handlers
│   ├── complete/       # Autocomplete engine
│   ├── config/         # Configuration parsing (XDG-aware)
│   ├── mcp/            # MCP protocol, tools, and handler
│   ├── markdown/       # Markdown file format (read/write)
│   ├── model/          # Domain types (Entry, Collection, Habit, FutureEntry)
│   ├── service/        # Business logic (entries, collections, habits, future log)
│   ├── storage/        # File-level persistence (atomic writes)
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
go build -o bujotui-mcp ./cmd/bujotui-mcp
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
| `Enter` | Jump to migration link |
| `m` | Calendar view |
| `f` | Future log |
| `h` | Habit tracker |
| `p` | Collections |
| `I` | Index |
| `?` | Show help |
| `q` | Quit |

### State Transitions

| Key | Action |
|-----|--------|
| `x` | Mark done |
| `>` | Migrate to date (opens picker) |
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

### Calendar View (`m`)

| Key | Action |
|-----|--------|
| `j/k` | Move up/down through days |
| `[ ]` | Previous/next month |
| `i/n` | Edit daily note |
| `Enter` | Open selected day |
| `Esc` | Back to normal |

### Future Log (`f`)

| Key | Action |
|-----|--------|
| `j/k` | Move up/down through entries |
| `[ ]` | Previous/next month tab |
| `a` | Add entry to selected month |
| `d` | Delete entry (confirm y/n) |
| `Esc` | Back to normal |

### Habit Tracker (`h`)

| Key | Action |
|-----|--------|
| `j/k` | Move up/down through habits |
| `[ ]` | Previous/next day column |
| `x` / `Space` | Toggle habit for selected day |
| `a` | Add new habit |
| `d` | Delete habit (confirm y/n) |
| `Esc` | Back to normal |

### Collections (`p`)

| Key | Action |
|-----|--------|
| `j/k` | Move up/down |
| `a` | Create new collection |
| `d` | Delete collection (confirm y/n) |
| `Enter` | Open collection |
| `Esc` | Back to normal |

Within a collection:

| Key | Action |
|-----|--------|
| `j/k` | Move up/down |
| `a` | Add item |
| `e` | Edit item |
| `d` | Delete item |
| `x` / `Space` | Toggle done |
| `J/K` | Reorder items |
| `Esc` | Back to collections list |

### Index (`I`)

| Key | Action |
|-----|--------|
| `j/k` | Move up/down |
| `/` | Filter/search |
| `Enter` | Open (collection or project filter) |
| `Esc` | Back to normal |

### Migration Linking

When an entry is migrated with `>`, both sides get linked:
- Original: `-> 2026-04-01` (shown in cyan)
- Copy: `<- 2026-03-28` (shown in cyan)

Press `Enter` on a linked entry to jump to the target/source date.

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

## MCP Server

bujotui includes an MCP server (`bujotui-mcp`) that exposes journal operations as tools over stdio. This lets AI agents (Claude Code, etc.) read and write journal entries, collections, habits, and the future log.

### MCP Tools

#### Journal Entries

| Tool | Description |
|------|-------------|
| `add_entry` | Add a new journal entry |
| `list_entries` | List entries for a date |
| `edit_entry` | Edit an entry by index |
| `transition_entry` | Change entry state (done, migrated, etc.) |
| `delete_entry` | Remove an entry |
| `set_note` | Set daily note for a date |
| `list_month` | List all entries and notes for a month |
| `search` | Search entries across all fields |

#### Collections

| Tool | Description |
|------|-------------|
| `list_collections` | List all collection names |
| `get_collection` | Get a collection's items |
| `create_collection` | Create a new empty collection |
| `delete_collection` | Delete a collection |
| `add_collection_item` | Add an item to a collection |
| `remove_collection_item` | Remove an item by index |
| `toggle_collection_item` | Toggle item done state |

#### Habits

| Tool | Description |
|------|-------------|
| `list_habits` | List habit names for a month |
| `add_habit` | Add a new habit to track |
| `remove_habit` | Remove a habit |
| `toggle_habit` | Toggle habit completion for a day |
| `get_habits_month` | Get full habit data with streaks |

#### Future Log

| Tool | Description |
|------|-------------|
| `list_future` | List future log entries for a year |
| `add_future_entry` | Add an entry to a future month |
| `remove_future_entry` | Remove a future log entry |

### Usage with Claude Code

#### Option 1: Project config (`.mcp.json`)

```json
{
  "mcpServers": {
    "bujotui": {
      "type": "stdio",
      "command": "/absolute/path/to/bujotui-mcp"
    }
  }
}
```

#### Option 2: CLI

```bash
claude mcp add --transport stdio bujotui -- /absolute/path/to/bujotui-mcp
```

### MCP CLI Flags

```bash
bujotui-mcp                          # default: logs to stderr
bujotui-mcp -logfile /tmp/bujo.log   # redirect logs to a file
bujotui-mcp -version                 # print version and exit
```

## Data Format

All data is stored as markdown files under the data directory (`~/.local/share/bujotui/`).

### Daily Entries

Monthly files in `daily/`:

```
daily/2026-03.md
```

```markdown
# 2026-03-27

- . 2026-03-27T14:30 [project] @person Description text
- . 2026-03-27T15:00 [work] @self state:done Finished the report
- . 2026-03-27T16:00 [work] @self state:migrated ->2026-03-28 Fix the bug
```

Migration links are stored inline: `->YYYY-MM-DD` on the source, `<-YYYY-MM-DD` on the copy.

### Collections

One file per collection in `collections/`:

```
collections/books-to-read.md
```

```markdown
# Books to Read

- [ ] The Pragmatic Programmer
- [x] Clean Code
- [ ] Designing Data-Intensive Applications
```

### Habits

Monthly files in `habits/`:

```
habits/2026-03.md
```

```markdown
# Habits 2026-03

## Exercise
1,3,5,7,10,15

## Read
1,2,3,4,5,6,7
```

### Future Log

Yearly files in `future/`:

```
future/2026.md
```

```markdown
# Future Log 2026

## April
- . Doctor appointment
- o Conference

## June
- . Tax deadline
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
