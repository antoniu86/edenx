package editor

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

const lineNumWidth = 5 // "  42 " — 4 digits + space

// draw renders the entire editor to the screen.
func (e *Editor) draw() {
	e.screen.Clear()
	w, h := e.screen.Size()

	// Layout:
	//   row 0:        tab bar
	//   rows 1..h-2:  editor content
	//   row h-1:      status bar

	contentTop := 1
	contentBot := h - 2
	contentHeight := contentBot - contentTop + 1

	e.drawTabBar(w)
	e.drawContent(contentTop, contentHeight, w)
	e.drawStatusBar(h-1, w)

	if e.mode == ModeNavigator {
		e.drawNavigator(contentTop, contentHeight, w)
	}
	if e.mode == ModeHelp {
		e.drawHelp(contentTop, contentHeight, w)
	}

	// Position the hardware cursor in normal editing mode.
	if e.mode == ModeEdit {
		buf := e.currentBuffer()
		tw := e.cfg.TabWidthOrDefault()
		contentWidth := w - lineNumWidth
		topVisual := buf.absVisualRowOfLine(buf.topLine, contentWidth, tw)
		cursorVisual := buf.absVisualRow(contentWidth, tw)
		screenRow := contentTop + cursorVisual - topVisual
		screenCol := lineNumWidth + (buf.cursorVisualCol(tw) % contentWidth)
		e.screen.ShowCursor(screenCol, screenRow)
	} else {
		e.screen.HideCursor()
	}

	e.screen.Show()
}

// drawTabBar draws the tab bar at row 0.
func (e *Editor) drawTabBar(w int) {
	t := e.theme
	x := 0
	for i, buf := range e.buffers {
		label := " " + buf.TabLabel() + " "
		bg, fg := t.TabBg, t.TabFg
		if i == e.activeIdx {
			bg, fg = t.ActiveTabBg, t.ActiveTabFg
		}
		st := tcell.StyleDefault.Background(bg).Foreground(fg)
		for _, ch := range label {
			if x >= w {
				break
			}
			e.screen.SetContent(x, 0, ch, nil, st)
			x++
		}
		// Separator between inactive tabs.
		if x < w && i != e.activeIdx {
			e.screen.SetContent(x, 0, '│', nil, tcell.StyleDefault.Background(t.TabBg).Foreground(t.TabFg))
			x++
		}
	}
	// Fill the rest of the tab bar.
	st := tcell.StyleDefault.Background(t.TabBg).Foreground(t.TabFg)
	for ; x < w; x++ {
		e.screen.SetContent(x, 0, ' ', nil, st)
	}
}

// expandTabs returns a display rune slice with tabs replaced by spaces, and a
// parallel byte-offset slice mapping each display position back to the
// original line byte offset (used for selection / search / syntax lookups).
func expandTabs(line string, tabWidth int) ([]rune, []int) {
	if tabWidth < 1 {
		tabWidth = 4
	}
	runes := make([]rune, 0, len(line))
	byteAt := make([]int, 0, len(line)+1)
	for byteIdx, r := range line {
		if r == '\t' {
			for i := 0; i < tabWidth; i++ {
				runes = append(runes, ' ')
				byteAt = append(byteAt, byteIdx)
			}
		} else {
			runes = append(runes, r)
			byteAt = append(byteAt, byteIdx)
		}
	}
	byteAt = append(byteAt, len(line)) // sentinel
	return runes, byteAt
}

