package main

import (
	"flag"
	"fmt"
	"os"

	"edenx.dev/eden/internal/config"
	"edenx.dev/eden/internal/editor"
)

const helpText = `EdenX — a fast, minimal terminal text editor with encrypted file support.
Author: Temian Antoniu Mihai <antoniu@temian.ro>

Usage:
  eden [options] [file ...]

Options:
  --theme NAME   Set colour theme: default, green, dark, light, monokai
  -r, --readonly Open files in read-only mode

Keyboard Shortcuts:
  FILE
    Ctrl+S              Save
    Ctrl+E              Save encrypted (.ednx)
    Ctrl+Q              Quit

  EDIT
    Ctrl+Z / Ctrl+Y     Undo / Redo
    Shift+Arrow         Select text
    Ctrl+A              Select all
    Ctrl+C              Copy selection
    Ctrl+X              Cut selection
    Ctrl+V              Paste
    Ctrl+D              Duplicate line
    Ctrl+K              Delete to end of line
    Alt+↑ / Alt+↓      Move line up / down

  NAVIGATION
    Arrow keys          Move cursor
    Ctrl+← / Ctrl+→    Jump word left / right
    Home / End          Start / end of line
    PgUp / PgDn         Scroll by page
    Ctrl+J              Jump to line number
    Ctrl+T              New tab
    Ctrl+W              Close tab
    Alt+← / Alt+→      Switch tabs
    Ctrl+N              File navigator

  SEARCH & REPLACE
    Ctrl+F              Search
    Ctrl+R              Search & Replace
    ↑ / ↓               Previous / next match
    Ctrl+I              Toggle case-insensitive
    Ctrl+G              Toggle regex mode ($1 $2 in replace field)
    y / n / a           Replace / skip / replace all

  VIEW
    F1                  In-editor help overlay
    F2                  Cycle theme
    F3                  Toggle read-only mode

Syntax Highlighting:
  Go, Python, JS/TS, Rust, C/C++, Ruby, PHP, HTML, XML,
  CSS/SCSS, JSON, YAML, TOML, Markdown, Shell, Makefile

Config (~/.config/eden/config.json):
  {"theme": "dark", "tab_width": 4, "expand_tabs": true}

Website: https://edenx.dev
`

func main() {
	themeFlag := flag.String("theme", "", "Theme: default, green, dark, light, monokai")
	readonlyFlag := flag.Bool("r", false, "Open files in read-only mode")
	flag.BoolVar(readonlyFlag, "readonly", false, "Open files in read-only mode")
	flag.Usage = func() { fmt.Print(helpText) }
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}
	if *themeFlag != "" {
		cfg.Theme = *themeFlag
	}

	e, err := editor.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "eden: failed to start: %v\n", err)
		os.Exit(1)
	}
	defer e.Close()

	args := flag.Args()
	if len(args) > 0 {
		for _, f := range args {
			if err := e.OpenFile(f); err != nil {
				fmt.Fprintf(os.Stderr, "eden: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		e.NewEmptyBuffer()
	}
	if *readonlyFlag {
		e.MarkAllReadOnly()
	}

	if err := e.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "eden: %v\n", err)
		os.Exit(1)
	}
}
