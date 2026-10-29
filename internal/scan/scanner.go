package scan

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Options struct {
	Workers int      // max concurrent directory readers (default 2×CPU)
	KeepMin int64    // files below this size are aggregated per directory
	OneFS   bool     // do not descend into other mounted filesystems
	Exclude []string // absolute paths to skip
}

type TypeStat struct {
	Ext   string `json:"e"`
	Size  int64  `json:"s"`
	Count int64  `json:"c"`
}

type Progress struct {
	Files     int64  `json:"files"`
	Dirs      int64  `json:"dirs"`
	Bytes     int64  `json:"bytes"`
	Errors    int64  `json:"errors"`
	Current   string `json:"current"`
	ElapsedMs int64  `json:"elapsedMs"`
}

type Result struct {
	Root     *Node
	Files    int64
	Dirs     int64
	Errors   int64
	Duration time.Duration
	Types    []TypeStat
}

// entry is one directory entry as reported by the platform reader. name
// is only valid for the duration of the readDir callback.
type entry struct {
	name   []byte
	isDir  bool
	isLink bool
	err    bool
	size   int64 // allocated bytes
	dev    uint64
	ino    uint64
	nlink  uint32
}

type Scanner struct {
	opts    Options
	sem     chan struct{}
	wg      sync.WaitGroup
	excl    map[string]bool
	rootDev uint64
	root    string

	files, dirs, bytes, errs atomic.Int64
	current                  atomic.Pointer[string]
	start                    atomic.Int64

	inoMu  sync.Mutex
	inodes map[[2]uint64]struct{}

	typMu sync.Mutex
	types map[string]*[2]int64
}

func NewScanner(o Options) *Scanner {
	if o.Workers <= 0 {
		o.Workers = runtime.NumCPU() * 2 // more just adds kernel lock contention
	}
	return &Scanner{
		opts:   o,
		sem:    make(chan struct{}, o.Workers),
		inodes: map[[2]uint64]struct{}{},
		types:  map[string]*[2]int64{},
	}
}

func (s *Scanner) Progress() Progress {
	p := Progress{
		Files:  s.files.Load(),
		Dirs:   s.dirs.Load(),
		Bytes:  s.bytes.Load(),
		Errors: s.errs.Load(),
	}
	if c := s.current.Load(); c != nil {
		p.Current = *c
	}
	if st := s.start.Load(); st != 0 {
		p.ElapsedMs = time.Since(time.Unix(0, st)).Milliseconds()
	}
	return p
}

func (s *Scanner) Scan(root string) (*Result, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	s.root = abs
	s.rootDev, _ = devOf(abs)
	s.excl = defaultExcludes()
	for _, e := range s.opts.Exclude {
		if a, err := filepath.Abs(e); err == nil {
			s.excl[a] = true
		}
	}
	delete(s.excl, abs) // never exclude the thing we were asked to scan

	start := time.Now()
	s.start.Store(start.UnixNano())
	rootNode := &Node{Name: abs, Dir: true}
	s.wg.Add(1)
	s.scanDir(rootNode, abs)
	s.wg.Wait()
	finalize(rootNode)

	types := make([]TypeStat, 0, len(s.types))
	for k, v := range s.types {
		types = append(types, TypeStat{Ext: k, Size: v[0], Count: v[1]})
	}
	sort.Slice(types, func(i, j int) bool { return types[i].Size > types[j].Size })

	return &Result{
		Root:     rootNode,
		Files:    s.files.Load(),
		Dirs:     s.dirs.Load(),
		Errors:   s.errs.Load(),
		Duration: time.Since(start),
		Types:    types,
	}, nil
}

func joinPath(dir string, name []byte) string {
	if dir == "/" {
		return "/" + string(name)
	}
	return dir + "/" + string(name)
}

func (s *Scanner) seenInode(dev, ino uint64) bool {
	k := [2]uint64{dev, ino}
	s.inoMu.Lock()
	defer s.inoMu.Unlock()
	if _, ok := s.inodes[k]; ok {
		return true
	}
	s.inodes[k] = struct{}{}
	return false
}

// scanDir reads one directory, then fans its subdirectories out to free
// workers, recursing inline when all workers are busy (so it never blocks).
func (s *Scanner) scanDir(n *Node, path string) {
	defer s.wg.Done()
	if s.opts.OneFS && path != s.root {
		if d, err := devOf(path); err == nil && d != s.rootDev {
			return
		}
	}
	s.current.Store(&path)

	var (
		subdirs  []*Node
		subpaths []string
		types    map[string]*[3]int64 // size, count, bytes below KeepMin
		files    int64
		total    int64
		extBuf   [16]byte
	)
	err := readDir(path, func(e *entry) {
		if e.err {
			s.errs.Add(1)
			return
		}
		if e.isDir {
			p := joinPath(path, e.name)
			if s.excl[p] {
				return
			}
			subdirs = append(subdirs, &Node{Name: string(e.name), Dir: true, Parent: n})
			subpaths = append(subpaths, p)
			return
		}
		if e.isLink {
			return
		}
		size := e.size
		if e.nlink > 1 && s.seenInode(e.dev, e.ino) {
			size = 0 // hard link already counted
		}
		files++
		total += size
		if types == nil {
			types = map[string]*[3]int64{}
		}
		k := extKey(e.name, &extBuf)
		t := types[string(k)]
		if t == nil {
			t = new([3]int64)
			types[string(k)] = t
		}
		t[0] += size
		t[1]++
		if size >= s.opts.KeepMin {
			n.Children = append(n.Children, &Node{Name: string(e.name), Size: size, Files: 1, Parent: n})
		} else {
			n.SmallSize += size
			n.SmallCount++
			t[2] += size
		}
	})
	if err != nil {
		n.Err = true
		s.errs.Add(1)
	}
	n.Size += total
	n.Files += files
	n.Children = append(n.Children, subdirs...)
	s.files.Add(files)
	s.bytes.Add(total)
	s.dirs.Add(1)

	if types != nil {
		var best int64
		for k, v := range types {
			if v[2] > best {
				best, n.SmallExt = v[2], k
			}
		}
		s.typMu.Lock()
		for k, v := range types {
			g := s.types[k]
			if g == nil {
				g = new([2]int64)
				s.types[k] = g
			}
			g[0] += v[0]
			g[1] += v[1]
		}
		s.typMu.Unlock()
	}

	for i, c := range subdirs {
		s.wg.Add(1)
		select {
		case s.sem <- struct{}{}:
			go func(c *Node, p string) {
				s.scanDir(c, p)
				<-s.sem
			}(c, subpaths[i])
		default:
			s.scanDir(c, subpaths[i])
		}
	}
}

// extKey lower-cases the extension into buf without allocating.
func extKey(name []byte, buf *[16]byte) []byte {
	i := bytes.LastIndexByte(name, '.')
	if i <= 0 || i == len(name)-1 || len(name)-i-1 > 12 {
		return nil
	}
	e := name[i+1:]
	for j, c := range e {
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		buf[j] = c
	}
	return buf[:len(e)]
}
