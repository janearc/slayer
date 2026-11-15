package finish

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/ninjafilm"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// picturesFrom and picturesTo are the environment's way of asking the
// demonstration below to run: a pfm of a frame's light, and the
// directory the developed pictures are written in. Absent, it skips.
const (
	picturesFrom = "FINISH_PFM"
	picturesTo   = "FINISH_PICTURES"
)

// TestWriteEveryStocksPicture develops the light of a frame through
// every stock on the shelf and through the plain tone, and writes each
// picture as a png. It is how a stock is looked at; with FINISH_STOPS set
// it also writes each stock a few stops either side of its rating.
func TestWriteEveryStocksPicture(t *testing.T) {
	from, to := os.Getenv(picturesFrom), os.Getenv(picturesTo)
	if from == "" || to == "" {
		t.Skipf("set %s and %s to develop a frame through every stock",
			picturesFrom, picturesTo)
	}
	light, err := raw.Read(from)
	if err != nil {
		t.Fatal(err)
	}
	shelf, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range shelf {
		for _, rated := range ratings() {
			write(t, filepath.Join(to, name+rated.suffix+".png"), name,
				rated.stops, light)
		}
	}
	write(t, filepath.Join(to, "plain.png"), "", 0, light)
}

// rating is a number of stops and how a file that holds it is named.
type rating struct {
	stops  float64
	suffix string
}

// ratings are the stops the demonstration tries: the rating itself, then
// a stop and two stops either side, as FINISH_STOPS says; one rating
// when it says nothing.
func ratings() []rating {
	if os.Getenv("FINISH_STOPS") == "" {
		return []rating{{0, ""}}
	}
	return []rating{{-2, "-under2"}, {-1, "-under1"}, {0, ""},
		{1, "-over1"}, {2, "-over2"}, {3, "-over3"}, {4, "-over4"}}
}

// write develops the light through a stock, at its rating plus stops, or
// through the plain tone when no stock is named, and writes the png.
func write(t *testing.T, path, name string, stops float64,
	light *frame.Frame) {
	t.Helper()
	made := Developer{plain: colour.Tone{Knee: 0.7, Top: 8}}
	if name != "" {
		var err error
		if made, err = stockDeveloper(name, 1921, stops); err != nil {
			t.Fatal(err)
		}
	}
	picture, err := made.Develop(0, []*frame.Frame{light})
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, picture); err != nil {
		t.Fatal(err)
	}
}
