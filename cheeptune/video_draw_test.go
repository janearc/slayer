//go:build draw

package cheeptune

import (
	"os"
	"testing"
)

func TestTheWorkedVideoExamplesRenderAndPassTheirSheets(t *testing.T) {
	for _, ex := range workedExamples {
		s, err := ReadScore(ex[0])
		if err != nil {
			t.Errorf("%s: %v", ex[0], err)
			continue
		}
		sheet, err := readCutsFile(ex[1], s)
		if err != nil {
			t.Errorf("%s: %v", ex[1], err)
			continue
		}
		p, err := Open(ex[0], sheet.Voicing)
		if err != nil {
			t.Errorf("%s with %s: %v", ex[0], sheet.Voicing, err)
			continue
		}
		if p.Peak >= 1 {
			t.Errorf("%s clips: %v", ex[0], p.Peak)
		}
		spans := sheet.Spans()
		if spans[0].Start != 0 || spans[len(spans)-1].End.Seconds() < 30 {
			t.Errorf("%s: spans run %v to %v", ex[0], spans[0].Start,
				spans[len(spans)-1].End)
		}
	}
}

// readCutsFile reads a sheet from a path outside the compiled-in set.
func readCutsFile(path string, s *Score) (*CutSheet, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseCuts(b, s)
}
