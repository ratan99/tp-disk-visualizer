package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func OpenBrowser(url string) error { return exec.Command("open", url).Start() }

func Reveal(path string) error { return exec.Command("open", "-R", path).Run() }

func OpenPath(path string) error { return exec.Command("open", path).Run() }

// MoveToTrashMany trashes all paths with a single Finder call, falling
// back to one at a time if that fails (e.g. one item vanished).
func MoveToTrashMany(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	const script = `on run argv
set fs to {}
repeat with p in argv
set end of fs to (POSIX file (p as text) as alias)
end repeat
tell application "Finder" to delete fs
end run`
	if err := exec.Command("osascript", append([]string{"-e", script}, paths...)...).Run(); err == nil {
		return nil
	}
	var last error
	for _, p := range paths {
		if err := MoveToTrash(p); err != nil {
			last = err
		}
	}
	return last
}

// MoveToTrash asks Finder to trash the item, so "Put Back" works. If
// Finder automation is denied it falls back to renaming into ~/.Trash.
func MoveToTrash(path string) error {
	const script = `on run argv
set f to POSIX file (item 1 of argv) as alias
tell application "Finder" to delete f
end run`
	out, err := exec.Command("osascript", "-e", script, path).CombinedOutput()
	if err == nil {
		return nil
	}
	home, herr := os.UserHomeDir()
	if herr != nil {
		return fmt.Errorf("finder: %s", strings.TrimSpace(string(out)))
	}
	dst := filepath.Join(home, ".Trash", filepath.Base(path))
	if _, serr := os.Lstat(dst); serr == nil {
		dst += " " + time.Now().Format("15.04.05")
	}
	if rerr := os.Rename(path, dst); rerr != nil {
		return fmt.Errorf("finder: %s; fallback: %v", strings.TrimSpace(string(out)), rerr)
	}
	return nil
}
