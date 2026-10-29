package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ratan99/tp-disk-visualizer/internal/clean"
	"github.com/ratan99/tp-disk-visualizer/internal/scan"
	"github.com/ratan99/tp-disk-visualizer/web"
)

type Info struct {
	Root       string    `json:"root"`
	Total      int64     `json:"total"`
	Files      int64     `json:"files"`
	Dirs       int64     `json:"dirs"`
	Errors     int64     `json:"errors"`
	DurationMs int64     `json:"durationMs"`
	ScannedAt  time.Time `json:"scannedAt"`
	DiskTotal  uint64    `json:"diskTotal"`
	DiskFree   uint64    `json:"diskFree"`
	Version    string    `json:"version"`
	KeepMin    int64     `json:"keepMin"`
}

func NewInfo(res *scan.Result, opts scan.Options, version string) *Info {
	total, free := diskUsage(res.Root.Name)
	return &Info{
		Root:       res.Root.Name,
		Total:      res.Root.Size,
		Files:      res.Root.Files,
		Dirs:       res.Dirs,
		Errors:     res.Errors,
		DurationMs: res.Duration.Milliseconds(),
		ScannedAt:  time.Now(),
		DiskTotal:  total,
		DiskFree:   free,
		Version:    version,
		KeepMin:    opts.KeepMin,
	}
}

// renderPage injects the page config into the embedded UI. json.Marshal
// escapes <, > and &, so file names cannot break out of the script tag.
func renderPage(cfg any) ([]byte, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	page := strings.Replace(web.Index, "/*__TPDV__*/null", string(b), 1)
	return []byte(page), nil
}

type reportData struct {
	Info    Info               `json:"info"`
	Tree    *JNode             `json:"tree"`
	Top     []TopItem          `json:"top"`
	Types   []scan.TypeStat    `json:"types"`
	Suggest []clean.Suggestion `json:"suggest"`
	Home    string             `json:"home"`
}

// RenderReport builds a self-contained, read-only HTML report.
func RenderReport(t *scan.Tree, info Info, types []scan.TypeStat) ([]byte, error) {
	t.RLock()
	root := t.Root
	min := root.Size / 30000
	if min < 1 {
		min = 1
	}
	info.Total, info.Files = root.Size, root.Files
	data := reportData{Info: info, Tree: Encode(root, 64, min), Top: TopFiles(root, 500), Types: types}
	t.RUnlock()
	data.Suggest = clean.Find(t, SuggestMin)
	data.Home, _ = os.UserHomeDir()
	return renderPage(map[string]any{"mode": "report", "data": data})
}

func ReportName() string {
	return fmt.Sprintf("tpdv-report-%s.html", time.Now().Format("2006-01-02-1504"))
}
