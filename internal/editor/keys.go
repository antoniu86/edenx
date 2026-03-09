package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"edenx.dev/eden/internal/crypto"
	"github.com/gdamore/tcell/v2"
)

func (e *Editor) handleKey(ev *tcell.EventKey) {
	e.statusMsg = "" // clear transient status on any keypress
	switch e.mode {
	case ModeEdit:
		e.handleEditKey(ev)
	case ModeSearch:
		e.handleSearchKey(ev)
	case ModeReplace:
		e.handleReplaceKey(ev)
	case ModeNavigator:
		e.handleNavigatorKey(ev)
	case ModeFilenamePrompt:
		e.handleFilenamePromptKey(ev)
	case ModePasswordPrompt:
		e.handlePasswordPromptKey(ev)
	case ModeConfirmUnsaved:
		e.handleConfirmKey(ev)
	case ModeHelp:
		e.handleHelpKey(ev)
	case ModeJumpToLine:
		e.handleJumpToLineKey(ev)
	}
}

func (e *Editor) handleEditKey(ev *tcell.EventKey) {
	buf := e.currentBuffer()
	shift := ev.Modifiers()&tcell.ModShift != 0
	alt := ev.Modifiers()&tcell.ModAlt != 0
	ctrl := ev.Modifiers()&tcell.ModCtrl != 0

	// In read-only mode block all mutating keys before anything else.
	if buf.IsReadOnly() {
		k := ev.Key()
		isMutating := k == tcell.KeyRune || k == tcell.KeyEnter ||
			k == tcell.KeyBackspace || k == tcell.KeyBackspace2 ||
			k == tcell.KeyDelete || k == tcell.KeyTab ||
			k == tcell.KeyCtrlS || k == tcell.KeyCtrlE ||
			k == tcell.KeyCtrlZ || k == tcell.KeyCtrlY ||
			k == tcell.KeyCtrlX || k == tcell.KeyCtrlV ||
			k == tcell.KeyCtrlR || k == tcell.KeyCtrlD || k == tcell.KeyCtrlK ||
			(alt && (k == tcell.KeyUp || k == tcell.KeyDown))
		if isMutating {
			e.setStatus("[Read-only]")
			return
		}
	}

	// moveFn handles shift-selection: anchor on first shift+move, clear on plain move.
	moveFn := func(move func()) {
		if shift {
			if !buf.selActive {
				buf.StartSelection()
			}
			move()
		} else {
			buf.ClearSelection()
			move()
		}
	}

	switch ev.Key() {
		case tcell.KeyCtrlE:
		e.saveCurrentBufferEncrypted()

	case tcell.KeyRune:
		buf.ClearSelection()
		buf.InsertChar(ev.Rune())

	case tcell.KeyEnter:
		buf.InsertNewline()
		buf.ClearSelection()

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		buf.Backspace()

	case tcell.KeyDelete:
		buf.Delete()

	case tcell.KeyTab:
		buf.ClearSelection()
		if e.cfg.ExpandTabs {
			for i := 0; i < e.cfg.TabWidthOrDefault(); i++ {
				buf.InsertChar(' ')
			}
		} else {
			buf.InsertChar('\t')
		}

	case tcell.KeyUp:
		if alt {
			buf.ClearSelection()
			buf.MoveLineUp()
			return
		}
		moveFn(buf.MoveUp)
	case tcell.KeyDown:
		if alt {
			buf.ClearSelection()
			buf.MoveLineDown()
			return
		}
		moveFn(buf.MoveDown)

	case tcell.KeyLeft:
		if alt {
			e.prevTab()
			return
		}
		if ctrl {
			moveFn(buf.MoveWordLeft)
			return
		}
		moveFn(buf.MoveLeft)

	case tcell.KeyRight:
		if alt {
			e.nextTab()
			return
		}
		if ctrl {
			moveFn(buf.MoveWordRight)
			return
		}
		moveFn(buf.MoveRight)

	case tcell.KeyHome:
		moveFn(buf.MoveHome)
	case tcell.KeyEnd:
		moveFn(buf.MoveEnd)

	case tcell.KeyPgUp:
		_, h := e.screen.Size()
		moveFn(func() { buf.PageUp(h - 2) })
	case tcell.KeyPgDn:
		_, h := e.screen.Size()
		moveFn(func() { buf.PageDown(h - 2) })

	case tcell.KeyCtrlS:
		e.saveCurrentBuffer()

	case tcell.KeyCtrlQ:
		e.tryQuit()

	case tcell.KeyCtrlZ:
		if !buf.Undo() {
			e.setStatus("Nothing to undo")
		}

	case tcell.KeyCtrlY:
		if !buf.Redo() {
			e.setStatus("Nothing to redo")
		}

	case tcell.KeyCtrlA:
		buf.SelectAll()

	case tcell.KeyCtrlC:
		if text := buf.SelectedText(); text != "" {
			e.copy(text)
			e.setStatus("Copied")
		}

	case tcell.KeyCtrlX:
		if text := buf.SelectedText(); text != "" {
			e.copy(text)
			buf.deleteSelection()
			e.setStatus("Cut")
		}

	case tcell.KeyCtrlV:
		if text := e.paste(); text != "" {
			buf.InsertText(text)
		}

	case tcell.KeyCtrlF:
		e.searchTerm = ""
		e.matches = nil
		e.matchIdx = 0
		e.mode = ModeSearch

	case tcell.KeyCtrlR:
		e.searchTerm = ""
		e.replaceTerm = ""
		e.matches = nil
		e.matchIdx = 0
		e.searchField = 0
		e.mode = ModeReplace

	case tcell.KeyCtrlT:
		e.newTab()

	case tcell.KeyCtrlW:
		e.closeTab(e.activeIdx)

	case tcell.KeyCtrlN:
		e.openNavigator()

	case tcell.KeyF2:
		e.cycleTheme()

	case tcell.KeyF3:
		e.toggleReadOnly()

	case tcell.KeyF1:
		e.helpScroll = 0
		e.mode = ModeHelp

	case tcell.KeyCtrlD:
		buf.ClearSelection()
		buf.DuplicateLine()

	case tcell.KeyCtrlK:
		buf.ClearSelection()
		buf.DeleteToEndOfLine()

	case tcell.KeyCtrlJ:
		e.promptInput = ""
		e.mode = ModeJumpToLine
	}
}

