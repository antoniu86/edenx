package editor

import "github.com/gdamore/tcell/v2"

const scrollLines = 3 // lines scrolled per wheel tick

func (e *Editor) handleMouse(ev *tcell.EventMouse) {
	x, y := ev.Position()
	btn := ev.Buttons()
	_, h := e.screen.Size()
	w, _ := e.screen.Size()
	contentWidth := w - lineNumWidth
	contentTop := 1

	switch {
	case btn&tcell.WheelUp != 0:
		e.scrollUp(scrollLines)
	case btn&tcell.WheelDown != 0:
		e.scrollDown(scrollLines)
	case btn&tcell.Button1 != 0:
		if y == 0 {
			e.handleTabBarClick(x)
		} else if y >= contentTop && y < h-1 {
			e.handleContentClick(x, y-contentTop, contentWidth)
		}
	}
}

// handleTabBarClick switches to the tab the user clicked on.
func (e *Editor) handleTabBarClick(x int) {
	pos := 0
	for i, buf := range e.buffers {
		label := []rune(" " + buf.TabLabel() + " ")
		end := pos + len(label)
		if x >= pos && x < end {
			e.activeIdx = i
			return
		}
		pos = end
		if i != e.activeIdx {
			pos++ // │ separator
		}
	}
}

// handleContentClick moves the cursor to the buffer position under the click.
func (e *Editor) handleContentClick(sx, contentRow, contentWidth int) {
	buf := e.currentBuffer()
	tw := e.cfg.TabWidthOrDefault()

	if contentWidth <= 0 {
		return
	}

	// Walk visual rows from topLine to find which buffer line the click lands on.
	visual := 0
	for r := buf.topLine; r < len(buf.lines); r++ {
		runes, byteAt := expandTabs(buf.lines[r], tw)
		vrows := visualRows(len(runes), contentWidth)

		if visual+vrows > contentRow {
			chunk := contentRow - visual     // which wrapped chunk within the line
			col := sx - lineNumWidth         // column within the content area
			if col < 0 {
				col = 0
			}
			displayIdx := chunk*contentWidth + col // index into the expanded rune slice

			var byteOffset int
			if displayIdx >= len(runes) {
				byteOffset = len(buf.lines[r])
			} else {
				byteOffset = byteAt[displayIdx]
			}

			buf.ClearSelection()
			buf.SetCursor(r, byteOffset)
			return
		}
		visual += vrows
	}

	// Click below last line — move to end of file.
	last := len(buf.lines) - 1
	buf.ClearSelection()
	buf.SetCursor(last, len(buf.lines[last]))
}

func (e *Editor) scrollUp(n int) {
	buf := e.currentBuffer()
	buf.topLine = clamp(buf.topLine-n, 0, len(buf.lines)-1)
	buf.cursor.Row = clamp(buf.cursor.Row-n, 0, len(buf.lines)-1)
	buf.cursor.Col = clamp(buf.cursor.Col, 0, len(buf.lines[buf.cursor.Row]))
}

func (e *Editor) scrollDown(n int) {
	buf := e.currentBuffer()
	buf.topLine = clamp(buf.topLine+n, 0, len(buf.lines)-1)
	buf.cursor.Row = clamp(buf.cursor.Row+n, 0, len(buf.lines)-1)
	buf.cursor.Col = clamp(buf.cursor.Col, 0, len(buf.lines[buf.cursor.Row]))
}
