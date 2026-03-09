package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"edenx.dev/eden/internal/config"
	"edenx.dev/eden/internal/crypto"
	"github.com/gdamore/tcell/v2"
)

// Mode is the current editor input state.
type Mode int

const (
	ModeEdit            Mode = iota
	ModeSearch               // Ctrl+F: typing search term
	ModeReplace              // Ctrl+R: search and replace
	ModeNavigator            // file navigator overlay
	ModeFilenamePrompt       // save-as filename prompt
	ModePasswordPrompt       // encrypt/decrypt password prompt
	ModeConfirmUnsaved       // unsaved changes warning
	ModeHelp                 // F1: keyboard shortcut reference
	ModeJumpToLine           // Ctrl+J: jump to line number
)

// Editor is the top-level application struct.
type Editor struct {
	screen tcell.Screen
	cfg    *config.Config
	theme  Theme

	buffers   []*Buffer
	activeIdx int

	mode Mode

	// Search / Replace state
	searchTerm      string
	replaceTerm     string
	matches         []Match
	matchIdx        int
	caseInsensitive bool
	regexMode       bool
	searchRegex     *regexp.Regexp // compiled regex when regexMode is active
	wrapped         bool
	searchField     int // 0 = search field, 1 = replace field

	// Navigator
	nav *Navigator

	// Prompt state (filename or password)
	promptLabel      string
	promptSecret     bool
	promptInput      string
	saveAsEncrypt    bool
	pendingBuf       *Buffer // buffer awaiting password for open/save
	promptConfirming bool    // true when re-entering password for confirmation
	promptFirstPass  []byte  // holds first password entry during confirmation step

	// Confirm state
	confirmMsg      string
	confirmTabIdx   int
	confirmQuit     bool
	confirmCallback   func() // called when user presses 'y'
	confirmNoCallback func() // called when user presses 'n'

	// Help overlay scroll
	helpScroll int

	// Clipboard
	clipboard string

	// Status bar message
	statusMsg   string
	statusError bool

	quit bool
}

// New creates and initialises the Editor.
func New(cfg *config.Config) (*Editor, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	screen.EnableMouse()
	e := &Editor{
		screen:    screen,
		cfg:       cfg,
		theme:     GetTheme(cfg.Theme),
		buffers:   []*Buffer{NewBuffer()},
		activeIdx: 0,
	}
	return e, nil
}

// Close shuts down the terminal screen and zeros all cached passwords.
func (e *Editor) Close() {
	for _, buf := range e.buffers {
		buf.zeroPassword()
	}
	if e.screen != nil {
		e.screen.Fini()
	}
}

// Run is the main event loop.
func (e *Editor) Run() error {
	for !e.quit {
		e.draw()
		ev := e.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			e.screen.Sync()
		case *tcell.EventKey:
			e.handleKey(ev)
		case *tcell.EventMouse:
			e.handleMouse(ev)
		}
	}
	return nil
}

// NewEmptyBuffer adds or reuses an empty unnamed buffer.
func (e *Editor) NewEmptyBuffer() {
	buf := e.currentBuffer()
	if buf.modified || buf.filename != "" || (len(buf.lines) == 1 && buf.lines[0] != "") {
		e.buffers = append(e.buffers, NewBuffer())
		e.activeIdx = len(e.buffers) - 1
	}
}

// OpenFile opens a path into a buffer (reuses pristine current buf or creates new tab).
func (e *Editor) OpenFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", path, err)
	}

	cur := e.currentBuffer()
	var buf *Buffer
	if !cur.modified && cur.filename == "" && len(cur.lines) == 1 && cur.lines[0] == "" {
		buf = cur
	} else {
		buf = NewBuffer()
		e.buffers = append(e.buffers, buf)
		e.activeIdx = len(e.buffers) - 1
	}

	if crypto.IsEncrypted(data) {
		buf.filename = path
		buf.encrypted = true
		buf.pendingData = data
		e.promptLabel = "Password for " + filepath.Base(path)
		e.promptSecret = true
		e.promptInput = ""
		e.pendingBuf = buf
		e.saveAsEncrypt = false
		e.mode = ModePasswordPrompt
		return nil
	}

	buf.SetContent(string(data))
	buf.filename = path
	buf.modified = false
	return nil
}

// --- Internal helpers ---

