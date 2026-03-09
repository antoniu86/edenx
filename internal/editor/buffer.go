package editor

import (
	"path/filepath"
	"strings"
)

// Pos is a (row, col) position in a buffer.
type Pos struct {
	Row, Col int
}

func posLess(a, b Pos) bool {
	return a.Row < b.Row || (a.Row == b.Row && a.Col < b.Col)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// snapshot is a saved state for undo/redo.
type snapshot struct {
	lines  []string
	cursor Pos
}

// Buffer holds the editable state of a single open document.
type Buffer struct {
	filename    string
	lines       []string
	cursor      Pos
	topLine     int    // first visible row (scroll offset)
	modified    bool
	encrypted   bool
	readonly    bool
	password    []byte // cached after first successful decrypt/encrypt; zeroed on close
	pendingData []byte // raw .ednx bytes pending decryption

	// Selection
	selActive bool
	selAnchor Pos

	// Undo / Redo
	undoStack []snapshot
	redoStack []snapshot
}

// NewBuffer returns a Buffer with a single empty line.
func NewBuffer() *Buffer {
	return &Buffer{lines: []string{""}}
}

// --- Read-only accessors used by render.go ---

func (b *Buffer) IsModified() bool  { return b.modified }
func (b *Buffer) IsEncrypted() bool { return b.encrypted }
func (b *Buffer) IsReadOnly() bool  { return b.readonly }
func (b *Buffer) SetReadOnly(v bool) { b.readonly = v }

func (b *Buffer) DisplayName() string {
	if b.filename == "" {
		return "[untitled]"
	}
	return filepath.Base(b.filename)
}

func (b *Buffer) TabLabel() string {
	name := b.DisplayName()
	if b.encrypted {
		name = "[enc] " + name
	}
	if b.modified {
		name += " +"
	}
	return name
}

func (b *Buffer) LineCount() int { return len(b.lines) }

func (b *Buffer) Line(row int) string {
	if row < 0 || row >= len(b.lines) {
		return ""
	}
	return b.lines[row]
}

// IsSelected reports whether (row, col) falls within the active selection.
func (b *Buffer) IsSelected(row, col int) bool {
	if !b.selActive {
		return false
	}
	p := Pos{row, col}
	start, end := b.selAnchor, b.cursor
	if posLess(end, start) {
		start, end = end, start
	}
	return !posLess(p, start) && posLess(p, end)
}

// lineVisualWidth returns the visual width of line with tabs expanded to
// tabWidth-space stops.
func lineVisualWidth(line string, tabWidth int) int {
	if tabWidth < 1 {
		tabWidth = 4
	}
	w := 0
	for _, r := range line {
		if r == '\t' {
			w += tabWidth
		} else {
			w++
		}
	}
	return w
}

// visualRows returns the number of screen rows a line occupies when the
// content area is contentWidth columns wide, with tabs expanded.
func visualRows(visualWidth, contentWidth int) int {
	if contentWidth <= 0 || visualWidth == 0 {
		return 1
	}
	n := visualWidth / contentWidth
	if visualWidth%contentWidth != 0 {
		n++
	}
	return n
}

// cursorVisualCol returns the cursor's visual screen column within its line,
// expanding tabs to tabWidth spaces.
func (b *Buffer) cursorVisualCol(tabWidth int) int {
	if tabWidth < 1 {
		tabWidth = 4
	}
	line := b.lines[b.cursor.Row]
	vc := 0
	for byteIdx, r := range line {
		if byteIdx >= b.cursor.Col {
			break
		}
		if r == '\t' {
			vc += tabWidth
		} else {
			vc++
		}
	}
	return vc
}

// absVisualRow returns the absolute visual row index of the cursor,
// counting from the first line of the buffer.
func (b *Buffer) absVisualRow(contentWidth, tabWidth int) int {
	v := 0
	for r := 0; r < b.cursor.Row; r++ {
		v += visualRows(lineVisualWidth(b.lines[r], tabWidth), contentWidth)
	}
	v += b.cursorVisualCol(tabWidth) / contentWidth
	return v
}

// absVisualRowOfLine returns the absolute visual row index of the start
// of buffer line r.
func (b *Buffer) absVisualRowOfLine(r, contentWidth, tabWidth int) int {
	v := 0
	for i := 0; i < r && i < len(b.lines); i++ {
		v += visualRows(lineVisualWidth(b.lines[i], tabWidth), contentWidth)
	}
	return v
}

// UpdateScroll adjusts topLine so the cursor stays visible, accounting for
// wrapped lines and tab expansion.
func (b *Buffer) UpdateScroll(height, contentWidth, tabWidth int) {
	if contentWidth <= 0 {
		contentWidth = 1
	}
	cursorVisual := b.absVisualRow(contentWidth, tabWidth)
	topVisual := b.absVisualRowOfLine(b.topLine, contentWidth, tabWidth)

	if cursorVisual < topVisual {
		b.topLine = b.cursor.Row
	} else if cursorVisual >= topVisual+height {
		target := cursorVisual - height + 1
		v := 0
		for r := 0; r < len(b.lines); r++ {
			nextV := v + visualRows(lineVisualWidth(b.lines[r], tabWidth), contentWidth)
			if nextV > target {
				b.topLine = r
				return
			}
			v = nextV
		}
		b.topLine = len(b.lines) - 1
	}
}

// Content returns the full buffer text.
func (b *Buffer) Content() string {
	return strings.Join(b.lines, "\n")
}

// SetContent loads s into the buffer, resetting all state.
func (b *Buffer) SetContent(s string) {
	b.lines = strings.Split(s, "\n")
	b.cursor = Pos{}
	b.topLine = 0
	b.modified = false
	b.selActive = false
	b.undoStack = nil
	b.redoStack = nil
}

// SetCursor moves the cursor, clamping to valid bounds.
func (b *Buffer) SetCursor(row, col int) {
	b.cursor.Row = clamp(row, 0, len(b.lines)-1)
	b.cursor.Col = clamp(col, 0, len(b.lines[b.cursor.Row]))
}

// SelectedText returns the currently selected text (empty if no selection).
func (b *Buffer) SelectedText() string {
	if !b.selActive {
		return ""
	}
	start, end := b.selAnchor, b.cursor
	if posLess(end, start) {
		start, end = end, start
	}
	if start.Row == end.Row {
		return b.lines[start.Row][start.Col:end.Col]
	}
	var parts []string
	parts = append(parts, b.lines[start.Row][start.Col:])
	for r := start.Row + 1; r < end.Row; r++ {
		parts = append(parts, b.lines[r])
	}
	parts = append(parts, b.lines[end.Row][:end.Col])
	return strings.Join(parts, "\n")
}

// zeroPassword overwrites the cached password bytes with zeros and clears the slice.
func (b *Buffer) zeroPassword() {
	for i := range b.password {
		b.password[i] = 0
	}
	b.password = nil
}

// ClearSelection deactivates the selection.
func (b *Buffer) ClearSelection() { b.selActive = false }

// StartSelection anchors the selection at the current cursor.
func (b *Buffer) StartSelection() {
	b.selAnchor = b.cursor
	b.selActive = true
}

// SelectAll selects the entire buffer contents.
func (b *Buffer) SelectAll() {
	b.selAnchor = Pos{0, 0}
	lastRow := len(b.lines) - 1
	b.cursor = Pos{lastRow, len(b.lines[lastRow])}
	b.selActive = true
}

// FindMatches returns all occurrences of term in the buffer.
func (b *Buffer) FindMatches(term string, caseInsensitive, useRegex bool) ([]Match, error) {
	return FindAll(b.lines, term, caseInsensitive, useRegex)
}

// --- Undo / Redo ---

const maxUndoStack = 100

func (b *Buffer) checkpoint() {
	cp := snapshot{lines: make([]string, len(b.lines)), cursor: b.cursor}
	copy(cp.lines, b.lines)
	b.undoStack = append(b.undoStack, cp)
	if len(b.undoStack) > maxUndoStack {
		// Drop the oldest entry to keep memory bounded.
		b.undoStack = b.undoStack[1:]
	}
	b.redoStack = nil
}

// Undo reverts the last edit. Returns false if nothing to undo.
func (b *Buffer) Undo() bool {
	if len(b.undoStack) == 0 {
		return false
	}
	cp := snapshot{lines: make([]string, len(b.lines)), cursor: b.cursor}
	copy(cp.lines, b.lines)
	b.redoStack = append(b.redoStack, cp)
	s := b.undoStack[len(b.undoStack)-1]
	b.undoStack = b.undoStack[:len(b.undoStack)-1]
	b.lines = make([]string, len(s.lines))
	copy(b.lines, s.lines)
	b.cursor = s.cursor
	b.modified = true
	b.selActive = false
	return true
}

// Redo reapplies the last undone edit. Returns false if nothing to redo.
func (b *Buffer) Redo() bool {
	if len(b.redoStack) == 0 {
		return false
	}
	cp := snapshot{lines: make([]string, len(b.lines)), cursor: b.cursor}
	copy(cp.lines, b.lines)
	b.undoStack = append(b.undoStack, cp)
	s := b.redoStack[len(b.redoStack)-1]
	b.redoStack = b.redoStack[:len(b.redoStack)-1]
	b.lines = make([]string, len(s.lines))
	copy(b.lines, s.lines)
	b.cursor = s.cursor
	b.modified = true
	b.selActive = false
	return true
}

// --- Editing operations ---

// InsertChar inserts rune r at the cursor.
func (b *Buffer) InsertChar(r rune) {
	b.deleteSelection()
	b.checkpoint()
	row, col := b.cursor.Row, b.cursor.Col
	line := b.lines[row]
	b.lines[row] = line[:col] + string(r) + line[col:]
	b.cursor.Col++
	b.modified = true
}

// InsertNewline splits the current line at the cursor, preserving indentation.
func (b *Buffer) InsertNewline() {
	b.deleteSelection()
	b.checkpoint()
	row, col := b.cursor.Row, b.cursor.Col
	line := b.lines[row]

	// Capture leading whitespace from the current line.
	indent := ""
	for _, ch := range line {
		if ch == ' ' || ch == '\t' {
			indent += string(ch)
		} else {
			break
		}
	}

	before, after := line[:col], line[col:]
	b.lines[row] = before
	newLines := make([]string, len(b.lines)+1)
	copy(newLines, b.lines[:row+1])
	newLines[row+1] = indent + after
	copy(newLines[row+2:], b.lines[row+1:])
	b.lines = newLines
	b.cursor.Row++
	b.cursor.Col = len(indent)
	b.modified = true
}

// Backspace deletes the character before the cursor (or the selection).
func (b *Buffer) Backspace() {
	if b.selActive {
		b.deleteSelection()
		return
	}
	b.checkpoint()
	row, col := b.cursor.Row, b.cursor.Col
	if col > 0 {
		line := b.lines[row]
		b.lines[row] = line[:col-1] + line[col:]
		b.cursor.Col--
	} else if row > 0 {
		prev := b.lines[row-1]
		b.cursor.Col = len(prev)
		b.lines[row-1] = prev + b.lines[row]
		b.lines = append(b.lines[:row], b.lines[row+1:]...)
		b.cursor.Row--
	}
	b.modified = true
}

// Delete deletes the character at the cursor (or the selection).
func (b *Buffer) Delete() {
	if b.selActive {
		b.deleteSelection()
		return
	}
	b.checkpoint()
	row, col := b.cursor.Row, b.cursor.Col
	line := b.lines[row]
	if col < len(line) {
		b.lines[row] = line[:col] + line[col+1:]
	} else if row < len(b.lines)-1 {
		b.lines[row] = line + b.lines[row+1]
		b.lines = append(b.lines[:row+1], b.lines[row+2:]...)
	}
	b.modified = true
}

func (b *Buffer) deleteSelection() {
	if !b.selActive {
		return
	}
	b.checkpoint()
	start, end := b.selAnchor, b.cursor
	if posLess(end, start) {
		start, end = end, start
	}
	if start.Row == end.Row {
		line := b.lines[start.Row]
		b.lines[start.Row] = line[:start.Col] + line[end.Col:]
	} else {
		first := b.lines[start.Row][:start.Col]
		last := b.lines[end.Row][end.Col:]
		b.lines[start.Row] = first + last
		b.lines = append(b.lines[:start.Row+1], b.lines[end.Row+1:]...)
	}
	b.cursor = start
	b.selActive = false
	b.modified = true
}

// InsertText inserts a multi-line string at the cursor (used for paste).
func (b *Buffer) InsertText(s string) {
	b.deleteSelection()
	b.checkpoint()
	parts := strings.Split(s, "\n")
	row, col := b.cursor.Row, b.cursor.Col
	line := b.lines[row]
	if len(parts) == 1 {
		b.lines[row] = line[:col] + s + line[col:]
		b.cursor.Col += len(s)
	} else {
		tail := line[col:]
		b.lines[row] = line[:col] + parts[0]
		newLines := make([]string, len(b.lines)+len(parts)-1)
		copy(newLines, b.lines[:row+1])
		for i := 1; i < len(parts)-1; i++ {
			newLines[row+i] = parts[i]
		}
		newLines[row+len(parts)-1] = parts[len(parts)-1] + tail
		copy(newLines[row+len(parts):], b.lines[row+1:])
		b.lines = newLines
		b.cursor.Row += len(parts) - 1
		b.cursor.Col = len(parts[len(parts)-1])
	}
	b.modified = true
}

// ReplaceMatch replaces the text at match m with replacement.
func (b *Buffer) ReplaceMatch(m Match, replacement string) {
	b.checkpoint()
	line := b.lines[m.Start.Row]
	b.lines[m.Start.Row] = line[:m.Start.Col] + replacement + line[m.End.Col:]
	b.cursor = Pos{m.Start.Row, m.Start.Col + len(replacement)}
	b.modified = true
}

// --- Cursor movement ---

func (b *Buffer) MoveLeft() {
	if b.cursor.Col > 0 {
		b.cursor.Col--
	} else if b.cursor.Row > 0 {
		b.cursor.Row--
		b.cursor.Col = len(b.lines[b.cursor.Row])
	}
}

func (b *Buffer) MoveRight() {
	line := b.lines[b.cursor.Row]
	if b.cursor.Col < len(line) {
		b.cursor.Col++
	} else if b.cursor.Row < len(b.lines)-1 {
		b.cursor.Row++
		b.cursor.Col = 0
	}
}

func (b *Buffer) MoveUp() {
	if b.cursor.Row > 0 {
		b.cursor.Row--
		b.cursor.Col = clamp(b.cursor.Col, 0, len(b.lines[b.cursor.Row]))
	}
}

func (b *Buffer) MoveDown() {
	if b.cursor.Row < len(b.lines)-1 {
		b.cursor.Row++
		b.cursor.Col = clamp(b.cursor.Col, 0, len(b.lines[b.cursor.Row]))
	}
}

func (b *Buffer) MoveHome() { b.cursor.Col = 0 }
func (b *Buffer) MoveEnd()  { b.cursor.Col = len(b.lines[b.cursor.Row]) }

func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_'
}

