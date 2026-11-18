package gpu

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"math"
	"strings"
)

// A single fixed-size word card holds one generic rotor-tip density field.
// No grid/particle geometry is allocated; seed words retain all64bits.
var propellerFields = []string{"origin_x", "origin_y", "origin_z", "axis_x", "axis_y", "axis_z", "radial_x", "radial_y", "radial_z", "cross_x", "cross_y", "cross_z", "blades", "radius", "omega", "slip", "active", "core", "density", "deficit", "recovery", "seconds", "fragment_scale", "fragment_contrast", "seed_low", "seed_high", "enabled"}

// propellerNumbers names the propeller card words for the Metal source.
func propellerNumbers() string {
	var text strings.Builder
	fmt.Fprintf(&text, "constexpr constant int propeller_kind = %d;\n", propellerKind)
	for at, name := range propellerFields {
		fmt.Fprintf(&text, "constexpr constant int propeller_%s = %d;\n", name, at)
	}
	return text.String()
}

// propellerCardOf packs a propeller cavitation field into card words.
func propellerCardOf(field medium.PropellerCavitation) ([]uint32, error) {
	p := field.Parameters()
	core := float32(p.CoreRadiusM)
	if !(core*core > 0) {
		return nil, unsupported{"a propeller core whose square underflows float32"}
	}
	card := newCard(propellerFields)
	up := maths.Up
	if math.Abs(p.Axis.Dot(up)) > .99 {
		up = maths.XAxis
	}
	radial := p.Axis.Cross(up).Unit()
	cross := p.Axis.Cross(radial).Unit()
	for _, v := range []struct {
		name  string
		value maths.Vec
	}{{"origin", p.Origin}, {"axis", p.Axis}, {"radial", radial}, {"cross", cross}} {
		card.set(v.name+"_x", v.value.X)
		card.set(v.name+"_y", v.value.Y)
		card.set(v.name+"_z", v.value.Z)
	}
	card.whole("blades", uint32(p.Blades))
	for _, v := range []struct {
		name  string
		value float64
	}{{"radius", p.TipRadius}, {"omega", 2 * math.Pi * p.RPM / 60}, {"slip", p.SlipstreamMPS}, {"active", math.Min(p.LengthM, p.PressureDeficitPa/p.RecoveryPaPerM)}, {"core", p.CoreRadiusM}, {"density", p.DensityPerM}, {"deficit", p.PressureDeficitPa}, {"recovery", p.RecoveryPaPerM}, {"seconds", field.Seconds()}, {"fragment_scale", p.FragmentScaleM}, {"fragment_contrast", p.FragmentContrast}} {
		if math.IsInf(float64(float32(v.value)), 0) || math.IsNaN(v.value) ||
			v.value != 0 && float32(v.value) == 0 {
			return nil, unsupported{"non-finite float32 propeller field"}
		}
		card.set(v.name, v.value)
	}
	card.seed("seed", p.FragmentSeed)
	if p.VisualOnset {
		card.whole("enabled", 1)
	}
	return card.words, nil
}

// propellerVolume places a propeller cavitation volume in the scene.
func (encoded *scene) propellerVolume(space maths.Subspace, field medium.PropellerCavitation, material int) error {
	words, err := propellerCardOf(field)
	if err != nil {
		return err
	}
	first := encoded.boxes.words(words)
	given := placedAt(space)
	given["puff"], given["baked"] = nil, nil
	bounds := field.Bounds()
	given["info"] = wholes(0, 0, propellerKind, material)
	given["grid"], given["shone"] = wholes(0, 0, 0, first), wholes(-1, -1)
	given["lighting"] = wholes(0, 0, 0, -1)
	given["low"], given["high"] = point(bounds.Min), point(bounds.Max)
	given["tint"] = []float64{1, 1, 1}
	noAir(given)
	encoded.volumes.row(volumeStruct, given)
	encoded.counts.volumes++
	return nil
}
