package app

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ratan99/tp-disk-visualizer/internal/clean"
	"github.com/ratan99/tp-disk-visualizer/internal/scan"
)

// Server hosts the browser UI on 127.0.0.1. API calls need the random
// token from the launch URL, and the Host header is pinned to block DNS
// rebinding, since the API can move files to the Trash.
type Server struct {
	root    string
	opts    scan.Options
	version string
	token   string
	addr    string

	mu      sync.RWMutex
	scanner *scan.Scanner
	tree    *scan.Tree
	info    *Info
	types   []scan.TypeStat
	scanErr string
	freed   atomic.Int64
}

func Serve(root string, opts scan.Options, port int, noOpen bool, version string) error {
	tok := make([]byte, 16)
	if _, err := rand.Read(tok); err != nil {
		return err
	}
	s := &Server{root: root, opts: opts, version: version, token: hex.EncodeToString(tok)}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	s.addr = ln.Addr().String()
	url := fmt.Sprintf("http://%s/?t=%s", s.addr, s.token)

	go s.initialScan()
	fmt.Printf("\n  %s scanning %s\n  UI: %s\n", paint("1;36", "tpdv"), tildify(root), paint("4", url))
	if !noOpen {
		if err := OpenBrowser(url); err != nil {
			fmt.Fprintln(os.Stderr, "  could not open browser:", err)
		}
	}
	srv := &http.Server{Handler: s.routes(), ReadHeaderTimeout: 10 * time.Second}
	return srv.Serve(ln)
}

func (s *Server) initialScan() {
	sc := scan.NewScanner(s.opts)
	s.mu.Lock()
	s.scanner = sc
	s.mu.Unlock()

	res, err := sc.Scan(s.root)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.scanErr = err.Error()
		fmt.Fprintln(os.Stderr, "  scan failed:", err)
		return
	}
	s.tree = scan.NewTree(res.Root)
	s.info = NewInfo(res, s.opts, s.version)
	s.types = res.Types
	PrintSummary(os.Stdout, s.tree, s.info, 10)
	fmt.Println("  Browser UI is ready. Press Ctrl+C to quit.")
}

func (s *Server) current() (*scan.Tree, *Info) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tree, s.info
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /api/status", s.auth(s.handleStatus))
	mux.HandleFunc("GET /api/tree", s.auth(s.ready(s.handleTree)))
	mux.HandleFunc("GET /api/top", s.auth(s.ready(s.handleTop)))
	mux.HandleFunc("GET /api/types", s.auth(s.ready(s.handleTypes)))
	mux.HandleFunc("GET /api/report", s.auth(s.ready(s.handleReport)))
	mux.HandleFunc("GET /api/suggest", s.auth(s.ready(s.handleSuggest)))
	mux.HandleFunc("POST /api/trash", s.auth(s.ready(s.handleTrashMany)))
	mux.HandleFunc("POST /api/action", s.auth(s.ready(s.handleAction)))

	_, port, _ := net.SplitHostPort(s.addr)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.addr && r.Host != "localhost:"+port {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) validToken(r *http.Request) bool {
	t := r.Header.Get("X-TPDV-Token")
	if t == "" {
		t = r.URL.Query().Get("t")
	}
	return subtle.ConstantTimeCompare([]byte(t), []byte(s.token)) == 1
}

func (s *Server) auth(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.validToken(r) {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		fn(w, r)
	}
}

