package maths_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

//go:embed testdata/segment-box.yaml
var segmentBoxRaw []byte

func TestSegmentBoxRetainsBothEndpointsAndMargin(test *testing.T) {
	var fixture struct {
		From     maths.Vec `json:"from"`
		To       maths.Vec `json:"to"`
		Margin   float64   `json:"margin"`
		Expected maths.Box `json:"expected"`
	}
	if err := yamlfile.Read("testdata/segment-box.yaml", segmentBoxRaw, &fixture); err != nil {
		test.Fatal(err)
	}
	forward := maths.SegmentBox(fixture.From, fixture.To, fixture.Margin)
	reverse := maths.SegmentBox(fixture.To, fixture.From, fixture.Margin)
	if forward != fixture.Expected || reverse != fixture.Expected {
		test.Fatal("segment bound changed with endpoint order", forward, reverse)
	}
}