// MoveWordRight jumps to the start of the next word.
func (b *Buffer) MoveWordRight() {
	row, col := b.cursor.Row, b.cursor.Col
	line := b.lines[row]

	if col >= len(line) {
		if row < len(b.lines)-1 {
			b.cursor.Row++
			b.cursor.Col = 0
		}
		return
	}

	// Skip current character group (word, punctuation, or whitespace).
	if line[col] == ' ' || line[col] == '\t' {
		for col < len(line) && (line[col] == ' ' || line[col] == '\t') {
			col++
		}
	} else if isWordChar(line[col]) {
		for col < len(line) && isWordChar(line[col]) {
			col++
		}
		for col < len(line) && (line[col] == ' ' || line[col] == '\t') {
			col++
		}
	} else {
		for col < len(line) && !isWordChar(line[col]) && line[col] != ' ' && line[col] != '\t' {
			col++
		}
		for col < len(line) && (line[col] == ' ' || line[col] == '\t') {
			col++
		}
	}

	b.cursor.Col = col
}

// MoveWordLeft jumps to the start of the previous word.
func (b *Buffer) MoveWordLeft() {
	row, col := b.cursor.Row, b.cursor.Col

	if col == 0 {
		if row > 0 {
			b.cursor.Row--
			b.cursor.Col = len(b.lines[b.cursor.Row])
		}
		return
	}

	line := b.lines[row]
	col-- // step back one

	// Skip whitespace going left.
	for col > 0 && (line[col] == ' ' || line[col] == '\t') {
		col--
	}

	if col == 0 {
		b.cursor.Col = 0
		return
	}

	// Skip the word or punctuation group going left.
	if isWordChar(line[col]) {
		for col > 0 && isWordChar(line[col-1]) {
			col--
		}
	} else {
		for col > 0 && !isWordChar(line[col-1]) && line[col-1] != ' ' && line[col-1] != '\t' {
			col--
		}
	}

	b.cursor.Col = col
}

