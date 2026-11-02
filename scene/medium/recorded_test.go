package medium

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// the recording: what the media computed, at ad2fff9, held bit for bit.
//
// each line is a medium, a method and a digest of every number the method gave
// over its fixed inputs.
//
// it was written from slayer's copy while the copy was still the original in
// every number: its code the original's but for comments and guards on inputs
// that are no number.
//
// those guards were checked against the original over 400,000 points, no
// difference (the commit that says so, "deceit sweep 3"). SLAYER_RECORD=1
// writes it again; nothing else should.
//
// it holds the code to the recording, so a change moves nothing; it is not
// evidence that the media are right, which a recording never is. the audit says
// that, against physics and arithmetic by hand.
const recording = "testdata/recorded.txt"

// recorded is every medium the recording holds, by name: each kind, with
// the dressings and the edges its numbers can take.
func recorded() map[string]Medium {
	puff := Puff{Radius: 2, Thick: 1.5}
	return map[string]Medium{
		"air":            Air{Radius: 10, Height: 1.2, Thick: 0.8},
		"air, no height": Air{Radius: 10, Thick: 0.8},
		"aurora": Aurora{Radius: 10, Low: 0.5, High: 3, Lat: 1.1,
			Width: 0.08, Thick: 2},
		"aurora, curtained, turned": Aurora{Radius: 10, Low: 0.5, High: 3,
			Lat: 1.2, Width: 0.1, Axis: maths.V(0.3, 1, 0.2), Thick: 2,
			Curtains: 0.8, Seed: 7},
		"clouds": Clouds{Radius: 10, Low: 0.2, High: 1, Cover: 0.5,
			Thick: 3, Seed: 4},
		"clouds, sparse": Clouds{Radius: 10, Low: 0.2, High: 1, Cover: 0.1,
			Thick: 3, Seed: 5},
		"disc": Disc{Inner: 2, Outer: 6, Thick: 0.4},
		"disc, ringed": Disc{Inner: 2, Outer: 6, Thick: 0.4, Dense: 3,
			Profile: []float64{0.2, 1, 0.1, 0.8, 0.5}, Grain: 0.6,
			Ringlets: 40, Seed: 3},
		"disc, one ring": Disc{Inner: 2, Outer: 6, Thick: 0.4,
			Profile: []float64{0.7}},
		"jet": Jet{Length: 10, Base: 1, Spread: 0.1, Thick: 2},
		"jet, knotted": Jet{Length: 10, Base: 1, Spread: 0.15, Thick: 2,
			From: 1.5, Knots: 0.6, Seed: 2},
		"jet, flowing": Jet{Length: 8, Base: 0.8, Spread: 0.1, Thick: 2,
			Knots: 0.7, Seed: 6, Flow: 2},
		"puff":            puff,
		"wispy":           Wispy{Of: puff, Seed: 8, Scale: 0.7, Amount: 0.6},
		"wispy, unscaled": Wispy{Of: puff, Seed: 8, Amount: 1.5},
		"giant":           Giant{Radius: 5, Depth: 0.6, Thick: 4},
		"pillars": Pillars{Count: 3, Height: 10, Width: 12, Thick: 1.5,
			Seed: 9},
		"pillars, filamented": Pillars{Count: 5, Height: 10, Width: 20,
			Thick: 1.5, Seed: 11, Filaments: 0.7},
		"plume": Plume{Height: 10, Width: 6, Thick: 1},
		"tail":  Tail{Length: 12, Base: 1, Spread: 0.1, Thick: 1},
		"tail, curved": Tail{Length: 12, Base: 1, Spread: 0.12, Curve: 0.3,
			Thick: 1},
		"tail, bent back": Tail{Length: 6, Base: 0.5, Spread: 0.3,
			Curve: -0.8, Thick: 2},
		"disc, ringed by default": Disc{Inner: 1, Outer: 3, Thick: 0.2,
			Profile: []float64{1, 0.3}, Grain: 1},
		"pillars, many": Pillars{Count: 9, Height: 6, Width: 9, Thick: 2,
			Seed: 1, Filaments: 1},
		"jet, wide open": Jet{Length: 4, Base: 2, Spread: 0.6, Thick: 1,
			From: 0.5, Knots: 1, Seed: 13},
		"plume, squat": Plume{Height: 3, Width: 8, Thick: 2},
		"aurora, wide": Aurora{Radius: 4, Low: 0.1, High: 0.8, Lat: 0.4,
			Width: 0.5, Thick: 1, Curtains: 0.3, Seed: 2},
		"air, thick": Air{Radius: 3, Height: 0.3, Thick: 5},
		"clouds, overcast": Clouds{Radius: 4, Low: 0.05, High: 0.6,
			Cover: 0.9, Thick: 1, Seed: 12},
	}
}

