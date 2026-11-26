package regionset

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestRectanglesEndingAtTheFrameEdgeAreAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		width := rapid.IntRange(1, 10000).Draw(t, "frame width")
		height := rapid.IntRange(1, 10000).Draw(t, "frame height")
		left := rapid.IntRange(0, width-1).Draw(t, "left")
		top := rapid.IntRange(0, height-1).Draw(t, "top")
		text := fmt.Sprintf("job: job.yaml\nthreshold: 1\nregions:\n"+
			"  - name: view\n    observation: detail\n    left: %d\n"+
			"    top: %d\n    width: %d\n    height: %d\n",
			left, top, width-left, height-top)
		got, err := Read(strings.NewReader(text), width, height)
		if err != nil || len(got.Regions) != 1 {
			t.Fatalf("edge rectangle: %+v, %v", got, err)
		}
		outside := strings.Replace(text, fmt.Sprintf("width: %d\n", width-left),
			fmt.Sprintf("width: %d\n", width-left+1), 1)
		if _, err := Read(strings.NewReader(outside), width, height); err == nil {
			t.Fatalf("accepted rectangle beyond frame: %s", outside)
		}
	})
}
