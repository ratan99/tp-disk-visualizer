//go:build !darwin

package app

import (
	"errors"
	"os/exec"
)

var errUnsupported = errors.New("not supported on this OS yet")

func OpenBrowser(url string) error { return exec.Command("xdg-open", url).Start() }

func Reveal(path string) error { return errUnsupported }

func OpenPath(path string) error { return exec.Command("xdg-open", path).Start() }

func MoveToTrash(path string) error { return errUnsupported }

func MoveToTrashMany(paths []string) error { return errUnsupported }