func (e *Editor) handleJumpToLineKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		e.promptInput = ""
		e.mode = ModeEdit
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.promptInput) > 0 {
			e.promptInput = e.promptInput[:len(e.promptInput)-1]
		}
	case tcell.KeyEnter:
		line := 0
		for _, ch := range e.promptInput {
			if ch < '0' || ch > '9' {
				e.setError("Invalid line number")
				e.promptInput = ""
				e.mode = ModeEdit
				return
			}
			line = line*10 + int(ch-'0')
		}
		e.promptInput = ""
		e.mode = ModeEdit
		if line < 1 {
			line = 1
		}
		buf := e.currentBuffer()
		target := line - 1 // convert to 0-based
		buf.SetCursor(target, 0)
		e.setStatus(fmt.Sprintf("Jumped to line %d", line))
	case tcell.KeyRune:
		r := ev.Rune()
		if r >= '0' && r <= '9' {
			e.promptInput += string(r)
		}
	}
}

func (e *Editor) handleHelpKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyUp:
		if e.helpScroll > 0 {
			e.helpScroll--
		}
	case tcell.KeyDown:
		e.helpScroll++
	default:
		// Any other key closes the overlay.
		e.mode = ModeEdit
	}
}

func (e *Editor) prevTab() {
	if e.activeIdx > 0 {
		e.activeIdx--
	} else {
		e.activeIdx = len(e.buffers) - 1
	}
}

func (e *Editor) nextTab() {
	if e.activeIdx < len(e.buffers)-1 {
		e.activeIdx++
	} else {
		e.activeIdx = 0
	}
}

// --- Search mode ---