func (s *Server) ready(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if t, _ := s.current(); t == nil {
			http.Error(w, "scan in progress", http.StatusServiceUnavailable)
			return
		}
		fn(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if !s.validToken(r) {
		http.Error(w, "Open the URL printed in the terminal (it contains an access token).", http.StatusForbidden)
		return
	}
	page, err := renderPage(map[string]any{"mode": "live", "token": s.token, "version": s.version})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(page)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	sc, t, info, scanErr := s.scanner, s.tree, s.info, s.scanErr
	s.mu.RUnlock()
	resp := map[string]any{"freed": s.freed.Load(), "error": scanErr, "scanning": t == nil && scanErr == ""}
	if t == nil {
		if sc != nil {
			resp["progress"] = sc.Progress()
		}
		writeJSON(w, 200, resp)
		return
	}
	cur := *info
	t.RLock()
	cur.Total, cur.Files = t.Root.Size, t.Root.Files
	t.RUnlock()
	cur.DiskTotal, cur.DiskFree = diskUsage(s.root)
	resp["info"] = cur
	writeJSON(w, 200, resp)
}

func (s *Server) lookup(t *scan.Tree, r *http.Request) *scan.Node {
	v := r.URL.Query().Get("id")
	if v == "" {
		return t.Root
	}
	id, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return t.Get(id)
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	t, _ := s.current()
	t.RLock()
	defer t.RUnlock()
	n := s.lookup(t, r)
	if n == nil {
		writeJSON(w, 404, map[string]string{"error": "item no longer exists"})
		return
	}
	min := n.Size / 25000
	if min < 1 {
		min = 1
	}
	writeJSON(w, 200, map[string]any{
		"path":   n.Path(),
		"crumbs": Crumbs(n),
		"node":   Encode(n, 10, min),
		"total":  t.Root.Size,
	})
}

func (s *Server) handleTop(w http.ResponseWriter, r *http.Request) {
	t, _ := s.current()
	limit, _ := strconv.Atoi(r.URL.Query().Get("n"))
	if limit <= 0 || limit > 2000 {
		limit = 200
	}
	t.RLock()
	defer t.RUnlock()
	n := s.lookup(t, r)
	if n == nil {
		writeJSON(w, 404, map[string]string{"error": "item no longer exists"})
		return
	}
	writeJSON(w, 200, TopFiles(n, limit))
}

func (s *Server) handleTypes(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	types := s.types
	s.mu.RUnlock()
	writeJSON(w, 200, types)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	t, info := s.current()
	s.mu.RLock()
	types := s.types
	s.mu.RUnlock()
	page, err := RenderReport(t, *info, types)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", ReportName()))
	_, _ = w.Write(page)
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Op string `json:"op"`
		ID int    `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "bad request"})
		return
	}
	t, _ := s.current()
	t.RLock()
	n := t.Get(req.ID)
	var path string
	var size int64
	if n != nil {
		path, size = n.Path(), n.Size
	}
	t.RUnlock()
	if n == nil {
		writeJSON(w, 404, map[string]string{"error": "item no longer exists"})
		return
	}

	var err error
	resp := map[string]any{"ok": true}
	switch req.Op {
	case "reveal":
		err = Reveal(path)
	case "open":
		err = OpenPath(path)
	case "trash":
		if err = checkTrashable(path); err == nil && clean.NoTrash(path) {
			err = errors.New("use the suggested command instead of the Trash")
		}
		if err == nil {
			err = MoveToTrash(path)
		}
		if err == nil {
			t.Lock()
			if t.Get(req.ID) == n {
				t.Remove(n)
			}
			t.Unlock()
			resp["freed"] = size
			resp["totalFreed"] = s.freed.Add(size)
		}
	case "rescan":
		if !n.Dir {
			err = errors.New("only folders can be rescanned")
			break
		}
		var res *scan.Result
		if res, err = scan.NewScanner(s.opts).Scan(path); err == nil {
			t.Lock()
			if t.Get(req.ID) != n {
				err = errors.New("item changed while rescanning")
			} else {
				t.Replace(n, res.Root)
				resp["id"] = res.Root.ID
			}
			t.Unlock()
		}
	default:
		err = fmt.Errorf("unknown action %q", req.Op)
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, resp)
}

func (s *Server) handleSuggest(w http.ResponseWriter, r *http.Request) {
	t, _ := s.current()
	home, _ := os.UserHomeDir()
	writeJSON(w, 200, map[string]any{"items": clean.Find(t, SuggestMin), "home": home})
}

// handleTrashMany moves several items to the Trash in one Finder call.
func (s *Server) handleTrashMany(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeJSON(w, 400, map[string]string{"error": "bad request"})
		return
	}
	t, _ := s.current()

	type item struct {
		n    *scan.Node
		path string
	}
	var items []item
	var failed []map[string]string
	t.RLock()
	for _, id := range req.IDs {
		if n := t.Get(id); n != nil {
			items = append(items, item{n, n.Path()})
		}
	}
	t.RUnlock()
	// Drop items inside another selected item, and protected locations.
	selected := map[string]bool{}
	for _, it := range items {
		selected[it.path] = true
	}
	var paths []string
	var kept []item
	for _, it := range items {
		if insideAny(it.path, selected) {
			continue
		}
		err := checkTrashable(it.path)
		if err == nil && clean.NoTrash(it.path) {
			err = errors.New("use the suggested command instead of the Trash")
		}
		if err != nil {
			failed = append(failed, map[string]string{"p": it.path, "error": err.Error()})
			continue
		}
		kept = append(kept, it)
		paths = append(paths, it.path)
	}

	trashErr := MoveToTrashMany(paths)
	var freed int64
	trashed := 0
	t.Lock()
	for _, it := range kept {
		if _, err := os.Lstat(it.path); err == nil {
			msg := "still exists"
			if trashErr != nil {
				msg = trashErr.Error()
			}
			failed = append(failed, map[string]string{"p": it.path, "error": msg})
			continue
		}
		if t.Get(int(it.n.ID)) == it.n {
			freed += it.n.Size
			t.Remove(it.n)
		}
		trashed++
	}
	t.Unlock()
	writeJSON(w, 200, map[string]any{"ok": true, "trashed": trashed, "freed": freed, "totalFreed": s.freed.Add(freed), "failed": failed})
}

func insideAny(p string, set map[string]bool) bool {
	for d := filepath.Dir(p); d != p && d != "/" && d != "."; p, d = d, filepath.Dir(d) {
		if set[d] {
			return true
		}
	}
	return false
}

// checkTrashable refuses locations whose removal would break the system
// or the user account, whatever the UI asks for.
func checkTrashable(p string) error {
	p = filepath.Clean(p)
	home, _ := os.UserHomeDir()
	protected := p == "/" || p == home || filepath.Dir(p) == "/" ||
		(home != "" && p == filepath.Join(home, "Library"))
	for _, pre := range []string{"/System", "/usr", "/bin", "/sbin", "/private/etc", "/private/var/db"} {
		if p == pre || strings.HasPrefix(p, pre+"/") {
			protected = true
		}
	}
	if protected {
		return fmt.Errorf("refusing to trash %s: protected location", p)
	}
	return nil
}
