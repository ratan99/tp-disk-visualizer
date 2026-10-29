package app

import (
	"fmt"
	"io"
	"sort"

	"github.com/ratan99/tp-disk-visualizer/internal/clean"
)

var safetyLabels = []struct {
	safety clean.Safety
	title  string
	color  string
}{
	{clean.Safe, "Safe to delete — caches and build output, recreated automatically", "32"},
	{clean.Rebuild, "Rebuildable — recreated by reinstalling, rebuilding or re-downloading", "36"},
	{clean.Review, "Review first — may hold things you want", "33"},
}

// SuggestMin hides clean-up suggestions smaller than this.
const SuggestMin = 1 << 20

// PrintSuggestions lists clean-up candidates grouped by safety and rule.
// It never deletes anything.
func PrintSuggestions(w io.Writer, items []clean.Suggestion, perGroup int, root string) {
	var total int64
	for _, s := range items {
		total += s.Size
	}
	if len(items) == 0 {
		fmt.Fprintf(w, "  No clean-up suggestions here. Scan ~ to include caches in ~/Library and ~/.cache.\n\n")
		return
	}
	fmt.Fprintf(w, "  %s  up to %s in %d items (nothing has been deleted)\n",
		paint("1", "Clean-up suggestions"), paint("1;32", Human(total)), len(items))

	for _, lvl := range safetyLabels {
		type group struct {
			title, why, cmd string
			size            int64
			items           []clean.Suggestion
		}
		var groups []*group
		byTitle := map[string]*group{}
		var lvlSize int64
		for _, s := range items {
			if s.Safety != lvl.safety {
				continue
			}
			g := byTitle[s.Title]
			if g == nil {
				g = &group{title: s.Title, why: s.Why, cmd: s.Command}
				byTitle[s.Title] = g
				groups = append(groups, g)
			}
			g.size += s.Size
			g.items = append(g.items, s)
			lvlSize += s.Size
		}
		if len(groups) == 0 {
			continue
		}
		fmt.Fprintf(w, "\n  %s  %s\n", paint("1;"+lvl.color, Human(lvlSize)), paint("1", lvl.title))
		sort.SliceStable(groups, func(i, j int) bool { return groups[i].size > groups[j].size })
		for _, g := range groups {
			count := ""
			if len(g.items) > 1 {
				count = fmt.Sprintf(" (%d)", len(g.items))
			}
			fmt.Fprintf(w, "\n    %9s  %s%s\n", Human(g.size), paint("1", g.title), count)
			fmt.Fprintf(w, "               %s\n", paint("2", g.why))
			if g.cmd != "" {
				fmt.Fprintf(w, "               run: %s\n", paint(lvl.color, g.cmd))
			}
			for i, s := range g.items {
				if i == perGroup {
					fmt.Fprintf(w, "               %s\n", paint("2", fmt.Sprintf("… %d more", len(g.items)-perGroup)))
					break
				}
				fmt.Fprintf(w, "      %9s  %s\n", Human(s.Size), tildify(s.Path))
			}
		}
	}
	fmt.Fprintf(w, "\n  To review and move items to the Trash, open the browser UI: tpdv %s\n\n", tildify(root))
}