func (e *Editor) currentBuffer() *Buffer {
	return e.buffers[e.activeIdx]
}

func (e *Editor) newTab() {
	e.buffers = append(e.buffers, NewBuffer())
	e.activeIdx = len(e.buffers) - 1
}

func (e *Editor) closeTab(idx int) {
	if e.buffers[idx].modified {
		e.confirmTabIdx = idx
		e.confirmQuit = false
		e.confirmMsg = fmt.Sprintf(" Unsaved changes in %q — close anyway? [y/n] ", e.buffers[idx].DisplayName())
		e.mode = ModeConfirmUnsaved
		return
	}
	e.removeTab(idx)
}

func (e *Editor) removeTab(idx int) {
	e.buffers[idx].zeroPassword()
	if len(e.buffers) == 1 {
		e.buffers[0] = NewBuffer()
		return
	}
	e.buffers = append(e.buffers[:idx], e.buffers[idx+1:]...)
	if e.activeIdx >= len(e.buffers) {
		e.activeIdx = len(e.buffers) - 1
	}
}

func (e *Editor) saveCurrentBuffer() {
	buf := e.currentBuffer()
	if buf.filename == "" {
		e.promptLabel = "Save as"
		e.promptSecret = false
		e.promptInput = ""
		e.saveAsEncrypt = false
		e.mode = ModeFilenamePrompt
		return
	}
	if buf.encrypted {
		if len(buf.password) > 0 {
			// Password already known — ask whether to keep or change it.
			e.confirmMsg = fmt.Sprintf(" %q — change encryption password? [y/n] ", filepath.Base(buf.filename))
			e.confirmCallback = func() {
				// y → prompt for a new password.
				e.promptLabel = "New password for " + filepath.Base(buf.filename)
				e.promptSecret = true
				e.promptInput = ""
				e.pendingBuf = buf
				e.saveAsEncrypt = true
				e.mode = ModePasswordPrompt
			}
			e.confirmNoCallback = func() {
				// n → reuse cached password, save silently.
				e.writeEncrypted(buf, string(buf.password))
			}
			e.confirmQuit = false
			e.mode = ModeConfirmUnsaved
			return
		}
		// No cached password yet — ask for one.
		e.promptLabel = "Password to save " + filepath.Base(buf.filename)
		e.promptSecret = true
		e.promptInput = ""
		e.pendingBuf = buf
		e.saveAsEncrypt = true
		e.mode = ModePasswordPrompt
		return
	}
	e.writePlain(buf)
}

func (e *Editor) saveCurrentBufferEncrypted() {
	buf := e.currentBuffer()
	if buf.filename == "" || !strings.HasSuffix(buf.filename, ".ednx") {
		// Suggest a .ednx name derived from the current filename.
		suggested := ""
		if buf.filename != "" {
			base := strings.TrimSuffix(filepath.Base(buf.filename), filepath.Ext(buf.filename))
			suggested = base + ".ednx"
		}
		e.promptLabel = "Save encrypted as (original file kept)"
		e.promptSecret = false
		e.promptInput = suggested
		e.saveAsEncrypt = true
		e.mode = ModeFilenamePrompt
		return
	}
	e.promptLabel = "Encryption password"
	e.promptSecret = true
	e.promptInput = ""
	e.pendingBuf = buf
	e.saveAsEncrypt = true
	e.mode = ModePasswordPrompt
}

func (e *Editor) writePlain(buf *Buffer) {
	if err := os.WriteFile(buf.filename, []byte(buf.Content()), 0644); err != nil {
		e.setError("Save failed: " + err.Error())
		return
	}
	buf.modified = false
	e.setStatus("Saved: " + filepath.Base(buf.filename))
}

func (e *Editor) writeEncrypted(buf *Buffer, password string) {
	enc, err := crypto.Encrypt([]byte(buf.Content()), password)
	if err != nil {
		e.setError("Encryption failed: " + err.Error())
		return
	}
	if err := os.WriteFile(buf.filename, enc, 0644); err != nil {
		e.setError("Save failed: " + err.Error())
		return
	}
	buf.modified = false
	buf.encrypted = true
	e.setStatus("Saved encrypted: " + filepath.Base(buf.filename))
}

// --- Clipboard ---

func (e *Editor) copy(text string) {
	e.clipboard = text
	for _, cmd := range [][]string{
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
	} {
		c := exec.Command(cmd[0], cmd[1:]...)
		c.Stdin = strings.NewReader(text)
		if err := c.Run(); err == nil {
			return
		}
	}
}