// DuplicateLine inserts a copy of the current line below and moves down.
func (b *Buffer) DuplicateLine() {
	b.checkpoint()
	row := b.cursor.Row
	line := b.lines[row]
	newLines := make([]string, len(b.lines)+1)
	copy(newLines, b.lines[:row+1])
	newLines[row+1] = line
	copy(newLines[row+2:], b.lines[row+1:])
	b.lines = newLines
	b.cursor.Row++
	b.cursor.Col = clamp(b.cursor.Col, 0, len(b.lines[b.cursor.Row]))
	b.modified = true
}

// MoveLineUp swaps the current line with the one above.
func (b *Buffer) MoveLineUp() {
	row := b.cursor.Row
	if row == 0 {
		return
	}
	b.checkpoint()
	b.lines[row], b.lines[row-1] = b.lines[row-1], b.lines[row]
	b.cursor.Row--
	b.cursor.Col = clamp(b.cursor.Col, 0, len(b.lines[b.cursor.Row]))
	b.modified = true
}

// MoveLineDown swaps the current line with the one below.
func (b *Buffer) MoveLineDown() {
	row := b.cursor.Row
	if row >= len(b.lines)-1 {
		return
	}
	b.checkpoint()
	b.lines[row], b.lines[row+1] = b.lines[row+1], b.lines[row]
	b.cursor.Row++
	b.cursor.Col = clamp(b.cursor.Col, 0, len(b.lines[b.cursor.Row]))
	b.modified = true
}

// DeleteToEndOfLine deletes from the cursor to the end of the line.
// If the cursor is already at end of line, it joins with the next line.
func (b *Buffer) DeleteToEndOfLine() {
	b.checkpoint()
	row, col := b.cursor.Row, b.cursor.Col
	line := b.lines[row]
	if col < len(line) {
		b.lines[row] = line[:col]
	} else if row < len(b.lines)-1 {
		b.lines[row] = line + b.lines[row+1]
		b.lines = append(b.lines[:row+1], b.lines[row+2:]...)
	}
	b.modified = true
}

func (b *Buffer) PageUp(height int) {
	b.cursor.Row = clamp(b.cursor.Row-height, 0, len(b.lines)-1)
	b.cursor.Col = clamp(b.cursor.Col, 0, len(b.lines[b.cursor.Row]))
	b.topLine = clamp(b.topLine-height, 0, len(b.lines)-1)
}

func (b *Buffer) PageDown(height int) {
	b.cursor.Row = clamp(b.cursor.Row+height, 0, len(b.lines)-1)
	b.cursor.Col = clamp(b.cursor.Col, 0, len(b.lines[b.cursor.Row]))
}
