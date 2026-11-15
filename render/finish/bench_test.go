package finish

import (
	"testing"

	"github.com/janearc/slayer/ninjafilm"
	"github.com/janearc/slayer/render/frame"
)

// the size of a portrait short: what a stock is timed at.
const (
	shortWide = 1080
	shortHigh = 1920
)

// BenchmarkAStock is what developing one 1080 by 1920 frame costs, for
// each stock on the shelf. Nothing had timed a print before this.
func BenchmarkAStock(b *testing.B) {
	shelf, err := ninjafilm.Stocks()
	if err != nil {
		b.Fatal(err)
	}
	light := gradient(shortWide, shortHigh)
	for _, name := range shelf {
		made, err := stockDeveloper(name, 1, 0)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(name, func(b *testing.B) {
			for i := range b.N {
				if _, err := made.Develop(i, []*frame.Frame{light}); err !=
					nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkAPlainTone is what the plain tone costs at the same size, to
// set a stock's cost beside.
func BenchmarkAPlainTone(b *testing.B) {
	made := Developer{plain: fallback}
	light := gradient(shortWide, shortHigh)
	for range b.N {
		if _, err := made.Develop(0, []*frame.Frame{light}); err != nil {
			b.Fatal(err)
		}
	}
}
