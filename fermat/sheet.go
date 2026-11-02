package fermat

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/janearc/slayer/render/frame"
)

// Write puts the proof's sheet in dir: proof.md, which says whether it
// holds and lists every picked pixel, and a page of work for each, as
// pixel-X-Y.md.
func (proof Proof) Write(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, pixel := range proof.Pixels {
		path := filepath.Join(dir, pixel.page())
		work := []byte(pixel.work())
		if err := os.WriteFile(path, work, 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dir, "proof.md"),
		[]byte(proof.index()), 0o644)
}

// page is the name of a pixel's page of work.
func (pixel Pixel) page() string {
	return fmt.Sprintf("pixel-%d-%d.md", pixel.X, pixel.Y)
}

// index is proof.md: the verdict, the raw and how it was asked for, and
// each pixel, held or failed, with its page.
func (proof Proof) index() string {
	var page strings.Builder
	fmt.Fprintf(&page, "# fermat: a proof of %s\n\n",
		filepath.Base(proof.Raw))
	fmt.Fprintf(&page, "%s.\n\n", proof)
	fmt.Fprintf(&page, "- raw: %s, %d by %d\n", proof.Raw, proof.Width,
		proof.Height)
	fmt.Fprintf(&page, "- sha-256: %s\n", proof.Sum)
	fmt.Fprintf(&page, "- asked: %s\n\n", proof.Asked)
	page.WriteString("the pixels were picked by the raw's own sha-256. " +
		"each was made again alone, and holds if its light is the " +
		"raw's to the bit.\n\n")
	for i, pixel := range proof.Pixels {
		verdict := "holds"
		if !pixel.Holds() {
			verdict = "FAILS: " + pixel.why()
		}
		fmt.Fprintf(&page, "%d. pixel %d, %d: %s ([work](%s))\n", i+1,
			pixel.X, pixel.Y, verdict, pixel.page())
	}
	return page.String()
}

// why is how a pixel that fails differs from the raw.
func (pixel Pixel) why() string {
	if pixel.Err != nil {
		return "it could not be made: " + pixel.Err.Error()
	}
	return fmt.Sprintf("the raw kept %s, and it was made %s",
		light(pixel.Kept), light(pixel.Made))
}

// work is a pixel's page: whether it holds, then every step that made
// it, in order.
func (pixel Pixel) work() string {
	var page strings.Builder
	fmt.Fprintf(&page, "# pixel %d, %d\n\n", pixel.X, pixel.Y)
	if pixel.Holds() {
		fmt.Fprintf(&page, "holds: made again as %s, the raw's "+
			"light to the bit.\n\n", light(pixel.Made))
	} else {
		fmt.Fprintf(&page, "fails: %s.\n\n", pixel.why())
	}
	page.WriteString("every step that made it, in order. ray is which of " +
		"the pixel's rays, and mirror how many mirrors deep.\n\n")
	for i, step := range pixel.Steps {
		ray := fmt.Sprintf("ray %d", step.Ray)
		if step.Ray < 0 {
			ray = "all rays"
		}
		fmt.Fprintf(&page, "%d. %s, mirror %d: %s: %s\n", i+1, ray,
			step.Bounce, step.What, step.Said)
	}
	return page.String()
}

// light is a pixel's light in its shortest exact digits and its bits,
// so two that print alike but differ show where.
func light(pixel frame.Pixel) string {
	return fmt.Sprintf("rgb(%v, %v, %v) [%08x %08x %08x]", pixel.R,
		pixel.G, pixel.B, math.Float32bits(pixel.R),
		math.Float32bits(pixel.G), math.Float32bits(pixel.B))
}

// Mark draws a square about each picked pixel of a picture developed
// from the raw: green where it holds, red where it fails.
func (proof Proof) Mark(picture *image.RGBA) {
	held := color.RGBA{G: 220, A: 255}
	failed := color.RGBA{R: 255, A: 255}
	for _, pixel := range proof.Pixels {
		ink := held
		if !pixel.Holds() {
			ink = failed
		}
		square(picture, pixel.X, pixel.Y, 3, ink)
	}
}

// square draws the edge of a square of half side about x, y, leaving
// the pixel itself and what lies off the picture alone.
func square(picture *image.RGBA, x, y, half int, ink color.RGBA) {
	bounds := picture.Bounds()
	for i := -half; i <= half; i++ {
		edge := [][2]int{{x + i, y - half}, {x + i, y + half},
			{x - half, y + i}, {x + half, y + i}}
		for _, at := range edge {
			if (image.Point{X: at[0], Y: at[1]}).In(bounds) {
				picture.SetRGBA(at[0], at[1], ink)
			}
		}
	}
}