// drawContent draws the editor content area with soft word wrap.
func (e *Editor) drawContent(top, height, w int) {
	buf := e.currentBuffer()
	tw := e.cfg.TabWidthOrDefault()
	contentWidth := w - lineNumWidth
	buf.UpdateScroll(height, contentWidth, tw)
	t := e.theme
	bgSt := tcell.StyleDefault.Background(t.EditorBg).Foreground(t.EditorFg)
	lineNumSt := tcell.StyleDefault.Background(t.LineNumBg).Foreground(t.LineNumFg)

	synDef := synDefForFile(buf.filename)

	// Compute bracket match once per draw (only in edit mode).
	var bracketA, bracketB Pos
	hasBracket := false
	if e.mode == ModeEdit {
		bracketA, bracketB, hasBracket = findMatchingBracket(buf)
	}

	screenRow := 0
	bufRow := buf.topLine

	for screenRow < height {
		y := top + screenRow

		// Past end of file — clear remaining rows.
		if bufRow >= buf.LineCount() {
			for col := 0; col < lineNumWidth; col++ {
				e.screen.SetContent(col, y, ' ', nil, lineNumSt)
			}
			for x := lineNumWidth; x < w; x++ {
				e.screen.SetContent(x, y, ' ', nil, bgSt)
			}
			screenRow++
			continue
		}

		line := buf.Line(bufRow)

		// Expand tabs for display; byteAt maps each display column → byte offset.
		runes, byteAt := expandTabs(line, tw)

		// Tokenize once per logical line (shared across wrapped chunks).
		var synToks []SynToken
		if synDef != nil {
			synToks = tokenizeLine(line, synDef)
		}

		vrows := visualRows(len(runes), contentWidth)

		for chunk := 0; chunk < vrows && screenRow < height; chunk++ {
			y = top + screenRow

			// Line number: show on first chunk only, blank on continuations.
			var lineNumStr string
			if chunk == 0 {
				lineNumStr = fmt.Sprintf("%4d ", bufRow+1)
			} else {
				lineNumStr = "     "
			}
			for i, ch := range lineNumStr {
				if i >= lineNumWidth {
					break
				}
				e.screen.SetContent(i, y, ch, nil, lineNumSt)
			}

			// Content chunk: rune indices [startRune, startRune+contentWidth).
			startRune := chunk * contentWidth
			for col := 0; col < contentWidth; col++ {
				ri := startRune + col // rune index
				x := lineNumWidth + col

				var ch rune = ' '
				if ri < len(runes) {
					ch = runes[ri]
				}

				st := bgSt

				if ri < len(runes) {
					bo := byteAt[ri] // byte offset for this rune

					// Base style: syntax highlight.
					if synToks != nil && bo < len(synToks) && synToks[bo] != SynPlain {
						st = tcell.StyleDefault.Background(t.EditorBg).Foreground(t.synColor(synToks[bo]))
					}

					// Bracket match overrides syntax.
					if hasBracket && ((bufRow == bracketA.Row && bo == bracketA.Col) ||
						(bufRow == bracketB.Row && bo == bracketB.Col)) {
						st = tcell.StyleDefault.Background(t.BracketBg).Foreground(t.BracketFg)
					}

					// Selection overrides syntax.
					if buf.IsSelected(bufRow, bo) {
						st = tcell.StyleDefault.Background(t.SelectBg).Foreground(t.SelectFg)
					}

					// Search match overrides everything.
					if e.mode == ModeSearch || e.mode == ModeReplace {
						for mi, m := range e.matches {
							if MatchContains(m, bufRow, bo) {
								if mi == e.matchIdx {
									st = tcell.StyleDefault.Background(t.CurMatchBg).Foreground(t.CurMatchFg)
								} else {
									st = tcell.StyleDefault.Background(t.SearchBg).Foreground(t.SearchFg)
								}
								break
							}
						}
					}
				}

				e.screen.SetContent(x, y, ch, nil, st)
			}

			screenRow++
		}

		bufRow++
	}
}

