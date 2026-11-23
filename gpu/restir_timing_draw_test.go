//go:build draw && darwin

package gpu

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/set"
	staged "github.com/janearc/slayer/stage"
)

// TestRestirTiming prints milliseconds a frame for the demo sets. It
// draws nothing to keep and asserts nothing about speed; it is the
// measurement doc/restir.md and the report quote, run on purpose:
//
//	RESTIR_TIMING=1 RESTIR_SIZES=320x180,1280x720 RESTIR_COUNTS=1k,10k,100k \
//	  RESTIR_FRAMES=10 RESTIR_EXACT=1 go test -tags draw -run TestRestirTiming -v ./gpu
//
// The world is read from the demo set files, as a job reads them. The
// eye drifts a little each frame, so temporal reuse has a camera move to
// follow.
func TestRestirTiming(t *testing.T) {
	if os.Getenv("RESTIR_TIMING") == "" {
		t.Skip("set RESTIR_TIMING=1 to measure")
	}
	sizes := splitWords(os.Getenv("RESTIR_SIZES"), "320x180")
	counts := splitWords(os.Getenv("RESTIR_COUNTS"), "1k")
	frames, _ := strconv.Atoi(os.Getenv("RESTIR_FRAMES"))
	if frames < 2 {
		frames = 6
	}
	for _, count := range counts {
		raw, err := os.ReadFile("../sets/restir-crowd-" + count + ".yaml")
		if err != nil {
			t.Fatal(err)
		}
		made, err := set.Read("restir-crowd-"+count+".yaml", raw)
		if err != nil {
			t.Fatal(err)
		}
		said, _ := made.World()
		world, err := staged.World(said)
		if err != nil {
			t.Fatal(err)
		}
		for _, size := range sizes {
			var width, height int
			if _, err := fmt.Sscanf(size, "%dx%d", &width, &height); err != nil {
				t.Fatal(err)
			}
			lens := film.Pinhole{Wide: 60 * maths.Degree}
			eyeAt := func(frame int) maths.Subspace {
				from := maths.V(0.02*float64(frame), 6, 22)
				return maths.Subspace{Origin: from,
					Orientation: maths.LookAt(from, maths.V(0, 1.5, 0))}
			}
			drawer := NewRestir()
			var walls []time.Duration
			var last Profile
			for frame := range frames {
				_, profile, err := drawer.Draw(world, eyeAt(frame), lens, 0, width, height, 1)
				if errors.Is(err, metal.ErrNoDevice) {
					t.Skip("no gpu")
				}
				if err != nil {
					t.Fatal(err)
				}
				walls = append(walls, profile.Total)
				last = profile
				if frame == 0 || frame == frames-1 {
					t.Logf("restir %s %s frame %d: total %v encode %v compile %v accel %v tree %v passes wall %v gpu %v trace dispatch %v finish %v lamps %d",
						count, size, frame, profile.Total, profile.Encode, profile.Compile,
						profile.AccelerationWall, profile.RestirTreeWall, profile.RestirWall,
						profile.RestirGPU, profile.DispatchWall, profile.Finish, profile.RestirLamps)
				}
			}
			t.Logf("RESTIR %s %s: frame 0 %v, warm frames (1..%d) mean %v min %v; last: restir passes wall %v gpu %v, final trace dispatch %v",
				count, size, walls[0], frames-1, mean(walls[1:]), least(walls[1:]),
				last.RestirWall, last.RestirGPU, last.DispatchWall)
			if os.Getenv("RESTIR_EXACT") != "" {
				var exact []time.Duration
				for frame := range 3 {
					_, profile, err := SampledAccelerated(world, eyeAt(frame), lens, 0, width, height, 1)
					if err != nil {
						t.Logf("EXACT %s %s: %v", count, size, err)
						break
					}
					exact = append(exact, profile.Total)
					if frame == 2 {
						t.Logf("EXACT native %s %s: frame 0 %v, frames 1..2 mean %v; dispatch %v gpu %v",
							count, size, exact[0], mean(exact[1:]), profile.DispatchWall, profile.TraceGPU)
					}
				}
				if _, _, err := SampledFaster(world, eyeAt(0), lens, 0, width, height, 1); err != nil {
					t.Logf("FASTLIGHT (lamp-store, sky-looks) %s %s: %v", count, size, err)
				}
			}
		}
	}
}