// draws is a fixed run of numbers from zero up to one: the inputs the recording
// is made over, the same on every machine, owing nothing to a random source
// that could change under it.
type draws uint64

// next is the run's next number.
func (d *draws) next() float64 {
	*d = *d*6364136223846793005 + 1442695040888963407
	return float64(uint64(*d)>>11) / (1 << 53)
}

// within is a point in a box grown a fifth either way, or in a cube eight
// across where the box has no edge.
func (d *draws) within(b maths.Box) maths.Vec {
	at := func(lo, hi float64) float64 {
		if math.IsInf(lo, 0) || math.IsInf(hi, 0) {
			lo, hi = -4, 4
		}
		pad := (hi - lo) / 5
		return lo - pad + (hi-lo+2*pad)*d.next()
	}
	return maths.V(at(b.Min.X, b.Max.X), at(b.Min.Y, b.Max.Y),
		at(b.Min.Z, b.Max.Z))
}

// digest is what a medium's methods give over the recording's inputs, a
// digest each: Bounds, Density at 3000 points, Span along 600 rays when it
// has one, its Tips when it is pillars, and its Density at three times
// when it moves.
func digest(m Medium) map[string]string {
	sums := map[string]string{}
	sum := func(method string, numbers ...float64) {
		h := sha256.New()
		word := make([]byte, 8)
		for _, n := range numbers {
			binary.LittleEndian.PutUint64(word, math.Float64bits(n))
			h.Write(word)
		}
		sums[method] = fmt.Sprintf("%x", h.Sum(nil))[:24]
	}
	b := m.Bounds()
	sum("bounds", b.Min.X, b.Min.Y, b.Min.Z, b.Max.X, b.Max.Y, b.Max.Z)
	densities := func(of Medium, seed draws) []float64 {
		var out []float64
		for range 3000 {
			out = append(out, of.Density(seed.within(b)))
		}
		return out
	}
	sum("density", densities(m, 1)...)
	if spans, ok := m.(interface {
		Span(maths.Ray, float64, float64) (float64, float64, bool)
	}); ok {
		var out []float64
		d := draws(2)
		for range 600 {
			at := d.within(b)
			r := maths.Ray{At: at, Dir: maths.V(d.next()-0.5, d.next()-0.5,
				d.next()-0.5).Unit()}
			enter, leave, met := spans.Span(r, 0, math.Inf(1))
			kept := 0.0
			if met {
				kept = 1
			}
			out = append(out, enter, leave, kept)
		}
		sum("span", out...)
	}
	if p, ok := m.(Pillars); ok {
		var out []float64
		for _, tip := range p.Tips() {
			out = append(out, tip.X, tip.Y, tip.Z)
		}
		sum("tips", out...)
	}
	if moving, ok := m.(Timed); ok {
		var out []float64
		for _, at := range []time.Duration{0, time.Second, 7 * time.Second} {
			out = append(out, densities(moving.Tick(at), 3)...)
		}
		sum("ticked", out...)
	}
	return sums
}

// lines is the recording's lines for every medium: name, method, digest, in
// order.
func lines() []string {
	var out []string
	for name, m := range recorded() {
		for method, sum := range digest(m) {
			out = append(out, fmt.Sprintf("%s\t%s\t%s", name, method, sum))
		}
	}
	sort.Strings(out)
	return out
}

// every medium here computes what the recording says, bit for bit, over the
// recording's inputs: none lost, none added, none changed.
func TestEveryMediumIsRecorded(t *testing.T) {
	got := lines()
	if os.Getenv("SLAYER_RECORD") == "1" {
		head := "# scene/medium at ad2fff9, by digest: see recording " +
			"in recorded_test.go\n"
		text := head + strings.Join(got, "\n") + "\n"
		if err := os.WriteFile(recording, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	file, err := os.Open(recording)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var want []string
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		if line := scan.Text(); !strings.HasPrefix(line, "#") {
			want = append(want, line)
		}
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	held := map[string]bool{}
	for _, line := range want {
		held[line] = true
	}
	for _, line := range got {
		if !held[line] {
			t.Errorf("not recorded: %s", line)
		}
	}
	if len(got) != len(want) {
		t.Errorf("%d lines here, %d recorded", len(got), len(want))
	}
}
