package scan

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Node is a file or directory in the scanned tree. Files smaller than
// Options.KeepMin are not stored as nodes; they are folded into their
// directory's SmallSize/SmallCount to keep memory flat on huge trees.
type Node struct {
	ID         int32
	Name       string // base name; the root holds its absolute path
	Size       int64  // allocated bytes on disk (whole subtree for dirs)
	Files      int64  // number of files in the subtree
	SmallSize  int64  // bytes of direct child files below KeepMin
	SmallCount int64
	SmallExt   string // dominant extension (by bytes) among the small files
	Parent     *Node
	Children   []*Node
	Dir        bool
	Err        bool // directory could not be read
}

func (n *Node) Path() string {
	if n.Parent == nil {
		return n.Name
	}
	return filepath.Join(n.Parent.Path(), n.Name)
}

// Ext returns the lower-cased extension without the dot, or "".
func Ext(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i <= 0 || i == len(name)-1 || len(name)-i-1 > 12 {
		return ""
	}
	return strings.ToLower(name[i+1:])
}

func sortChildren(n *Node) {
	sort.Slice(n.Children, func(i, j int) bool { return n.Children[i].Size > n.Children[j].Size })
}

// finalize rolls directory sizes up the tree and sorts children by size.
func finalize(n *Node) {
	for _, c := range n.Children {
		if c.Dir {
			finalize(c)
			n.Size += c.Size
			n.Files += c.Files
		}
	}
	sortChildren(n)
}

// Tree indexes nodes by ID and supports in-place edits after a scan
// (removing trashed items, swapping in a rescanned subtree).
type Tree struct {
	sync.RWMutex
	Root  *Node
	index []*Node
}

func NewTree(root *Node) *Tree {
	t := &Tree{Root: root}
	t.register(root)
	return t
}

func (t *Tree) register(n *Node) {
	n.ID = int32(len(t.index))
	t.index = append(t.index, n)
	for _, c := range n.Children {
		t.register(c)
	}
}

func (t *Tree) unregister(n *Node) {
	if int(n.ID) < len(t.index) && t.index[n.ID] == n {
		t.index[n.ID] = nil
	}
	for _, c := range n.Children {
		t.unregister(c)
	}
}

func (t *Tree) Get(id int) *Node {
	if id < 0 || id >= len(t.index) {
		return nil
	}
	return t.index[id]
}

// Remove detaches n and subtracts its size from every ancestor.
func (t *Tree) Remove(n *Node) {
	p := n.Parent
	if p == nil {
		return
	}
	for i, c := range p.Children {
		if c == n {
			p.Children = append(p.Children[:i], p.Children[i+1:]...)
			break
		}
	}
	for a := p; a != nil; a = a.Parent {
		a.Size -= n.Size
		a.Files -= n.Files
		sortChildren(a)
	}
	t.unregister(n)
}

// Replace swaps old for a freshly scanned subtree of the same path.
func (t *Tree) Replace(old, fresh *Node) {
	fresh.Name = old.Name
	fresh.Parent = old.Parent
	t.unregister(old)
	t.register(fresh)
	p := old.Parent
	if p == nil {
		t.Root = fresh
		return
	}
	for i, c := range p.Children {
		if c == old {
			p.Children[i] = fresh
		}
	}
	dSize, dFiles := fresh.Size-old.Size, fresh.Files-old.Files
	for a := p; a != nil; a = a.Parent {
		a.Size += dSize
		a.Files += dFiles
		sortChildren(a)
	}
}
