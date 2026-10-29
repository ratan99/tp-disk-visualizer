package clean

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ratan99/tp-disk-visualizer/internal/scan"
)

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFind(t *testing.T) {
	root := t.TempDir()
	const mb = 1 << 20
	write(t, filepath.Join(root, "web/package.json"), 2)
	write(t, filepath.Join(root, "web/node_modules/react/index.js"), 2*mb)
	write(t, filepath.Join(root, "web/node_modules/a/node_modules/b/x.js"), 2*mb) // nested: must not be listed twice
	write(t, filepath.Join(root, "web/.next/cache/x"), 2*mb)
	write(t, filepath.Join(root, "py/.venv/pyvenv.cfg"), 2)
	write(t, filepath.Join(root, "py/.venv/lib/x"), 2*mb)
	write(t, filepath.Join(root, "rs/Cargo.toml"), 2)
	write(t, filepath.Join(root, "rs/target/debug/app"), 2*mb)
	write(t, filepath.Join(root, "docs/build/thesis.pdf"), 2*mb)   // no Gradle file: keep
	write(t, filepath.Join(root, "notes/node_modules/x"), 2*mb)    // no package.json: keep
	write(t, filepath.Join(root, "data/venv/important.csv"), 2*mb) // no pyvenv.cfg: keep

	res, err := scan.NewScanner(scan.Options{KeepMin: 1 << 18}).Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Safety{}
	for _, s := range Find(scan.NewTree(res.Root), 1) {
		rel, _ := filepath.Rel(res.Root.Name, s.Path)
		got[rel] = s.Safety
	}
	want := map[string]Safety{
		"web/node_modules": Rebuild,
		"web/.next":        Safe,
		"py/.venv":         Rebuild,
		"rs/target":        Rebuild,
	}
	for p, s := range want {
		if got[p] != s {
			t.Errorf("%s: got %q, want %q", p, got[p], s)
		}
	}
	if len(got) != len(want) {
		t.Errorf("unexpected suggestions: %v", got)
	}
}

func TestNoTrash(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	if !NoTrash(filepath.Join(home, "go/pkg/mod")) {
		t.Error("go module cache must be cleaned with go clean -modcache")
	}
	if NoTrash(filepath.Join(home, ".npm/_cacache")) {
		t.Error("npm cache can go to the Trash")
	}
}