func (e *Editor) handleSearchKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		e.mode = ModeEdit
	case tcell.KeyUp:
		e.prevMatch()
	case tcell.KeyDown:
		e.nextMatch()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.searchTerm) > 0 {
			e.searchTerm = e.searchTerm[:len(e.searchTerm)-1]
			e.updateSearchMatches()
		}
	case tcell.KeyCtrlI: // toggle case sensitivity
		e.caseInsensitive = !e.caseInsensitive
		e.updateSearchMatches()
	case tcell.KeyCtrlG: // toggle regex mode
		e.regexMode = !e.regexMode
		e.updateSearchMatches()
	case tcell.KeyCtrlR:
		e.replaceTerm = ""
		e.searchField = 0
		e.mode = ModeReplace
	case tcell.KeyRune:
		e.searchTerm += string(ev.Rune())
		e.updateSearchMatches()
	}
}

// --- Replace mode ---

func (e *Editor) handleReplaceKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		e.mode = ModeEdit

	case tcell.KeyTab, tcell.KeyEnter:
		if e.searchField == 0 {
			e.updateSearchMatches()
			e.searchField = 1
		}

	case tcell.KeyCtrlG: // toggle regex mode
		e.regexMode = !e.regexMode
		e.updateSearchMatches()

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if e.searchField == 0 {
			if len(e.searchTerm) > 0 {
				e.searchTerm = e.searchTerm[:len(e.searchTerm)-1]
				e.updateSearchMatches()
			}
		} else {
			if len(e.replaceTerm) > 0 {
				e.replaceTerm = e.replaceTerm[:len(e.replaceTerm)-1]
			}
		}

	case tcell.KeyRune:
		r := ev.Rune()
		if e.searchField == 0 {
			e.searchTerm += string(r)
			e.updateSearchMatches()
			return
		}
		// In replace field: y/n/a act on matches; other keys extend replaceTerm.
		if len(e.matches) > 0 {
			switch r {
			case 'y':
				e.replaceCurrentMatch()
				return
			case 'n':
				e.nextMatch()
				return
			case 'a':
				e.replaceAllMatches()
				return
			}
		}
		e.replaceTerm += string(r)
	}
}

func (e *Editor) replaceCurrentMatch() {
	if len(e.matches) == 0 {
		return
	}
	m := e.matches[e.matchIdx]
	repl := e.expandReplacement(m)
	e.currentBuffer().ReplaceMatch(m, repl)
	e.updateSearchMatches()
}

func (e *Editor) replaceAllMatches() {
	buf := e.currentBuffer()
	// Collect fresh matches then replace back-to-front to preserve offsets.
	matches, _ := FindAll(buf.lines, e.searchTerm, e.caseInsensitive, e.regexMode)
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		repl := e.expandReplacement(m)
		buf.ReplaceMatch(m, repl)
	}
	count := len(matches)
	e.updateSearchMatches()
	e.mode = ModeEdit
	e.setStatus(fmt.Sprintf("Replaced %d occurrence(s)", count))
}

// expandReplacement returns the replacement string for match m, expanding
// regex capture groups ($1, $2 …) when regex mode is active.
func (e *Editor) expandReplacement(m Match) string {
	if !e.regexMode || e.searchRegex == nil {
		return e.replaceTerm
	}
	line := e.currentBuffer().lines[m.Start.Row]
	matched := line[m.Start.Col:m.End.Col]
	return RegexExpand(e.searchRegex, matched, e.replaceTerm)
}

// --- Navigator mode ---

func (e *Editor) handleNavigatorKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		e.mode = ModeEdit
		e.nav = nil
	case tcell.KeyUp:
		e.nav.MoveUp()
	case tcell.KeyDown:
		e.nav.MoveDown()
	case tcell.KeyEnter:
		path, isFile, err := e.nav.Enter()
		if err != nil {
			e.setError("Navigator: " + err.Error())
			return
		}
		if isFile {
			e.mode = ModeEdit
			e.nav = nil
			if err := e.OpenFile(path); err != nil {
				e.setError(err.Error())
			}
		}
		// If a directory was entered the navigator stays open with the new path.
	}
}

// --- Filename prompt ---

func (e *Editor) handleFilenamePromptKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		e.mode = ModeEdit
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.promptInput) > 0 {
			e.promptInput = e.promptInput[:len(e.promptInput)-1]
		}
	case tcell.KeyEnter:
		name := strings.TrimSpace(e.promptInput)
		if name == "" {
			e.mode = ModeEdit
			return
		}
		if e.saveAsEncrypt && !strings.HasSuffix(name, ".ednx") {
			name += ".ednx"
		}
		e.commitFilename(name)
	case tcell.KeyRune:
		e.promptInput += string(ev.Rune())
	}
}

