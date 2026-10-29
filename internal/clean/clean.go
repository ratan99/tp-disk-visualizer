// Package clean finds folders in a scanned tree that are usually safe to
// remove: caches, build output and re-downloadable dependencies. It only
// suggests; deleting is up to the caller.
package clean

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ratan99/tp-disk-visualizer/internal/scan"
)

type Safety string

const (
	Safe    Safety = "safe"    // caches and build output, recreated automatically
	Rebuild Safety = "rebuild" // recreated by running a command (install, build, download)
	Review  Safety = "review"  // may hold things you want; look before deleting
)

var safetyRank = map[Safety]int{Safe: 0, Rebuild: 1, Review: 2}

type Suggestion struct {
	ID       int32  `json:"i"`
	ParentID int32  `json:"pi"`
	Path     string `json:"p"`
	Name     string `json:"n"`
	Size     int64  `json:"s"`
	Files    int64  `json:"f"`
	Dir      bool   `json:"d"`
	Title    string `json:"title"`
	Safety   Safety `json:"safety"`
	Why      string `json:"why"`
	Command  string `json:"cmd,omitempty"`
	NoTrash  bool   `json:"noTrash,omitempty"` // use Command instead of the Trash
	Modified int64  `json:"mtime,omitempty"`   // project marker mtime (unix s)
}

type rule struct {
	title   string
	safety  Safety
	why     string
	cmd     string
	noTrash bool

	// Name rules: a folder with this name, confirmed by a marker file in
	// its parent (e.g. package.json next to node_modules) or inside it.
	markers []string // globs allowed
	inside  []string

	// Path rules: an exact location ("~/..." is the home folder).
	path     string
	children bool     // suggest each child separately
	exts     []string // suggest matching files instead (and keep walking)
	minFile  int64
}

var (
	nameRules = map[string][]*rule{}
	pathRules []*rule
)

func byName(r *rule, names ...string) {
	for _, n := range names {
		nameRules[n] = append(nameRules[n], r)
	}
}

func atPath(rules ...*rule) { pathRules = append(pathRules, rules...) }

type finder struct {
	byPath map[string]*rule
	min    int64
	out    []Suggestion
}

// Find walks t and returns suggestions of at least min bytes, safest
// first and largest first within each safety level.
func Find(t *scan.Tree, min int64) []Suggestion {
	home, _ := os.UserHomeDir()
	f := &finder{byPath: map[string]*rule{}, min: min}
	for _, r := range pathRules {
		p := r.path
		if strings.HasPrefix(p, "~/") {
			if home == "" {
				continue
			}
			p = filepath.Join(home, p[2:])
		}
		f.byPath[p] = r
	}
	t.RLock()
	f.walk(t.Root, t.Root.Name)
	t.RUnlock()
	sort.SliceStable(f.out, func(i, j int) bool {
		a, b := f.out[i], f.out[j]
		if safetyRank[a.Safety] != safetyRank[b.Safety] {
			return safetyRank[a.Safety] < safetyRank[b.Safety]
		}
		return a.Size > b.Size
	})
	return f.out
}

// NoTrash reports whether p is a location that must be cleaned with its
// tool's own command (deleting it directly would break the tool).
func NoTrash(p string) bool {
	home, _ := os.UserHomeDir()
	for _, r := range pathRules {
		if r.noTrash && home != "" && strings.HasPrefix(r.path, "~/") && p == filepath.Join(home, r.path[2:]) {
			return true
		}
	}
	return false
}

func (f *finder) walk(n *scan.Node, p string) {
	if r := f.byPath[p]; r != nil && f.applyPath(r, n, p) {
		return
	}
	if n.Parent != nil {
		for _, r := range nameRules[n.Name] {
			if ok, mtime := r.matches(p); ok {
				f.emit(n, p, r, mtime)
				return // never look inside a match (nested node_modules, …)
			}
		}
	}
	for _, c := range n.Children {
		if c.Dir {
			f.walk(c, filepath.Join(p, c.Name))
		}
	}
}

// applyPath handles a path rule; it reports whether the subtree is done.
func (f *finder) applyPath(r *rule, n *scan.Node, p string) bool {
	switch {
	case r.children:
		for _, c := range n.Children {
			cp := filepath.Join(p, c.Name)
			if c.Dir && f.byPath[cp] != nil {
				f.walk(c, cp) // a more specific rule exists for this child
				continue
			}
			f.emit(c, cp, r, 0)
		}
		return true
	case len(r.exts) > 0:
		for _, c := range n.Children {
			if !c.Dir && c.Size >= r.minFile && hasExt(c.Name, r.exts) {
				f.emit(c, filepath.Join(p, c.Name), r, 0)
			}
		}
		return false
	case n.Parent == nil:
		return false // never suggest the scan root itself
	default:
		f.emit(n, p, r, 0)
		return true
	}
}

func (f *finder) emit(n *scan.Node, p string, r *rule, mtime int64) {
	if n.Size < f.min || n.Size == 0 {
		return
	}
	pid := int32(-1)
	if n.Parent != nil {
		pid = n.Parent.ID
	}
	f.out = append(f.out, Suggestion{
		ID: n.ID, ParentID: pid, Path: p, Name: n.Name, Size: n.Size, Files: n.Files, Dir: n.Dir,
		Title: r.title, Safety: r.safety, Why: r.why, Command: r.cmd, NoTrash: r.noTrash, Modified: mtime,
	})
}

func (r *rule) matches(p string) (bool, int64) {
	if len(r.inside) > 0 {
		return findMarker(p, r.inside)
	}
	if len(r.markers) > 0 {
		return findMarker(filepath.Dir(p), r.markers)
	}
	return true, 0
}

func findMarker(dir string, markers []string) (bool, int64) {
	for _, m := range markers {
		path := filepath.Join(dir, m)
		if strings.ContainsAny(m, "*?[") {
			matches, _ := filepath.Glob(path)
			if len(matches) == 0 {
				continue
			}
			path = matches[0]
		}
		if fi, err := os.Stat(path); err == nil {
			return true, fi.ModTime().Unix()
		}
	}
	return false, 0
}

func hasExt(name string, exts []string) bool {
	e := scan.Ext(name)
	for _, x := range exts {
		if e == x {
			return true
		}
	}
	return false
}
