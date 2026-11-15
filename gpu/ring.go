package gpu

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/occultation"
	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/scene/texture"
)

// ringFields are the words of a ring's card, in order: everything the kernel
// needs to find the ring's share at a point of its plane, as planet's
// Ring.share does. A word that holds a count or a seed holds it whole, and
// every other holds a float. Behind these words come the lists they point to:
// the grains' weights, the moonlets, and the measured profile's levels.
var ringFields = []string{
	"inner", "outer", "half", "depth", "scale",
	"ringlet_strength", "ringlet_width", "ringlet_seed_low",
	"ringlet_seed_high", "ringlet_layers", "ringlet_finer",
	"ringlet_weaker", "ringlet_across",
	"stretch_wide", "stretch_from", "stretch_inner", "stretch_step",
	"stretch_bins", "stretch_levels", "stretch_first",
	"grain_pattern", "grain_seed_low", "grain_seed_high", "grain_layers",
	"grain_largest", "grain_shear", "grain_threshold", "grain_softness",
	"grain_spread", "grain_filled", "grain_widths", "grain_first",
	"moonlet_count", "moonlet_first", "moonlet_edge",
	"eye_x", "eye_y", "eye_z", "eye_ray",
	"glow", "glow_faint", "glow_source", "glow_radius", "glow_lift",
	"glow_red", "glow_green", "glow_blue",
}

// moonletFields are the words of one moonlet in a card's list of them.
var moonletFields = []string{"reach", "angle", "gap", "length", "opens"}

// the patterns of grains, as the kernel names them; none is zero, for a ring
// with no grains
const (
	noPattern = iota
	smoothPattern
	ridgesPattern
	cellsPattern
	specklePattern
)

// patternsOf is each pattern planet names, by the number the kernel knows it
// by.
var patternsOf = map[string]int{"smooth": smoothPattern,
	"ridges": ridgesPattern, "cells": cellsPattern,
	"speckle": specklePattern}

// the numbers of a ring the kernel is told, each true by definition: hashUnit
// is the unit of a float's twenty-four bits, which a hash is cut to; hashDrop
// is how many of a sixty-four bit hash's bits are dropped to cut it so; and
// profileFloor is where the occultation's tables stop a depth growing.
const (
	hashBits     = 24
	hashDrop     = seedHalfBits*wholeTwo - hashBits
	hashUnit     = 1.0 / (1 << hashBits)
	profileFloor = occultation.Floor
)

// flatPart is how small a part of the finest scale a ring carries a stretch
// of a ray across its plane may run and still read one density: a layer of
// noise of a wavelength s, read at the middle of a stretch of l instead of
// averaged over it, is off by at most (pi l / s) squared over six of its
// amplitude, which for a part of a thousandth is 1.6e-6, under the exact
// tolerance's 1e-5. picked, from that bound.
const flatPart = 1e-3

// neverFlat is the scale of a ring whose density has a step the stretch could
// straddle, which no stretch is short of.
const neverFlat = 0

// ringNumbers are the kernel's declarations of a ring's kind, its patterns,
// its card's words and its moonlets', and the numbers it is told.
func ringNumbers() string {
	var text strings.Builder
	fmt.Fprintf(&text, "constexpr constant int ring_kind = %d;\n", ringKind)
	for _, name := range []string{"smooth", "ridges", "cells", "speckle"} {
		fmt.Fprintf(&text, "constexpr constant int ring_%s = %d;\n", name,
			patternsOf[name])
	}
	for at, name := range ringFields {
		fmt.Fprintf(&text, "constexpr constant int ring_%s = %d;\n", name, at)
	}
	for at, name := range moonletFields {
		fmt.Fprintf(&text, "constexpr constant int moonlet_%s = %d;\n", name,
			at)
	}
	fmt.Fprintf(&text, "constexpr constant int moonlet_words = %d;\n",
		len(moonletFields))
	fmt.Fprintf(&text, "constexpr constant float ring_unit = %.9g;\n", hashUnit)
	fmt.Fprintf(&text, "constexpr constant float ring_floor = %g;\n",
		profileFloor)
	fmt.Fprintf(&text, "constexpr constant float ring_flat_part = %g;\n",
		flatPart)
	fmt.Fprintf(&text, "constexpr constant float ring_stretch = %g;\n",
		planet.MostStretch)
	return text.String()
}

// wordCard is a card as it is made: its words, and the names of the first of
// them, a ring's or a wake's.
type wordCard struct {
	fields []string
	words  []uint32
}

// newCard is a card of the fields given, with every word zero, and room for
// the lists.
func newCard(fields []string) *wordCard {
	return &wordCard{fields: fields, words: make([]uint32, len(fields))}
}

// at is where in the card a field is.
func (card *wordCard) at(name string) int {
	for at, each := range card.fields {
		if each == name {
			return at
		}
	}
	panic("gpu: a card has no word " + name)
}

