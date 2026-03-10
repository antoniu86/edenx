# EdenX

A fast, minimal terminal text editor with built-in encrypted file support.

**Website:** https://edenx.dev

## Features

- Encrypted file format (`.ednx`) — AES-256-GCM + Argon2
- Syntax highlighting — Go, Python, JS/TS, Rust, C/C++, Ruby, PHP, HTML, XML, CSS/SCSS, JSON, YAML, TOML, Markdown, Shell, Makefile
- Incremental search and replace with regex support and capture group expansion
- Multi-buffer tabbed editing
- Single-panel file navigator
- 8 color themes (including terminal default inheritance)
- Soft word wrap, auto-indent, configurable tab width
- Bracket matching highlight
- Mouse support — click to position, scroll, click tabs
- Undo / redo, text selection, copy, cut, paste
- Read-only mode (`-r` flag or `F3` toggle)

## Install

**From source (any Linux):**
```bash
git clone https://github.com/edenx/eden
cd eden
make && make install
```

**Debian/Ubuntu (.deb):**
```bash
make deb
sudo apt install ./eden_0.6.0_amd64.deb
```

## Usage

```bash
eden                      # open empty buffer
eden file.txt             # open plain text file
eden notes.ednx           # open encrypted file (prompts for password)
eden --theme dark file    # override theme
eden -r file.txt          # open in read-only mode
eden --help               # show all options and shortcuts
```

## Shortcuts

### File

| Key | Action |
|---|---|
| `Ctrl+S` | Save |
| `Ctrl+E` | Save as encrypted `.ednx` |
| `Ctrl+Q` | Quit |

### Edit

| Key | Action |
|---|---|
| `Ctrl+Z` / `Ctrl+Y` | Undo / Redo |
| `Ctrl+A` | Select all |
| `Shift+Arrow` | Extend selection |
| `Ctrl+C` | Copy selection |
| `Ctrl+X` | Cut selection |
| `Ctrl+V` | Paste |
| `Ctrl+D` | Duplicate line |
| `Ctrl+K` | Delete to end of line |
| `Alt+↑` / `Alt+↓` | Move line up / down |

### Navigation

| Key | Action |
|---|---|
| `Arrow keys` | Move cursor |
| `Ctrl+←` / `Ctrl+→` | Jump word left / right |
| `Home` / `End` | Start / end of line |
| `PgUp` / `PgDn` | Scroll by page |
| `Ctrl+J` | Jump to line number |
| `Ctrl+T` | New tab |
| `Ctrl+W` | Close tab |
| `Alt+←` / `Alt+→` | Switch tabs |
| `Ctrl+N` | File navigator |

### Search & Replace

| Key | Action |
|---|---|
| `Ctrl+F` | Search |
| `Ctrl+R` | Search & Replace |
| `↑` / `↓` | Previous / next match |
| `Ctrl+I` | Toggle case-insensitive |
| `Ctrl+G` | Toggle regex mode |
| `y` / `n` / `a` | Replace / skip / replace all |

> In regex mode, use `$1`, `$2` etc. in the replacement field for capture groups.

### View

| Key | Action |
|---|---|
| `F1` | Keyboard shortcut help overlay |
| `F2` | Cycle theme |
| `F3` | Toggle read-only mode |

## Config

`~/.config/eden/config.json`:
```json
{
  "theme": "dark"
}
```

Themes: `default`, `monokai`, `ocean`, `solarized`, `rose`, `green`, `dark`, `light`

Tab behaviour:
- `tab_width` — visual width of a tab / spaces inserted (default: `4`)
- `expand_tabs` — insert spaces instead of `\t` on Tab key (default: `true`)

## Build for all platforms

```bash
make release
# Produces: dist/eden-linux-amd64, eden-linux-arm64, eden-darwin-arm64, ...
```

## License

MIT
