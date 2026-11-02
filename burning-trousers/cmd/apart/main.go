// Command apart says how far apart two raws of one scene are, pixel by pixel,
// in just-noticeable differences: the share of pixels past one, how far apart
// the worst pixel is and where, and the budget's verdict.
//
// it is how a change is judged when there are two pictures, no test.
//
//	apart one.pfm other.pfm
//
// the raws are slayer's raw format, as the commands keep them with
// -raw. it gives back zero when the two were compared, one when they
// could not be, and two for arguments it cannot take.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/janearc/slayer/burning-trousers/divergence"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
)

// main is run, and the number it says.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run compares two raws and says how far apart they are.
func run(arguments []string, said, wrong io.Writer) int {
	if len(arguments) != 2 {
		fmt.Fprintln(wrong, "usage: apart one.pfm other.pfm")
		return 2
	}
	one, err := raw.Read(arguments[0])
	if err != nil {
		fmt.Fprintln(wrong, "apart:", err)
		return 1
	}
	other, err := raw.Read(arguments[1])
	if err != nil {
		fmt.Fprintln(wrong, "apart:", err)
		return 1
	}
	seen, err := divergence.Compare(one, other)
	if err != nil {
		fmt.Fprintln(wrong, "apart:", err)
		return 1
	}
	verdict := "outside"
	if seen.Within() {
		verdict = "within"
	}
	x, y := worstAt(one, other)
	fmt.Fprintf(said, "%.4f%% of %d pixels past one jnd, worst %.2f jnd "+
		"at %d, %d; %s the still budget\n", 100*float64(seen.Past(1)),
		one.W*one.H, float64(seen.Worst()), x, y, verdict)
	fmt.Fprintf(said, "there, the first is %v and the second %v\n",
		one.At(x, y).Light(), other.At(x, y).Light())
	return 0
}

// worstAt is where the two frames are furthest apart, the first such
// pixel row by row; the frames are the same size.
func worstAt(one, other *frame.Frame) (int, int) {
	at, worst := 0, divergence.JNDs(-1)
	for i := range one.Pix {
		if apart := divergence.Apart(one.Pix[i].Light(),
			other.Pix[i].Light()); apart > worst {
			at, worst = i, apart
		}
	}
	return at % one.W, at / one.W
}