// commitFilename checks for an existing file and either warns or saves directly.
func (e *Editor) commitFilename(name string) {
	buf := e.currentBuffer()

	doSave := func() {
		buf.filename = name
		if e.saveAsEncrypt {
			e.promptLabel = "Encryption password"
			e.promptSecret = true
			e.promptInput = ""
			e.pendingBuf = buf
			e.mode = ModePasswordPrompt
		} else {
			e.writePlain(buf)
			e.mode = ModeEdit
		}
	}

	if _, err := os.Stat(name); err == nil {
		// File exists — ask for confirmation before overwriting.
		e.confirmMsg = fmt.Sprintf(" %q already exists — overwrite? [y/n] ", name)
		e.confirmCallback = doSave
		e.confirmQuit = false
		e.mode = ModeConfirmUnsaved
		return
	}

	doSave()
}

// --- Password prompt ---

func (e *Editor) handlePasswordPromptKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		e.cancelPasswordPrompt()

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.promptInput) > 0 {
			e.promptInput = e.promptInput[:len(e.promptInput)-1]
		}

	case tcell.KeyEnter:
		if e.saveAsEncrypt && !e.promptConfirming {
			// First entry — store and ask for confirmation.
			e.promptFirstPass = []byte(e.promptInput)
			e.promptInput = ""
			e.promptConfirming = true
			return
		}

		if e.saveAsEncrypt && e.promptConfirming {
			// Confirmation step — compare with first entry.
			match := e.promptInput == string(e.promptFirstPass)
			// Zero the first-pass buffer regardless of outcome.
			for i := range e.promptFirstPass {
				e.promptFirstPass[i] = 0
			}
			password := e.promptInput
			e.promptInput = ""
			e.promptConfirming = false

			if !match {
				e.promptFirstPass = nil
				e.pendingBuf = nil
				e.setError("Passwords do not match — try again")
				e.mode = ModeEdit
				return
			}

			buf := e.pendingBuf
			e.pendingBuf = nil
			e.promptFirstPass = nil
			e.writeEncrypted(buf, password)
			buf.zeroPassword()
			buf.password = []byte(password)
			e.mode = ModeEdit
			return
		}

		// Decrypting a file on open — no confirmation needed.
		password := e.promptInput
		e.promptInput = ""
		buf := e.pendingBuf
		e.pendingBuf = nil

		plain, err := crypto.Decrypt(buf.pendingData, password)
		if err != nil {
			e.setError("Wrong password or corrupted file")
			if len(e.buffers) > 1 {
				e.removeTab(e.activeIdx)
			} else {
				e.buffers[0] = NewBuffer()
			}
			e.mode = ModeEdit
			return
		}
		buf.SetContent(string(plain))
		buf.modified = false
		buf.pendingData = nil
		buf.zeroPassword()
		buf.password = []byte(password)
		e.mode = ModeEdit
		e.setStatus("Decrypted: " + filepath.Base(buf.filename))

	case tcell.KeyRune:
		e.promptInput += string(ev.Rune())
	}
}

func (e *Editor) cancelPasswordPrompt() {
	for i := range e.promptFirstPass {
		e.promptFirstPass[i] = 0
	}
	e.promptFirstPass = nil
	e.promptConfirming = false
	e.promptInput = ""
	e.pendingBuf = nil
	e.mode = ModeEdit
}

// --- Confirm unsaved ---

func (e *Editor) handleConfirmKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'y', 'Y':
			e.confirmNoCallback = nil
			if e.confirmCallback != nil {
				cb := e.confirmCallback
				e.confirmCallback = nil
				cb()
			} else if e.confirmQuit {
				e.quit = true
			} else {
				e.removeTab(e.confirmTabIdx)
				e.mode = ModeEdit
			}
		case 'n', 'N':
			e.confirmCallback = nil
			if e.confirmNoCallback != nil {
				cb := e.confirmNoCallback
				e.confirmNoCallback = nil
				cb()
			}
			e.mode = ModeEdit
		}
	case tcell.KeyEscape:
		e.confirmCallback = nil
		e.confirmNoCallback = nil
		e.mode = ModeEdit
	}
}
