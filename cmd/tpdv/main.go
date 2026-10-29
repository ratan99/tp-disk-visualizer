// Command tpdv scans a folder and shows where the space went, either in
// the terminal, in an interactive browser treemap, or as an HTML report.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ratan99/tp-disk-visualizer/internal/app"
	"github.com/ratan99/tp-disk-visualizer/internal/clean"
	"github.com/ratan99/tp-disk-visualizer/internal/scan"
)

const version = "0.1.0"

const usage = `tpdv — fast disk space visualizer

Usage:
  tpdv [path]                  scan and open the interactive treemap in your browser (default: ~)
  tpdv ui [path] [flags]       same as above
  tpdv report [path] [flags]   write a self-contained HTML report
  tpdv top [path] [flags]      print the biggest items in the terminal
  tpdv suggest [path] [flags]  list caches & build folders that are safe to clean (deletes nothing)

Flags:
  -workers N     parallel directory readers (default 2×CPU)
  -min-file SZ   files smaller than this are grouped per folder (default 256K)
  -one-fs        stay on the filesystem of <path>
  -exclude LIST  comma-separated paths to skip
  -port N        ui: port to listen on (default: random)
  -no-open       ui: don't open the browser
  -o FILE        report: output file (default: tpdv-report-<date>.html)
  -open          report: open the report when done
  -n N           top/report/suggest: rows to print (default 15)

Examples:
  tpdv                       # your home folder in the browser
  tpdv /                     # whole disk (grant your terminal Full Disk Access)
  tpdv report ~/Library -open
  tpdv top ~/Downloads -n 30
  tpdv suggest               # what you can clean in ~ (node_modules, caches, Xcode, …)
`

func main() {
	args := os.Args[1:]
	cmd := "ui"
	if len(args) > 0 {
		switch args[0] {
		case "ui", "report", "top", "suggest":
			cmd, args = args[0], args[1:]
		case "help", "-h", "--help":
			fmt.Print(usage)
			return
		case "version", "-v", "--version":
			fmt.Println("tpdv", version)
			return
		}
	}
	if err := run(cmd, args); err != nil {
		fmt.Fprintln(os.Stderr, "tpdv:", err)
		os.Exit(1)
	}
}

func run(cmd string, args []string) error {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	workers := fs.Int("workers", 0, "")
	minFile := fs.String("min-file", "256K", "")
	oneFS := fs.Bool("one-fs", false, "")
	exclude := fs.String("exclude", "", "")
	port := fs.Int("port", 0, "")
	noOpen := fs.Bool("no-open", false, "")
	out := fs.String("o", "", "")
	openReport := fs.Bool("open", false, "")
	rows := fs.Int("n", 15, "")

	// Allow flags before or after the path.
	var paths []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		if fs.NArg() == 0 {
			break
		}
		paths = append(paths, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(paths) > 1 {
		return fmt.Errorf("expected one path, got %d", len(paths))
	}

	root := "~"
	if len(paths) == 1 {
		root = paths[0]
	}
	root, err := expand(root)
	if err != nil {
		return err
	}
	keep, err := app.ParseSize(*minFile)
	if err != nil {
		return err
	}
	opts := scan.Options{Workers: *workers, KeepMin: keep, OneFS: *oneFS}
	for _, e := range strings.Split(*exclude, ",") {
		if e = strings.TrimSpace(e); e != "" {
			if e, err = expand(e); err == nil {
				opts.Exclude = append(opts.Exclude, e)
			}
		}
	}

	switch cmd {
	case "ui":
		return app.Serve(root, opts, *port, *noOpen, version)

	case "suggest":
		res, err := app.ScanWithProgress(root, opts)
		if err != nil {
			return err
		}
		app.PrintSuggestions(os.Stdout, clean.Find(scan.NewTree(res.Root), app.SuggestMin), *rows, root)

	case "top", "report":
		res, err := app.ScanWithProgress(root, opts)
		if err != nil {
			return err
		}
		tree := scan.NewTree(res.Root)
		info := app.NewInfo(res, opts, version)
		app.PrintSummary(os.Stdout, tree, info, *rows)
		if cmd == "top" {
			return nil
		}
		page, err := app.RenderReport(tree, *info, res.Types)
		if err != nil {
			return err
		}
		dst := *out
		if dst == "" {
			dst = app.ReportName()
		}
		if err := os.WriteFile(dst, page, 0o644); err != nil {
			return err
		}
		abs, _ := filepath.Abs(dst)
		fmt.Printf("  Report written to %s (%s)\n\n", abs, app.Human(int64(len(page))))
		if *openReport {
			return app.OpenBrowser(abs)
		}
	}
	return nil
}

func expand(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[1:])
	}
	return filepath.Abs(p)
}