// set writes a number as the float that holds it.
func (card *wordCard) set(name string, number float64) {
	card.words[card.at(name)] = math.Float32bits(float32(number))
}

// whole writes a count or a seed's half as it is.
func (card *wordCard) whole(name string, number uint32) {
	card.words[card.at(name)] = number
}

// seed writes a seed, low half and high.
func (card *wordCard) seed(name string, seed uint64) {
	card.whole(name+"_low", uint32(seed))
	card.whole(name+"_high", uint32(seed>>seedHalfBits))
}

// list appends numbers as floats behind the card's words and says where they
// begin, by their place among the words.
func (card *wordCard) list(numbers ...float64) int {
	first := len(card.words)
	for _, number := range numbers {
		card.words = append(card.words, math.Float32bits(float32(number)))
	}
	return first
}

// ringletsOf writes the ring's ringlets, if it has any.
func (card *wordCard) ringletsOf(ring planet.Ring) {
	term, has := ring.RingletTerm()
	if !has {
		return
	}
	card.set("ringlet_strength", term.Strength())
	card.set("ringlet_width", term.Width())
	card.seed("ringlet_seed", term.Seed())
	card.whole("ringlet_layers", uint32(term.Layers()))
	card.set("ringlet_finer", term.Finer())
	card.set("ringlet_weaker", term.Weaker())
	card.set("ringlet_across", term.Across())
}

// stretchOf writes the ring's measured stretch and its profile's levels, if
// it has one: the levels as the occultation package baked them, in the
// float32 it reads them in.
func (card *wordCard) stretchOf(ring planet.Ring) {
	term, has := ring.StretchTerm()
	if !has {
		return
	}
	baked := term.Baked()
	levels := baked.Levels()
	card.set("stretch_wide", term.Wide())
	card.set("stretch_from", term.From())
	card.set("stretch_inner", float64(baked.Inner()))
	card.set("stretch_step", float64(baked.Step()))
	card.whole("stretch_bins", uint32(len(levels[0])))
	card.whole("stretch_levels", uint32(len(levels)))
	first := len(card.words)
	for _, level := range levels {
		for _, kept := range level {
			card.words = append(card.words, math.Float32bits(kept))
		}
	}
	card.whole("stretch_first", uint32(first))
}

// grainsOf writes the ring's grains, if it has any, or refuses a pattern
// the kernel has no way to draw.
func (card *wordCard) grainsOf(ring planet.Ring) error {
	term, has := ring.GrainTerm()
	if !has {
		return nil
	}
	pattern, known := patternsOf[term.Pattern()]
	if !known {
		return unsupported{fmt.Sprintf("a ring with grains in the pattern %q",
			term.Pattern())}
	}
	card.whole("grain_pattern", uint32(pattern))
	card.seed("grain_seed", term.Seed())
	card.whole("grain_layers", uint32(term.Layers()))
	card.set("grain_largest", term.Largest())
	card.set("grain_shear", term.Shear())
	card.set("grain_threshold", term.Threshold())
	card.set("grain_softness", term.Softness())
	card.set("grain_spread", term.Spread())
	card.set("grain_filled", term.Filled())
	card.set("grain_widths", term.Widths())
	card.whole("grain_first", uint32(card.list(term.Weights()...)))
	return nil
}

// moonletsOf writes the ring's moonlets, if it has any.
func (card *wordCard) moonletsOf(ring planet.Ring) {
	found := ring.MoonletTerms()
	if len(found) == 0 {
		return
	}
	card.whole("moonlet_count", uint32(len(found)))
	card.set("moonlet_edge", found[0].Edge())
	first := len(card.words)
	for _, each := range found {
		card.list(each.Reach(), each.Angle(), each.Gap(), each.Length(),
			opening(each.Opens()))
	}
	card.whole("moonlet_first", uint32(first))
}

// opening is one for a moonlet that clears its whole orbit, and zero for a
// propeller.
func opening(opens bool) float64 {
	if opens {
		return 1
	}
	return 0
}

// ringCardOf is a ring's card, for a slab half as thick as given: every term
// the ring has, or why the kernel cannot draw it.
func ringCardOf(ring planet.Ring, half float64) ([]uint32, error) {
	card := newCard(ringFields)
	inner, outer := ring.Span()
	card.set("inner", inner)
	card.set("outer", outer)
	card.set("half", half)
	card.set("depth", ring.Depth())
	card.set("scale", ringScale(ring))
	eye, ray := ring.Eye()
	card.set("eye_x", eye.X)
	card.set("eye_y", eye.Y)
	card.set("eye_z", eye.Z)
	card.set("eye_ray", ray)
	card.ringletsOf(ring)
	card.stretchOf(ring)
	card.moonletsOf(ring)
	if err := card.grainsOf(ring); err != nil {
		return nil, err
	}
	return card.words, nil
}

