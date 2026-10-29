package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ratan99/tp-disk-visualizer/internal/scan"
)

// Human formats bytes in decimal units, matching Finder.
func Human(b int64) string {
	if b < 1000 {
		return fmt.Sprintf("%d B", b)
	}
	units := []string{"kB", "MB", "GB", "TB", "PB"}
	f := float64(b)
	i := -1
	for f >= 1000 && i < len(units)-1 {
		f /= 1000
		i++
	}
	switch {
	case f >= 100:
		return fmt.Sprintf("%.0f %s", f, units[i])
	case f >= 10:
		return fmt.Sprintf("%.1f %s", f, units[i])
	}
	return fmt.Sprintf("%.2f %s", f, units[i])
}

func commas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 0 {
		return "-" + commas(-n)
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// ParseSize accepts 512, 64K, 1.5M, 2G (binary multiples).
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.TrimSuffix(strings.TrimSuffix(s, "B"), "I")
	mult := float64(1)
	if s != "" {
		switch s[len(s)-1] {
		case 'K':
			mult = 1 << 10
		case 'M':
			mult = 1 << 20
		case 'G':
			mult = 1 << 30
		}
		if mult > 1 {
			s = s[:len(s)-1]
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return int64(f * mult), nil
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

var color = isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == ""

func paint(code, s string) string {
	if !color {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func tildify(p string) string {
	if home, err := os.UserHomeDir(); err == nil && (p == home || strings.HasPrefix(p, home+"/")) {
		return "~" + p[len(home):]
	}
	return p
}

func truncMiddle(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	h := (n - 1) / 2
	return string(r[:h]) + "…" + string(r[len(r)-(n-1-h):])
}

func bar(frac float64, width int) string {
	full := int(frac*float64(width) + 0.5)
	if full > width {
		full = width
	}
	return paint("36", strings.Repeat("█", full)) + paint("2", strings.Repeat("░", width-full))
}

// ScanWithProgress runs a scan, drawing a live status line on stderr.
func ScanWithProgress(root string, opts scan.Options) (*scan.Result, error) {
	sc := scan.NewScanner(opts)
	done := make(chan struct{})
	var wg sync.WaitGroup
	if isTerminal(os.Stderr) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := time.NewTicker(120 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-done:
					fmt.Fprint(os.Stderr, "\r\033[K")
					return
				case <-t.C:
					p := sc.Progress()
					fmt.Fprintf(os.Stderr, "\r\033[K  scanning  %s files  %s  %s",
						commas(p.Files), Human(p.Bytes), truncMiddle(tildify(p.Current), 50))
				}
			}
		}()
	}
	res, err := sc.Scan(root)
	close(done)
	wg.Wait()
	return res, err
}

// PrintSummary writes the biggest folders and files to w.
func PrintSummary(w io.Writer, t *scan.Tree, info *Info, n int) {
	t.RLock()
	defer t.RUnlock()
	root := t.Root
	secs := float64(info.DurationMs) / 1000
	rate := ""
	if secs > 0 {
		rate = fmt.Sprintf(" (%s files/s)", commas(int64(float64(info.Files)/secs)))
	}
	fmt.Fprintf(w, "\n  %s  %s in %s files, %s folders · scanned in %.1fs%s\n",
		paint("1", tildify(root.Name)), paint("1;36", Human(root.Size)), commas(root.Files), commas(info.Dirs), secs, rate)
	if info.DiskTotal > 0 {
		fmt.Fprintf(w, "  Disk: %s free of %s\n", Human(int64(info.DiskFree)), Human(int64(info.DiskTotal)))
	}
	if info.Errors > 0 {
		fmt.Fprintf(w, "  %s\n", paint("33", fmt.Sprintf("%s items couldn't be read — grant Full Disk Access to your terminal to include them", commas(info.Errors))))
	}

	fmt.Fprintf(w, "\n  %s\n", paint("1", "Largest items"))
	for i, c := range root.Children {
		if i >= n {
			break
		}
		name := c.Name
		if c.Dir {
			name += "/"
		}
		frac := 0.0
		if root.Size > 0 {
			frac = float64(c.Size) / float64(root.Size)
		}
		fmt.Fprintf(w, "  %9s  %s %4.0f%%  %s\n", Human(c.Size), bar(frac, 20), frac*100, name)
	}

	fmt.Fprintf(w, "\n  %s\n", paint("1", "Largest files"))
	for _, f := range TopFiles(root, n) {
		fmt.Fprintf(w, "  %9s  %s\n", Human(f.S), tildify(filepath.Join(f.P, f.N)))
	}
	fmt.Fprintln(w)
}