func (e *Editor) paste() string {
	for _, cmd := range [][]string{
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
	} {
		out, err := exec.Command(cmd[0], cmd[1:]...).Output()
		if err == nil && len(out) > 0 {
			return string(out)
		}
	}
	return e.clipboard
}

// --- Read-only ---

// MarkAllReadOnly sets every open buffer to read-only (used for the -r flag).
func (e *Editor) MarkAllReadOnly() {
	for _, buf := range e.buffers {
		buf.readonly = true
	}
}

func (e *Editor) toggleReadOnly() {
	buf := e.currentBuffer()
	buf.readonly = !buf.readonly
	if buf.readonly {
		e.setStatus("Read-only enabled")
	} else {
		e.setStatus("Read-only disabled")
	}
}

// --- Theme ---

func (e *Editor) cycleTheme() {
	e.cfg.Theme = NextTheme(e.cfg.Theme)
	e.theme = GetTheme(e.cfg.Theme)
	_ = e.cfg.Save()
	e.setStatus("Theme: " + e.cfg.Theme)
}

// --- Search ---

func (e *Editor) updateSearchMatches() {
	matches, err := e.currentBuffer().FindMatches(e.searchTerm, e.caseInsensitive, e.regexMode)
	if err != nil {
		e.searchRegex = nil
		e.matches = nil
		e.matchIdx = 0
		e.setError("Invalid regex: " + err.Error())
		return
	}
	// Cache compiled regex for use in replacement expansion.
	if e.regexMode && e.searchTerm != "" {
		prefix := ""
		if e.caseInsensitive {
			prefix = "(?i)"
		}
		e.searchRegex, _ = regexp.Compile(prefix + e.searchTerm)
	} else {
		e.searchRegex = nil
	}
	e.matches = matches
	e.matchIdx = 0
	e.wrapped = false
	if len(e.matches) > 0 {
		cur := e.currentBuffer().cursor
		for i, m := range e.matches {
			if m.Start.Row > cur.Row || (m.Start.Row == cur.Row && m.Start.Col >= cur.Col) {
				e.matchIdx = i
				break
			}
		}
		e.jumpToMatch(e.matchIdx)
	}
}

func (e *Editor) jumpToMatch(idx int) {
	if len(e.matches) == 0 {
		return
	}
	m := e.matches[idx]
	e.currentBuffer().SetCursor(m.Start.Row, m.Start.Col)
}

func (e *Editor) nextMatch() {
	if len(e.matches) == 0 {
		return
	}
	next := (e.matchIdx + 1) % len(e.matches)
	if next <= e.matchIdx {
		e.wrapped = true
	}
	e.matchIdx = next
	e.jumpToMatch(e.matchIdx)
}

func (e *Editor) prevMatch() {
	if len(e.matches) == 0 {
		return
	}
	prev := (e.matchIdx - 1 + len(e.matches)) % len(e.matches)
	if prev >= e.matchIdx {
		e.wrapped = true
	}
	e.matchIdx = prev
	e.jumpToMatch(e.matchIdx)
}

// --- Navigator ---

func (e *Editor) openNavigator() {
	startDir := ""
	if e.currentBuffer().filename != "" {
		startDir = filepath.Dir(e.currentBuffer().filename)
	}
	nav, err := NewNavigatorAt(startDir)
	if err != nil {
		e.setError("Cannot open navigator: " + err.Error())
		return
	}
	e.nav = nav
	e.mode = ModeNavigator
}

// --- Status ---

func (e *Editor) setStatus(msg string) {
	e.statusMsg = msg
	e.statusError = false
}

func (e *Editor) setError(msg string) {
	e.statusMsg = msg
	e.statusError = true
}

// --- Helpers ---

func (e *Editor) tryQuit() {
	var unsaved []string
	for _, buf := range e.buffers {
		if buf.modified {
			unsaved = append(unsaved, buf.DisplayName())
		}
	}
	if len(unsaved) == 0 {
		e.quit = true
		return
	}
	e.confirmQuit = true
	e.confirmMsg = fmt.Sprintf(" Unsaved: %s — quit anyway? [y/n] ", strings.Join(unsaved, ", "))
	e.mode = ModeConfirmUnsaved
}