// glowCardOf is the card of a ring's glow slab: the ring's own, and the
// numbers its shine is made from.
func glowCardOf(slab planet.RingGlow) ([]uint32, error) {
	words, err := ringCardOf(slab.Ring(), slab.Half())
	if err != nil {
		return nil, err
	}
	card, shine := &wordCard{fields: ringFields, words: words}, slab.Shine()
	card.whole("glow", gasOn)
	card.set("glow_faint", slab.Faint())
	card.set("glow_source", shine.Source())
	card.set("glow_radius", shine.Radius())
	card.set("glow_lift", shine.Lift())
	light := shine.Colour()
	card.set("glow_red", light.R)
	card.set("glow_green", light.G)
	card.set("glow_blue", light.B)
	return card.words, nil
}

// words appends words to a buffer and says where the first went, by its
// place among the words.
func (buffer *packer) words(numbers []uint32) int {
	first := len(buffer.bytes) / wordBytes
	for _, number := range numbers {
		buffer.word(number)
	}
	return first
}

// ringFound is a ring the kernel asks itself: its card, its outer radius and
// half its thickness, which make its box, the slab's.
type ringFound struct {
	words       []uint32
	outer, half float64
}

// ringVolume appends one ring, or the glow of one: its box where the other
// volumes keep their boxes, and where its card begins where a boxed
// medium's cells begin.
func (encoded *scene) ringVolume(space maths.Subspace, found ringFound,
	material int) {
	first := encoded.boxes.words(found.words)
	given := placedAt(space)
	given["puff"], given["baked"] = nil, nil
	given["info"] = wholes(0, 0, ringKind, material)
	given["low"] = point(maths.V(-found.outer, -found.half, -found.outer))
	given["high"] = point(maths.V(found.outer, found.half, found.outer))
	given["grid"], given["shone"] = wholes(0, 0, 0, first), wholes(-1, -1)
	given["lighting"] = wholes(0, 0, 0, -1)
	given["tint"] = []float64{1, 1, 1}
	noAir(given)
	encoded.volumes.row(volumeStruct, given)
	encoded.counts.volumes++
}

// ringAsked is the ring a medium is, if it is a ring's slab or its glow that
// the kernel asks itself, or why it cannot be: a ring's slab is always asked,
// and a glow when its shine is the one made with it and it has no paint.
func ringAsked(med any, paint, shine texture.Texture) (ringFound, bool,
	error) {
	switch kind := med.(type) {
	case planet.RingSlab:
		words, err := ringCardOf(kind.Ring(), kind.Half())
		return ringBox(words, kind.Ring(), kind.Half()), true, err
	case planet.RingGlow:
		if paint != nil || kind.Shine() == nil ||
			shine != texture.Texture(kind.Shine()) {
			return ringFound{}, false, nil
		}
		words, err := glowCardOf(kind)
		return ringBox(words, kind.Ring(), kind.Half()), true, err
	}
	return ringFound{}, false, nil
}

// ringBox is a ring's card with the numbers of its box.
func ringBox(words []uint32, ring planet.Ring, half float64) ringFound {
	_, outer := ring.Span()
	return ringFound{words: words, outer: outer, half: half}
}

// ringIf appends the volume if its medium is a ring the kernel asks itself,
// and says so; a gas ring is a gas, with its optics beside.
func (encoded *scene) ringIf(space maths.Subspace, med any, textures painting,
	material int, gassy bool, optics airing) (bool, error) {
	found, asked, err := ringAsked(med, textures.paint, textures.shine)
	if !asked || err != nil {
		return asked, err
	}
	encoded.ringVolume(space, found, material)
	encoded.airIf(gassy, optics)
	return true, nil
}

// ringScale is the finest scale a ring carries across its plane, in metres: the
// least of its ringlets' finest layer, its profile's bins laid across it, its
// grains' smallest layer, and its moonlets' soft edges; the largest a float32
// holds for a ring with none. A speckle's cells step where their count does,
// and a stretch could straddle the step, so a ring with one has no scale.
func ringScale(ring planet.Ring) float64 {
	scale := math.MaxFloat32
	if term, has := ring.RingletTerm(); has {
		scale = math.Min(scale, term.Width()/math.Pow(term.Finer(),
			float64(term.Layers()-1)))
	}
	if term, has := ring.StretchTerm(); has {
		scale = math.Min(scale, float64(term.Baked().Step())/term.Wide())
	}
	if term, has := ring.GrainTerm(); has {
		if term.Pattern() == "speckle" {
			return neverFlat
		}
		scale = math.Min(scale, term.Largest()/math.Pow(2,
			float64(term.Layers()-1)))
	}
	for _, each := range ring.MoonletTerms() {
		scale = math.Min(scale, each.Edge()*math.Min(each.Gap(),
			each.Length()))
	}
	return scale
}
