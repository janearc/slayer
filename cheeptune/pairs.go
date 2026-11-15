package cheeptune

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// Pairs says which built-in scores and voicings go together, and what the
// ones that do not are refused for.
//
// A voicing is not a hat that fits any tune: it plays the roles that score
// has, at volumes set for that music, and a pair that does not fit is
// refused rather than clipped. That is worth being able to read rather than
// discover one peak at a time.
//
// It renders every pair. They go out across the cores, since each is its own
// piece and none of them needs another.
func Pairs(w io.Writer) error {
	scores, voicings := Builtin()

	// every pair is its own render and they have nothing to say to each
	// other, so they go out across the cores. the answers are put back
	// where they belong rather than as they arrive: a table that prints
	// in a different order each run is a table nobody can diff.
	type answer struct {
		score, line string
	}
	answers := make([]answer, len(scores)*len(voicings))
	var wg sync.WaitGroup
	for i, s := range scores {
		for j, v := range voicings {
			wg.Add(1)
			go func(at int, s, v string) {
				defer wg.Done()
				line := ""
				if p, err := Open(s, v); err != nil {
					line = fmt.Sprintf("%s: %s", v,
						why(err))
				} else {
					line = fmt.Sprintf("%s: %.3f", v,
						p.Peak)
				}
				answers[at] = answer{s, line}
			}(i*len(voicings)+j, s, v)
		}
	}
	wg.Wait()
	rows := map[string][]string{}
	for _, a := range answers {
		rows[a.score] = append(rows[a.score], a.line)
	}
	for _, s := range scores {
		fmt.Fprintf(w, "%s\n", s)
		sort.Strings(rows[s])
		for _, r := range rows[s] {
			fmt.Fprintf(w, "  %s\n", r)
		}
	}
	fmt.Fprintf(w, "\na number is the pair's peak, nought to one. "+
		"anything else is why it is refused.\n")
	return nil
}

// why is an error in one line, for a table.
func why(err error) string {
	s := strings.ReplaceAll(err.Error(), "\n", "; ")
	if i := strings.Index(s, ";"); i > 0 {
		s = s[:i] + " (and more)"
	}
	return s
}
