package flame

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
)

// Ignition places a flame in metres, with an optional absolute burn window.
// Duration zero leaves no end time after Start. With both zero, the legacy
// medium also stays active before time zero. Fade smooths ignition and
// burnout; it is authored occupancy, not a fuel-mass solver.
type Ignition struct {
	At              maths.Vec
	Fire            Fire
	Start, Duration time.Duration
	Fade            time.Duration
}

// Piece describes one immutable body placed inside a field.
type Piece struct {
	At              maths.Vec
	Body            *Body
	Start, Duration time.Duration
	Fade            time.Duration
}

// Field combines flamelets in one participating medium. Absorption and
// emission coefficients add where they overlap; the emission texture is
// their absorption-weighted radiance, not their unweighted average.
type Field struct {
	pieces []Piece
	bounds maths.Box
	at     time.Duration
}

// NewField copies and checks one to sixty-four source patches.
func NewField(cells []Ignition, exposure float64) (*Field, error) {
	if len(cells) < 1 || len(cells) > 64 {
		return nil, errFlame
	}
	field := &Field{bounds: maths.Box{
		Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)),
		Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}}
	palettes := map[paletteKey]glowing{}
	for _, cell := range cells {
		if !finitePoint(cell.At) || cell.Duration < 0 || cell.Fade < 0 || cell.Fade > cell.Duration/2 || cell.Start > time.Duration(math.MaxInt64)-cell.Duration {
			return nil, errFlame
		}
		body, err := bodyWithPalette(cell.Fire, exposure, palettes)
		if err != nil {
			return nil, err
		}
		box := body.Bounds()
		box.Min, box.Max = box.Min.Add(cell.At), box.Max.Add(cell.At)
		if !finitePoint(box.Min) || !finitePoint(box.Max) {
			return nil, errFlame
		}
		field.bounds = fieldBounds(field.bounds, box)
		field.pieces = append(field.pieces, Piece{At: cell.At,
			Body: body, Start: cell.Start, Duration: cell.Duration, Fade: cell.Fade})
	}
	return field, nil
}

// Density adds each patch's absorption coefficient in shared coordinates.
func (field *Field) Density(point maths.Vec) float64 {
	sum := 0.0
	for _, piece := range field.pieces {
		body, active := field.bodyAt(piece)
		if active {
			sum += body.Density(point.Sub(piece.At))
		}
	}

	return sum
}

// Bounds encloses every source throughout its motion.
func (field *Field) Bounds() maths.Box {
	return field.bounds
}

// Tint is the common wavelength dependence of the source patches' soot.
func (field *Field) Tint() maths.Light {
	return Tilt()
}

// Finest is the smallest eddy any source asks the grid to resolve.
func (field *Field) Finest() float64 {
	finest := math.Inf(1)
	for _, piece := range field.pieces {
		finest = min(finest, piece.Body.Finest())
	}
	return finest
}

// Pieces copies active placements with bodies at their own ignition ages.
func (field *Field) Pieces() []Piece {
	var active []Piece
	for _, piece := range field.pieces {
		body, ok := field.bodyAt(piece)
		if !ok {
			continue
		}
		piece.Body = &body
		active = append(active, piece)
	}
	return active
}

// bodyAt returns a value so density/shine evaluation does not allocate a
// placement slice or a heap body for every ray-march sample.
func (field *Field) bodyAt(piece Piece) (Body, bool) {
	if piece.Body.fire.Absorption == 0 {
		return Body{}, false
	}
	legacy := piece.Start == 0 && piece.Duration == 0
	if !legacy && (field.at < piece.Start ||
		(piece.Duration > 0 && field.at >= piece.Start+piece.Duration)) {
		return Body{}, false
	}
	body := *piece.Body
	body.at = field.at - piece.Start
	if piece.Fade > 0 {
		share := math.Min(1, math.Min(float64(body.at), float64(piece.Duration-body.at))/float64(piece.Fade))
		body.fire.Absorption *= share * share * (3 - 2*share)
	}
	return body, true
}

// Tick selects an absolute time without discarding any planned sources.
func (field *Field) Tick(at time.Duration) medium.Medium {
	later := *field
	later.at = at
	return &later
}

// Sources samples each patch's emission integral without merging away light.
func (field *Field) Sources(cells int) []Source {
	if cells < 1 {
		return nil
	}
	var sources []Source
	for _, piece := range field.Pieces() {
		for _, source := range piece.Body.Sources(cells) {
			source.At = source.At.Add(piece.At)
			sources = append(sources, source)
		}
	}
	return sources
}

type fieldShine struct{ field *Field }

// Shine supplies the absorption-weighted source radiance.
func (field *Field) Shine() texture.Timed {
	return fieldShine{field}
}

// At combines emitted coefficients before dividing by total absorption.
func (glow fieldShine) At(point maths.Vec) maths.Light {
	absorption := 0.0
	emitted := maths.Light{}
	for _, piece := range glow.field.pieces {
		body, active := glow.field.bodyAt(piece)
		if !active {
			continue
		}
		local := point.Sub(piece.At)
		density := body.Density(local)
		if density == 0 {
			continue
		}
		absorption += density
		emitted = emitted.Add(body.glows.at(body.Heat(local)).Scale(
			density))
	}
	if absorption == 0 {
		return maths.Light{}
	}
	return emitted.Scale(1 / absorption)
}

// Tick keeps emission on the same clock as the participating medium.
func (glow fieldShine) Tick(at time.Duration) texture.Texture {
	return fieldShine{glow.field.Tick(at).(*Field)}
}

// Flames identifies the standard multi-source emission field.
func (glow fieldShine) Flames() *Field {
	return glow.field
}

// finitePoint rejects coordinates that cannot define a bounded source.
func finitePoint(point maths.Vec) bool {
	for _, v := range []float64{point.X, point.Y, point.Z} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// fieldBounds encloses both component boxes.
func fieldBounds(a, b maths.Box) maths.Box {
	return maths.Box{Min: maths.V(min(a.Min.X, b.Min.X), min(a.Min.Y,
		b.Min.Y),
		min(a.Min.Z, b.Min.Z)),
		Max: maths.V(max(a.Max.X, b.Max.X), max(a.Max.Y, b.Max.Y),
			max(a.Max.Z,
				b.Max.Z))}
}