// drawStatusBar draws the bottom status bar.
func (e *Editor) drawStatusBar(y, w int) {
	t := e.theme
	st := tcell.StyleDefault.Background(t.StatusBg).Foreground(t.StatusFg)

	var text string

	switch e.mode {
	case ModeSearch:
		flags := ""
		if e.caseInsensitive {
			flags += " [Ci]"
		}
		if e.regexMode {
			flags += " [Re]"
		}
		matchInfo := ""
		if len(e.matches) > 0 {
			wrapped := ""
			if e.wrapped {
				wrapped = " (wrapped)"
			}
			matchInfo = fmt.Sprintf(" — %d of %d%s", e.matchIdx+1, len(e.matches), wrapped)
		} else if e.searchTerm != "" {
			matchInfo = " — no matches"
		}
		text = fmt.Sprintf(" Search: %q%s%s  ↑↓ navigate  Ctrl+I case  Ctrl+G regex  Ctrl+R replace  Esc exit",
			e.searchTerm, flags, matchInfo)

	case ModeReplace:
		flags := ""
		if e.caseInsensitive {
			flags += " [Ci]"
		}
		if e.regexMode {
			flags += " [Re]"
		}
		if e.searchField == 0 {
			text = fmt.Sprintf(" Search: %s_%s  Ctrl+I case  Ctrl+G regex  Tab/Enter→Replace field", e.searchTerm, flags)
		} else {
			matchInfo := ""
			if len(e.matches) > 0 {
				matchInfo = fmt.Sprintf("  %d of %d", e.matchIdx+1, len(e.matches))
			}
			text = fmt.Sprintf(" Search: %q → Replace: %q%s%s  y=replace  n=skip  a=all  Esc=cancel",
				e.searchTerm, e.replaceTerm, flags, matchInfo)
		}

	case ModeConfirmUnsaved:
		text = e.confirmMsg

	case ModeFilenamePrompt:
		text = fmt.Sprintf(" %s: %s_", e.promptLabel, e.promptInput)

	case ModePasswordPrompt:
		stars := strings.Repeat("*", len(e.promptInput))
		label := e.promptLabel
		if e.promptConfirming {
			label = "Confirm password"
		}
		text = fmt.Sprintf(" %s: %s_", label, stars)

	case ModeJumpToLine:
		text = fmt.Sprintf(" Jump to line: %s_  Enter=go  Esc=cancel", e.promptInput)

	case ModeHelp:
		text = " Help — ↑↓ scroll  Esc/F1 close"

	case ModeNavigator:
		text = fmt.Sprintf(" Navigator — Enter=open  Esc=cancel  Path: %s", e.nav.Path)

	default:
		buf := e.currentBuffer()
		name := buf.DisplayName()
		if buf.IsEncrypted() {
			name = "[enc] " + name
		}
		modFlag := ""
		if buf.IsReadOnly() {
			modFlag = " [RO]"
		} else if buf.IsModified() {
			modFlag = " [+]"
		}
		pos := fmt.Sprintf("%d:%d", buf.cursor.Row+1, buf.cursor.Col+1)
		if e.statusMsg != "" {
			text = " " + e.statusMsg
		} else {
			left := fmt.Sprintf(" %s%s  %s  [%s]", name, modFlag, pos, e.theme.Name)
			right := "Ctrl+S=save  Ctrl+E=encrypt  Ctrl+Q=close  F1=help "
			pad := w - len(left) - len(right)
			if pad < 1 {
				pad = 1
			}
			text = left + strings.Repeat(" ", pad) + right
		}
	}

	runes := []rune(text)
	for x := 0; x < w; x++ {
		var ch rune = ' '
		if x < len(runes) {
			ch = runes[x]
		}
		e.screen.SetContent(x, y, ch, nil, st)
	}
}

// drawNavigator draws the file navigator overlay.
func (e *Editor) drawNavigator(top, height, w int) {
	t := e.theme
	navW := w / 2
	if navW < 40 {
		navW = 40
	}
	navX := (w - navW) / 2
	navH := height - 2
	if navH < 5 {
		navH = 5
	}
	navY := top + 1

	borderSt := tcell.StyleDefault.Background(t.NavBg).Foreground(t.NavBorderFg)
	itemSt := tcell.StyleDefault.Background(t.NavBg).Foreground(t.NavFg)
	activeSt := tcell.StyleDefault.Background(t.NavSelBg).Foreground(t.NavSelFg)
	dirSt := tcell.StyleDefault.Background(t.NavBg).Foreground(t.CurMatchBg)

	// Top border.
	drawHLine(e.screen, navX, navY, navW, borderSt, '─', '┌', '┐')
	navY++

	// Title row.
	title := " " + e.nav.Path + " "
	drawRow(e.screen, navX, navY, navW, title, borderSt)
	navY++
	drawHLine(e.screen, navX, navY, navW, borderSt, '─', '├', '┤')
	navY++

	// Entry list.
	visibleRows := navH - 4 // top border + title + separator + bottom border
	startIdx := 0
	if e.nav.Idx >= visibleRows {
		startIdx = e.nav.Idx - visibleRows + 1
	}

	for row := 0; row < visibleRows; row++ {
		entryIdx := startIdx + row
		if navY >= top+height {
			break
		}

		st := itemSt
		label := ""
		if entryIdx < len(e.nav.Entries) {
			entry := e.nav.Entries[entryIdx]
			if entryIdx == e.nav.Idx {
				st = activeSt
			} else if entry.IsDir {
				st = dirSt
			}
			prefix := "  "
			if entry.IsDir {
				prefix = "  / "
			}
			label = prefix + entry.Name
			if entry.IsDir {
				label += "/"
			}
		}
		drawRow(e.screen, navX, navY, navW, label, st)
		navY++
	}

	// Bottom border.
	drawHLine(e.screen, navX, navY, navW, borderSt, '─', '└', '┘')
}

func drawHLine(s tcell.Screen, x, y, w int, st tcell.Style, fill, left, right rune) {
	s.SetContent(x, y, left, nil, st)
	for i := x + 1; i < x+w-1; i++ {
		s.SetContent(i, y, fill, nil, st)
	}
	s.SetContent(x+w-1, y, right, nil, st)
}

