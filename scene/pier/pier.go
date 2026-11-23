// Package pier builds deck, piling and observation-wheel geometry in metres.
package pier

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/internal/beams"
	"github.com/janearc/slayer/scene/thing"
)

// Deck describes a pier above its centerline's horizontal reference plane.
// PileBottom is an offset from that same plane. PileSpacing is a maximum.
type Deck struct {
	Width, Height, Thickness, Overlap  float64
	PileBottom, PileSpacing, PileWidth float64
	RailHeight                         float64
	Surface, Pilings, Rail             *thing.Material
}

// New follows a local centerline with a deck and three rows of pilings.
func New(line []maths.Vec, deck Deck) ([]thing.Thing, error) {
	if len(line) < 2 || !positive(deck.Width, deck.Thickness, deck.PileWidth,
		deck.PileSpacing) || !finite(deck.Height) ||
		!finite(deck.PileBottom) || deck.Height <= deck.PileBottom ||
		!finite(deck.Overlap) || deck.Overlap < 0 ||
		!finite(deck.RailHeight) || deck.RailHeight < 0 ||
		deck.Surface == nil || deck.Pilings == nil ||
		(deck.RailHeight > 0 && deck.Rail == nil) {
		return nil, fmt.Errorf("pier: invalid deck or piling dimensions")
	}
	var made assembly
	stations := 0
	for i := 1; i < len(line); i++ {
		from, to := line[i-1], line[i]
		way := to.Sub(from)
		length := way.Len()
		if !finite(from.X) || !finite(from.Y) || !finite(from.Z) ||
			!finite(to.X) || !finite(to.Y) || !finite(to.Z) ||
			from.Y != to.Y || !positive(length) ||
			length/deck.PileSpacing > 2000 {
			return nil, fmt.Errorf("pier: invalid centerline span %d", i)
		}
		count := int(math.Ceil(length / deck.PileSpacing))
		stations += count
		if stations > 2000 {
			return nil, fmt.Errorf("pier: too many piling stations")
		}
		made.span(from, to, deck, count)
	}
	return made.parts, made.err
}

type assembly struct {
	parts []thing.Thing
	err   error
}

// take retains the part or the first mesh creation error.
func (a *assembly) take(part thing.Thing, err error) {
	if a.err != nil {
		return
	}
	if err != nil {
		a.err = err
		return
	}
	a.parts = append(a.parts, part)
}

// span builds one deck segment with pilings and optional side rails.
func (a *assembly) span(from, to maths.Vec, d Deck, count int) {
	way := to.Sub(from)
	cross := maths.V(-way.Z, 0, way.X).Unit()
	a.take(beams.Span(from, to, d.Width, d.Height, d.Thickness,
		d.Overlap, d.Surface))
	for index := range count {
		at := from.Add(way.Scale((float64(index) + .5) / float64(count)))
		for _, row := range [...]float64{-.36, 0, .36} {
			p := at.Add(cross.Scale(d.Width * row))
			a.take(beams.Rod(p.Add(maths.V(0, d.PileBottom, 0)),
				p.Add(maths.V(0, d.Height, 0)), d.PileWidth, d.Pilings))
		}
	}
	if d.RailHeight > 0 {
		a.rails(from, to, cross, d, count)
	}
}

// rails put posts and two bars along both sides of the deck.
func (a *assembly) rails(from, to, cross maths.Vec, d Deck, count int) {
	way := to.Sub(from)
	for _, side := range [...]float64{-.48, .48} {
		offset := cross.Scale(d.Width * side).Add(maths.V(0, d.Height, 0))
		for _, level := range [...]float64{.42, 1} {
			bar := offset.Add(maths.V(0, level*d.RailHeight, 0))
			a.take(beams.Rod(from.Add(bar), to.Add(bar), .095, d.Rail))
		}
		for i := range count {
			at := from.Add(way.Scale(float64(i) / float64(count))).Add(offset)
			a.take(beams.Rod(at, at.Add(maths.V(0, d.RailHeight, 0)), .12, d.Rail))
		}
	}
}

// finite reports whether a value is a usable coordinate.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// positive checks for finite, strictly positive dimensions.
func positive(values ...float64) bool {
	for _, value := range values {
		if !finite(value) || value <= 0 {
			return false
		}
	}
	return true
}
