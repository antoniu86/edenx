package editor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NavEntry represents a file or directory in the navigator
type NavEntry struct {
	Name  string
	IsDir bool
}

// Navigator holds the state for the file navigator overlay
type Navigator struct {
	Path    string
	Entries []NavEntry
	Idx     int
}

// NewNavigator creates a navigator starting in the current working directory.
func NewNavigator() (*Navigator, error) {
	return NewNavigatorAt("")
}

// NewNavigatorAt creates a navigator starting in startDir (cwd if empty).
func NewNavigatorAt(startDir string) (*Navigator, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	n := &Navigator{Path: startDir}
	if err := n.load(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *Navigator) load() error {
	entries, err := os.ReadDir(n.Path)
	if err != nil {
		return err
	}

	var result []NavEntry

	// Add ".." unless we're at root
	if n.Path != "/" {
		result = append(result, NavEntry{Name: "..", IsDir: true})
	}

	for _, e := range entries {
		// Skip hidden files (starting with .)
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		result = append(result, NavEntry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
		})
	}

	// Sort: directories first, then files, both alphabetically
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == ".." {
			return true
		}
		if result[j].Name == ".." {
			return false
		}
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	n.Entries = result
	n.Idx = 0
	return nil
}

func (n *Navigator) MoveUp() {
	if n.Idx > 0 {
		n.Idx--
	}
}

func (n *Navigator) MoveDown() {
	if n.Idx < len(n.Entries)-1 {
		n.Idx++
	}
}

// Enter navigates into a directory or returns the selected file path.
// Returns (path, isFile, error).
func (n *Navigator) Enter() (string, bool, error) {
	if len(n.Entries) == 0 {
		return "", false, nil
	}
	entry := n.Entries[n.Idx]
	if entry.IsDir {
		newPath := filepath.Join(n.Path, entry.Name)
		newPath = filepath.Clean(newPath)
		old := n.Path
		n.Path = newPath
		if err := n.load(); err != nil {
			n.Path = old
			return "", false, err
		}
		return "", false, nil
	}
	return filepath.Join(n.Path, entry.Name), true, nil
}
