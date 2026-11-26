package branching

import (
	"errors"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Limb is a run of pieces wearing sleeves, each joint drooping and wandering.
type Limb struct {
	Pieces              int
	Span, Droop, Wander float64
	Skins               []Sleeve
}

// ErrLimb marks an invalid limb or mismatched sleeve.
var ErrLimb = errors.New("manufacture: a limb needs a piece or more, " +
	"a span above nought and no more than one, a droop of nought or " +
	"above with an end, and a sleeve or more with a joint for each " +
	"end of each piece")

// NewLimb checks a limb's extent and the sleeves along its pieces.
func NewLimb(pieces int, span, droop float64, skins ...Sleeve) (Limb, error) {
	sound := pieces >= 1 && span > 0 && span <= 1 && droop >= 0 &&
		!math.IsInf(droop, 0) && len(skins) > 0
	for _, skin := range skins {
		sound = sound && len(skin.Joints) == pieces+1
	}
	if !sound {
		return Limb{}, ErrLimb
	}
	return Limb{Pieces: pieces, Span: span, Droop: droop, Skins: skins}, nil
}

// Flat reports whether every sleeve is a blade.
func (made Limb) Flat() bool {
	for _, skin := range made.Skins {
		if !skin.Flat {
			return false
		}
	}
	return true
}

// ScaleOf is a space's scale, treating an unspecified scale as one.
func ScaleOf(space maths.Subspace) float64 {
	if space.Scale == 0 {
		return 1
	}
	return space.Scale
}

// phaseAt gives a wandering limb the phase of its position.
func phaseAt(at maths.Vec) float64 {
	mixed := math.Float64bits(at.X)*0x9e3779b97f4a7c15 ^
		math.Float64bits(at.Y)*0xbf58476d1ce4e5b9 ^
		math.Float64bits(at.Z)*0x94d049bb133111eb
	mixed ^= mixed >> 31
	return float64(mixed%1_000_003) / 1_000_003 * (2 * math.Pi)
}

// Runs places the limb's pieces end to end, preserving its droop and wander.
func (made Limb) Runs(on Place) ([]Place, error) {
	step := on.Size.Length.InMetres() * made.Span / float64(made.Pieces)
	each := Size{Length: measure.Metres(step), Girth: on.Size.Girth}
	at, scale := on.Space.Origin, ScaleOf(on.Space)
	way := on.Space.Orientation.Out(maths.YAxis)
	first := on.Space
	first.Scale = scale
	runs := []Place{{Space: first, Size: each}}
	phase := phaseAt(on.Space.Origin)
	for len(runs) < made.Pieces {
		at = at.Add(way.Scale(step * scale))
		way = way.Sub(maths.Up.Scale(made.Droop)).Unit()
		if side := way.Cross(maths.Up); made.Wander > 0 && side.Len() > 0 && !math.IsInf(side.Len(), 0) {
			joint := float64(len(runs))
			way = way.Add(side.Unit().Scale(made.Wander * math.Sin(1.3*joint+phase))).
				Add(maths.Up.Scale(0.5 * made.Wander * math.Sin(0.9*joint+2*phase))).
				Unit()
		}
		space, err := SpaceAlong(at, way)
		if err != nil {
			return nil, err
		}
		space.Scale = scale
		runs = append(runs, Place{Space: space, Size: each})
	}
	return runs, nil
}
