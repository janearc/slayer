package gpu

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// Layout v1 is eight words for the existing Fizz recipe and evaluated clock.
// It is used only for explicit ClearCloud Fizz; ordinary Fizz remains baked.
var fizzFields = []string{
	"radius", "height", "thick", "churn", "advected", "spread",
	"seed_low", "seed_high",
}

// fizzNumbers names the Fizz card words and the Fizz kind for the Metal
// source.
func fizzNumbers() string {
	var text strings.Builder
	fmt.Fprintf(&text, "constexpr constant int fizz_kind = %d;\n", fizzKind)
	for i, name := range fizzFields {
		fmt.Fprintf(&text, "constexpr constant int fizz_%s = %d;\n", name, i)
	}
	return text.String()
}

// fizzCardOf packs a native Fizz into its card words. It refuses a Fizz
// whose values fall outside the float32 range.
func fizzCardOf(f medium.Fizz) ([]uint32, error) {
	if f.Radius <= 0 || f.Height <= 0 || f.Thick < 0 || f.Churn < 0 || f.Churn > 1 {
		return nil, unsupported{"invalid native Fizz parameters"}
	}
	minimum := float64(math.Float32frombits(0x00800000))
	radius := float32(f.Radius)
	square := float64(radius * radius)
	if math.IsNaN(square) || math.IsInf(square, 0) || square < minimum {
		return nil, unsupported{"native Fizz radius squared is outside normal float32 range"}
	}
	card := newCard(fizzFields)
	for _, v := range []struct {
		name  string
		value float64
	}{
		{"radius", f.Radius}, {"height", f.Height},
		{"thick", f.Thick}, {"churn", f.Churn},
		{"advected", f.AdvectionDistance()},
		{"spread", f.ExpansionFactor()},
	} {
		converted := float64(float32(v.value))
		if math.IsNaN(converted) || math.IsInf(converted, 0) ||
			v.value != 0 && math.Abs(converted) < minimum {
			return nil, unsupported{"native Fizz controls are outside normal float32 range"}
		}
		card.set(v.name, v.value)
	}
	// A chosen float32 noise-coordinate budget, not a physical bubble limit.
	// Four octaves multiply frequency by 2.03; below 2^20 integer cells the
	// gradient retains useful fractional precision and signed-index headroom.
	coordinate := (f.Height + math.Abs(f.AdvectionDistance())) / (f.Radius / 2)
	if f.Churn > 0 && coordinate*math.Pow(2.03, 3) > 1<<20 {
		return nil, unsupported{"native Fizz exceeds its float32 noise-coordinate budget"}
	}
	card.seed("seed", f.Seed)
	return card.words, nil
}

// fizzVolume places a native Fizz volume in the scene, with its card as
// the grid words.
func (encoded *scene) fizzVolume(space maths.Subspace, f medium.Fizz, material int) error {
	words, err := fizzCardOf(f)
	if err != nil {
		return err
	}
	first := encoded.boxes.words(words)
	given := placedAt(space)
	bounds := f.Bounds()
	given["puff"], given["baked"] = nil, nil
	given["info"] = wholes(0, 0, fizzKind, material)
	given["grid"], given["shone"] = wholes(0, 0, 0, first), wholes(-1, -1)
	given["low"], given["high"] = point(bounds.Min), point(bounds.Max)
	given["lighting"] = wholes(0, 0, 0, -1)
	given["tint"] = []float64{1, 1, 1}
	noAir(given)
	encoded.volumes.row(volumeStruct, given)
	encoded.counts.volumes++
	return nil
}