func drawRow(s tcell.Screen, x, y, w int, text string, st tcell.Style) {
	s.SetContent(x, y, '│', nil, st)
	cx := x + 1
	for _, ch := range text {
		if cx >= x+w-1 {
			break
		}
		s.SetContent(cx, y, ch, nil, st)
		cx++
	}
	for ; cx < x+w-1; cx++ {
		s.SetContent(cx, y, ' ', nil, st)
	}
	s.SetContent(x+w-1, y, '│', nil, st)
}

// helpLines is the static content shown in the help overlay.
var helpLines = []string{
	"  EdenX — Keyboard Shortcuts",
	"  Author: Temian Antoniu Mihai <antoniu@temian.ro>",
	"",
	"  FILE",
	"    Ctrl+S          Save",
	"    Ctrl+E          Save encrypted (.ednx)",
	"    Ctrl+Q          Quit",
	"",
	"  EDIT",
	"    Ctrl+Z          Undo",
	"    Ctrl+Y          Redo",
	"    Shift+Arrow     Select text",
	"    Ctrl+A          Select all",
	"    Ctrl+C          Copy selection",
	"    Ctrl+X          Cut selection",
	"    Ctrl+V          Paste",
	"    Ctrl+D          Duplicate line",
	"    Ctrl+K          Delete to end of line",
	"    Alt+↑ / Alt+↓  Move line up / down",
	"",
	"  NAVIGATION",
	"    Ctrl+J          Jump to line",
	"    Arrow keys      Move cursor",
	"    Ctrl+← / →     Jump word left / right",
	"    Home / End      Start / end of line",
	"    PgUp / PgDn     Scroll by page",
	"    Ctrl+T          New tab",
	"    Ctrl+W          Close tab",
	"    Alt+← / Alt+→  Previous / next tab",
	"    Ctrl+N          File navigator",
	"",
	"  SEARCH & REPLACE",
	"    Ctrl+F          Search",
	"    Ctrl+R          Search & Replace",
	"    ↑ / ↓           Previous / next match",
	"    Ctrl+I          Toggle case sensitivity",
	"    Ctrl+G          Toggle regex mode",
	"    y / n / a       Replace / skip / replace all",
	"    (regex) $1 $2   Capture groups in replacement",
	"",
	"  VIEW",
	"    F1              This help",
	"    F2              Cycle theme",
	"    F3              Toggle read-only mode",
	"",
	"  SYNTAX HIGHLIGHTING",
	"    Go, Python, JS/TS, Rust, C/C++, Ruby, PHP",
	"    HTML, XML, CSS/SCSS, JSON, YAML, TOML",
	"    Markdown, Shell, Makefile",
}

// drawHelp draws the help overlay.
func (e *Editor) drawHelp(top, height, w int) {
	t := e.theme

	boxW := 54
	if boxW > w-4 {
		boxW = w - 4
	}
	boxX := (w - boxW) / 2

	// Visible content rows = height minus top/bottom border and title rows.
	innerH := height - 4 // top border + title + separator + bottom border
	if innerH < 1 {
		innerH = 1
	}

	// Clamp scroll.
	maxScroll := len(helpLines) - innerH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.helpScroll > maxScroll {
		e.helpScroll = maxScroll
	}
	if e.helpScroll < 0 {
		e.helpScroll = 0
	}

	borderSt := tcell.StyleDefault.Background(t.NavBg).Foreground(t.NavBorderFg)
	textSt := tcell.StyleDefault.Background(t.NavBg).Foreground(t.NavFg)

	y := top + 1

	// Top border.
	drawHLine(e.screen, boxX, y, boxW, borderSt, '─', '┌', '┐')
	y++

	// Title row.
	scrollHint := ""
	if len(helpLines) > innerH {
		scrollHint = fmt.Sprintf(" (%d/%d)", e.helpScroll+1, maxScroll+1)
	}
	drawRow(e.screen, boxX, y, boxW, "  Keyboard Shortcuts"+scrollHint, borderSt)
	y++

	// Separator.
	drawHLine(e.screen, boxX, y, boxW, borderSt, '─', '├', '┤')
	y++

	// Content lines.
	for row := 0; row < innerH; row++ {
		lineIdx := e.helpScroll + row
		line := ""
		if lineIdx < len(helpLines) {
			line = helpLines[lineIdx]
		}
		drawRow(e.screen, boxX, y, boxW, line, textSt)
		y++
	}

	// Bottom border.
	drawHLine(e.screen, boxX, y, boxW, borderSt, '─', '└', '┘')
}