func splitWords(text, fallback string) []string {
	if text == "" {
		text = fallback
	}
	var words []string
	word := ""
	for _, r := range text {
		if r == ',' {
			words = append(words, word)
			word = ""
			continue
		}
		word += string(r)
	}
	return append(words, word)
}

func mean(spans []time.Duration) time.Duration {
	var sum time.Duration
	for _, s := range spans {
		sum += s
	}
	return sum / time.Duration(len(spans))
}

func least(spans []time.Duration) time.Duration {
	low := spans[0]
	for _, s := range spans {
		low = min(low, s)
	}
	return low
}

// TestRestirBias prints, for a demo set, the mean light of restir (after
// several frames of a still camera) and of the exact native picture over
// the whole lit part of the picture, the ground and the objects. It is
// the check that restir loses or gains no light on average, run on
// purpose with RESTIR_BIAS=1.
func TestRestirBias(t *testing.T) {
	if os.Getenv("RESTIR_BIAS") == "" {
		t.Skip("set RESTIR_BIAS=1 to measure")
	}
	count := os.Getenv("RESTIR_COUNTS")
	if count == "" {
		count = "1k"
	}
	raw, err := os.ReadFile("../sets/restir-crowd-" + count + ".yaml")
	if err != nil {
		t.Fatal(err)
	}
	made, err := set.Read("x.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	said, _ := made.World()
	world, err := staged.World(said)
	if err != nil {
		t.Fatal(err)
	}
	const width, height = 320, 180
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	from := maths.V(0, 6, 22)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 1.5, 0))}
	exact, _, err := SampledAccelerated(world, eye, lens, 0, width, height, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, settings := range []struct {
		name string
		set  restirSettings
	}{{"initial, hidden dropped", restirSettings{Candidates: 4, Taps: 0, Radius: 16, Cap: 20, Visibility: true}},
		{"temporal, hidden dropped", restirSettings{Candidates: 4, Temporal: true, Taps: 0, Radius: 16, Cap: 20, Visibility: true}},
		{"all, hidden dropped", restirSettings{Candidates: 4, Temporal: true, Taps: 3, Radius: 16, Cap: 20, Visibility: true}}, {"initial candidates only", restirSettings{Candidates: 4, Taps: 0, Radius: 16, Cap: 20}},
		{"temporal", restirSettings{Candidates: 4, Temporal: true, Taps: 0, Radius: 16, Cap: 20}},
		{"spatial", restirSettings{Candidates: 4, Taps: 3, Radius: 16, Cap: 20}},
		{"temporal and spatial", defaultRestir()}} {
		drawer := &Restir{Settings: settings.set}
		var picture = exact
		for range 8 {
			picture, _, err = drawer.Draw(world, eye, lens, 0, width, height, 1)
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, region := range []struct {
			name           string
			x0, y0, x1, y1 int
		}{{"ground", 20, 140, 300, 175}, {"objects", 100, 80, 230, 115}, {"all below horizon", 0, 80, 320, 180}} {
			var as, bs []float64
			for y := region.y0; y < region.y1; y++ {
				for x := region.x0; x < region.x1; x++ {
					pa, pb := picture.At(x, y), exact.At(x, y)
					as = append(as, float64(pa.R)*0.2126+float64(pa.G)*0.7152+float64(pa.B)*0.0722)
					bs = append(bs, float64(pb.R)*0.2126+float64(pb.G)*0.7152+float64(pb.B)*0.0722)
				}
			}
			// the sums are ruled by a few pixels beside a lamp, which the
			// ratio of medians and of sums without the top one in a hundred
			// are not
			t.Logf("BIAS %s %s: restir/exact, sum %.3f, trimmed sum %.3f, median %.3f", settings.name, region.name,
				sumOf(as, 1)/sumOf(bs, 1), sumOf(as, 0.99)/sumOf(bs, 0.99), sumOf(as, 0.5001)/sumOf(bs, 0.5001))
		}
	}
}

// sumOf is the sum of the lowest share of the values.
func sumOf(values []float64, share float64) float64 {
	sorted := append([]float64(nil), values...)
	slices.Sort(sorted)
	var sum float64
	for _, v := range sorted[:int(float64(len(sorted))*share)] {
		sum += v
	}
	return sum
}
