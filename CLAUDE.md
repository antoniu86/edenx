# EdenX — Claude Code Context

This file is for Claude Code. Read it before making any changes.

## Project Identity
- **Brand:** EdenX  
- **Command:** `eden`  
- **Website:** edenx.dev  
- **Language:** Go 1.21+  
- **Terminal lib:** tcell/v2  

## Architecture

```
main.go                         Entry point, flag parsing
internal/config/config.go       ~/.config/eden/config.json
internal/crypto/crypto.go       AES-256-GCM + Argon2 (.ednx format)
internal/editor/
  theme.go                      5 themes + cycling logic
  buffer.go                     Buffer struct, editing ops, undo/redo
  search.go                     FindAll, Match type
  navigator.go                  File navigator (single panel)
  editor.go                     Editor struct, event loop, all modes
  render.go                     Screen drawing (tcell)
  brackets.go                   Bracket match scanning (findMatchingBracket)
  mouse.go                      Mouse click + scroll handling
  syntax.go                     Regex-based syntax tokenizer (16 languages)
```

## Encrypted File Format (.ednx)
```
[4]  magic:   EDNX
[1]  version: 0x01
[32] Argon2 salt
[12] AES-256-GCM nonce
[N]  ciphertext
```

## Editor Modes
| Mode | Description |
|---|---|
| ModeEdit | Normal editing |
| ModeSearch | Ctrl+F active |
| ModeReplace | Ctrl+H active |
| ModeNavigator | Ctrl+N overlay |
| ModeConfirmUnsaved | Unsaved warning |
| ModeFilenamePrompt | Save-as prompt |
| ModePasswordPrompt | Encrypt/decrypt prompt |

## Key Shortcuts
| Key | Action |
|---|---|
| Ctrl+S | Save |
| Ctrl+E | Save encrypted (.ednx) |
| Ctrl+Q | Quit |
| Ctrl+Z / Ctrl+Y | Undo / Redo |
| Ctrl+C/X/V | Copy / Cut / Paste |
| Shift+Arrow | Select text |
| Ctrl+F | Search |
| Ctrl+H | Search & Replace |
| Ctrl+T | New tab |
| Ctrl+W | Close tab |
| Alt+Left/Right | Switch tabs |
| Ctrl+N | File navigator |
| F2 | Cycle theme |

## Themes
`default` | `green` | `dark` | `light` | `monokai`  
Config: `~/.config/eden/config.json` → `{"theme": "dark"}`  
Override: `eden --theme monokai file.txt`

## Build
```bash
go mod tidy          # fetch deps (needs internet)
make                 # build ./eden
make install         # install to /usr/local/bin
make deb             # build .deb package
make release         # build all platform binaries in dist/
```

## Undo System
Uses full-snapshot undo (buffer.go: Snapshot struct).
Each editing op calls `b.checkpoint()` which pushes a copy of the
lines slice + cursor to the undo stack. Simple, correct, memory-fine
for typical file sizes.

## Known TODOs / v2 Features

### High priority (closes gap with nano)
- [x] Syntax highlighting — regex-based per filetype (Go, Python, JS/TS, JSON, YAML, Shell, C/C++, Rust, Markdown, HTML, CSS, TOML, Makefile)
- [x] Word-by-word navigation — Ctrl+←/→ to jump words
- [x] Auto-indent — preserve indentation on Enter
- [x] Configurable tab behaviour — tab_width (default 4), expand_tabs (default true), in config.json
- [x] Select all — Ctrl+A
- [x] Word-wrap for long lines — soft wrap, display only, buffer unchanged

### Medium priority (quality-of-life)
- [x] Mouse support — left-click to position cursor, wheel to scroll, click tab bar to switch tabs
- [x] Bracket matching highlight — highlights matching ( [ { pair under cursor
- [x] Line operations — Ctrl+D duplicate, Alt+↑/↓ move, Ctrl+K delete to EOL
- [ ] Auto-save / backup files
- [x] Read-only mode — -r/--readonly flag, F3 toggle; shows [RO] in status bar

### Toward vim parity (architectural)
- [x] Regex search — Ctrl+G toggles regex mode in search/replace; $1 $2 capture groups in replacement
- [ ] Ex-style command bar — :w :q :10 :s/foo/bar/g
- [ ] Modal editing — Insert/Normal/Visual modes (commit or skip; nano doesn't have it either)

### Distribution
- [ ] Homebrew formula for macOS
- [ ] Two-panel navigator
