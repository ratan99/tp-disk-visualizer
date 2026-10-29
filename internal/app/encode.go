package app

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/ratan99/tp-disk-visualizer/internal/scan"
)

// Node kinds sent to the browser.
const (
	kindFile  = 0
	kindDir   = 1
	kindSmall = 2 // aggregated small files of one folder
	kindRest  = 3 // aggregated items too small to draw
)

type JNode struct {
	I int32    `json:"i"`
	N string   `json:"n"`
	S int64    `json:"s"`
	F int64    `json:"f,omitempty"`
	T uint8    `json:"t"`
	E string   `json:"e,omitempty"`
	H bool     `json:"h,omitempty"` // has children not included
	X bool     `json:"x,omitempty"` // unreadable
	K []*JNode `json:"k,omitempty"`
}

type Crumb struct {
	I int32  `json:"i"`
	N string `json:"n"`
}

type TopItem struct {
	I int32  `json:"i"`
	N string `json:"n"`
	P string `json:"p"` // parent folder
	S int64  `json:"s"`
	E string `json:"e,omitempty"`
}

// Encode returns n down to depth levels, dropping (and aggregating)
// anything smaller than min bytes so the payload stays proportional to
// what can actually be drawn.
func Encode(n *scan.Node, depth int, min int64) *JNode {
	j := &JNode{I: n.ID, N: n.Name, S: n.Size, F: n.Files}
	if !n.Dir {
		j.E = scan.Ext(n.Name)
		return j
	}
	j.T = kindDir
	j.X = n.Err
	if depth == 0 {
		j.H = len(n.Children) > 0 || n.SmallCount > 0
		return j
	}
	var restSize, restCount, restFiles int64
	for _, c := range n.Children {
		if c.Size >= min && c.Size > 0 {
			j.K = append(j.K, Encode(c, depth-1, min))
		} else {
			restSize += c.Size
			restFiles += c.Files
			restCount++
		}
	}
	if n.SmallCount > 0 {
		if n.SmallSize >= min {
			j.K = append(j.K, &JNode{I: -1, N: fmt.Sprintf("%s small files", commas(n.SmallCount)), S: n.SmallSize, F: n.SmallCount, T: kindSmall, E: n.SmallExt})
		} else {
			restSize += n.SmallSize
			restFiles += n.SmallCount
			restCount += n.SmallCount
		}
	}
	if restCount > 0 && restSize > 0 {
		j.K = append(j.K, &JNode{I: -1, N: fmt.Sprintf("%s smaller items", commas(restCount)), S: restSize, F: restFiles, T: kindRest})
	}
	sort.SliceStable(j.K, func(a, b int) bool { return j.K[a].S > j.K[b].S })
	return j
}

func Crumbs(n *scan.Node) []Crumb {
	var out []Crumb
	for x := n; x != nil; x = x.Parent {
		out = append([]Crumb{{I: x.ID, N: x.Name}}, out...)
	}
	return out
}

// TopFiles returns the largest stored files under n.
func TopFiles(n *scan.Node, limit int) []TopItem {
	var files []*scan.Node
	stack := []*scan.Node{n}
	for len(stack) > 0 {
		x := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, c := range x.Children {
			if c.Dir {
				stack = append(stack, c)
			} else {
				files = append(files, c)
			}
		}
	}
	if !n.Dir {
		files = []*scan.Node{n}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Size > files[j].Size })
	if len(files) > limit {
		files = files[:limit]
	}
	out := make([]TopItem, len(files))
	for i, f := range files {
		out[i] = TopItem{I: f.ID, N: f.Name, P: filepath.Dir(f.Path()), S: f.Size, E: scan.Ext(f.Name)}
	}
	return out
}
