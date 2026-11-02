// Package fermat replays selected pixels of a raw CPU render and checks that
// their light matches bit for bit, recording the calculation steps.
//
// A hash of the raw image selects 64 pixels reproducibly. Each is recomputed
// independently through the full rendering path; a mismatch reports the pixel
// and the difference. This checks reproducibility, not an independent physical
// model of the scene.
//
// GPU approximations are compared with CPU references using tolerances; they
// are not subject to this exact replay check. The name recalls Fermat’s
// principle and asks which path the recorded light took.
package fermat

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand/v2"
	"os"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/trace"
)

// Many is the default number of hash-selected pixels checked per raw image.
const Many = 64

// Replay recomputes one pixel using the original scene, camera, lens, time,
// dimensions, and sampling, as trace.Replay does.
type Replay func(x, y int) (frame.Pixel, []trace.Step, error)

// Pixel records one selected location, saved RGB, replayed result, calculation
// steps, and any error encountered during replay.
type Pixel struct {
	X, Y  int
	Kept  frame.Pixel
	Made  frame.Pixel
	Steps []trace.Step
	Err   error
}

// Holds reports a successful replay with bitwise RGB agreement. PFM stores
// neither coverage nor depth, so they are not compared.
func (pixel Pixel) Holds() bool {
	if pixel.Err != nil {
		return false
	}
	bits := func(p frame.Pixel) [3]uint32 {
		return [3]uint32{math.Float32bits(p.R), math.Float32bits(p.G),
			math.Float32bits(p.B)}
	}
	return bits(pixel.Kept) == bits(pixel.Made)
}

// Proof records the raw path and SHA-256, dimensions, rendering request, and
// results for all selected pixels.
type Proof struct {
	Raw           string
	Sum           string
	Width, Height int
	Asked         string
	Pixels        []Pixel
}

// Holds reports whether every selected pixel matches.
func (proof Proof) Holds() bool {
	for _, pixel := range proof.Pixels {
		if !pixel.Holds() {
			return false
		}
	}
	return len(proof.Pixels) > 0
}

// Prove loads a raw image, replays explicitly named pixels, and selects
// additional pixels from its bytes. asked records the original request.
//
// Sampling detects widespread differences but can miss an isolated changed
// pixel unless named or selected. Unreadable raws and out-of-bounds named
// pixels return errors; replay failures are recorded in the proof.
func Prove(path, asked string, named [][2]int, many int,
	replay Replay) (Proof, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Proof{}, err
	}
	kept, err := raw.Read(path)
	if err != nil {
		return Proof{}, err
	}
	for _, at := range named {
		if at[0] < 0 || at[1] < 0 || at[0] >= kept.W || at[1] >= kept.H {
			return Proof{}, fmt.Errorf("fermat: pixel %d, %d is not in "+
				"%s, %d by %d", at[0], at[1], path, kept.W, kept.H)
		}
	}
	sum := sha256.Sum256(bytes)
	proof := Proof{Raw: path, Sum: hex.EncodeToString(sum[:]),
		Width: kept.W, Height: kept.H, Asked: asked}
	taken := map[[2]int]bool{}
	for _, at := range append(named, Picks(sum, kept.W, kept.H, many)...) {
		if taken[at] {
			continue
		}
		taken[at] = true
		pixel := Pixel{X: at[0], Y: at[1], Kept: kept.At(at[0], at[1])}
		pixel.Made, pixel.Steps, pixel.Err = replay(at[0], at[1])
		proof.Pixels = append(proof.Pixels, pixel)
	}
	return proof, nil
}

// Picks returns distinct coordinates selected from a generator seeded by sum,
// in selection order. Images with at most many pixels return every coordinate.
func Picks(sum [32]byte, width, height, many int) [][2]int {
	all := width * height
	if all <= 0 || many <= 0 {
		return nil
	}
	seed := rand.NewPCG(binary.LittleEndian.Uint64(sum[:8]),
		binary.LittleEndian.Uint64(sum[8:16]))
	order := rand.New(seed).Perm(all)
	if many < all {
		order = order[:many]
	}
	picks := make([][2]int, len(order))
	for i, place := range order {
		picks[i] = [2]int{place % width, place / width}
	}
	return picks
}

// String summarizes whether the proof holds and the number of checked pixels.
func (proof Proof) String() string {
	held := 0
	for _, pixel := range proof.Pixels {
		if pixel.Holds() {
			held++
		}
	}
	verdict := "holds"
	if !proof.Holds() {
		verdict = "fails"
	}
	return fmt.Sprintf("fermat: %s %s: %d of %d pixels made again to the "+
		"bit", proof.Raw, verdict, held, len(proof.Pixels))
}
