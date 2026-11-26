// Package snakes builds the checked, static snake profiles in local metres.
package snakes

import (
	"math"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Result is one static snake's visible parts and conservative local extent.
type Result struct {
	Parts  []thing.Thing
	Bounds maths.Box
}

// Extent bounds the straight static animal and its eyes without building it.
// The X axis runs from tail to snout, centered on the origin.
func Extent(p Profile) (maths.Box, error) {
	if err := p.Validate(); err != nil {
		return maths.Box{}, err
	}
	wide, high := 0., 0.
	for _, s := range p.Stations {
		wide = math.Max(wide, s.Width*p.Length)
		high = math.Max(high, s.Height*p.Length)
	}
	eye := p.Head.EyeSize * p.Length
	return maths.Box{Min: maths.V(-p.Length/2-eye, -high-eye, -wide-eye), Max: maths.V(p.Length/2+eye, high+eye, wide+eye)}, nil
}

// Build constructs a single closed, unposed body and paired eyes. The body
// and its paint retain their own snapshots of the checked profile.
func Build(p Profile) (Result, error) {
	box, err := Extent(p)
	if err != nil {
		return Result{}, err
	}
	p.Stations = append([]Station(nil), p.Stations...)
	const around = 48
	const subdivisions = 8
	rows := (len(p.Stations) - 1) * subdivisions
	points := make([]maths.Vec, 0, rows*around+2)
	ring := func(at, width, height float64) {
		for j := 0; j < around; j++ {
			angle := 2 * math.Pi * float64(j) / around
			points = append(points, maths.V((at-.5)*p.Length, height*p.Length*math.Cos(angle), width*p.Length*math.Sin(angle)))
		}
	}
	for i := 0; i < len(p.Stations)-1; i++ {
		a, b := p.Stations[i], p.Stations[i+1]
		for step := 0; step < subdivisions; step++ {
			f := float64(step) / subdivisions
			at := a.At + (b.At-a.At)*f
			width, height := section(p.Stations, at)
			ring(at, width, height)
		}
	}
	// The final vertex is the nose pole: it stays exactly at the profile's
	// total length while the last span rounds down to it.
	points = append(points, maths.V(-p.Length/2, 0, 0), maths.V(p.Length/2, 0, 0))
	faces := geometry.IndexedLoftFaces(rows, around)
	skin, err := mesh.New(points, faces, geometry.UnitNormals(points, faces), true)
	if err != nil {
		return Result{}, err
	}
	body := thing.Thing{Shape: skin, Material: &thing.Material{Name: p.Kind + " skin", Texture: snakePaint{profile: p}, Satin: .24}, Motion: thing.Still(maths.Local())}
	parts := []thing.Thing{body}
	width, height := section(p.Stations, p.Head.EyeAt)
	eyeRadius := p.Head.EyeSize * p.Length
	centreY := p.Head.EyeHeight * height * p.Length
	centreZ := width * p.Length * math.Sqrt(1-p.Head.EyeHeight*p.Head.EyeHeight)
	look := &thing.Material{Name: p.Kind + " eyes", Colour: maths.RGB(.015, .012, .009), Gloss: .32, Satin: .28}
	for _, side := range []float64{-1, 1} {
		surface := maths.V((p.Head.EyeAt-.5)*p.Length, centreY, side*centreZ)
		normal := eyeNormal(p.Stations, p.Head.EyeAt, p.Head.EyeHeight, side)
		centre := surface.Sub(normal.Scale(.75 * eyeRadius))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: eyeRadius}, Material: look, Motion: thing.Still(maths.NewSubspace(centre, maths.Identity, 1))})
	}
	return Result{Parts: parts, Bounds: box}, nil
}

// section keeps every nonterminal authored section, and rounds the final head
// spans with bounded monotone cubic interpolation. The positive terminal
// radii shape the taper, which closes at a zero-radius pole. Earlier body
// spans remain linear.
func section(stations []Station, at float64) (wide, high float64) {
	at = math.Max(0, math.Min(1, at))
	for i := 1; i < len(stations); i++ {
		a, b := stations[i-1], stations[i]
		if at <= b.At {
			f := (at - a.At) / (b.At - a.At)
			if i-1 < len(stations)-4 {
				return a.Width + (b.Width-a.Width)*f, a.Height + (b.Height-a.Height)*f
			}
			width := func(s Station) float64 { return s.Width }
			height := func(s Station) float64 { return s.Height }
			wide, high := roundedSection(stations, i-1, f, width), roundedSection(stations, i-1, f, height)
			if i == len(stations)-1 {
				// A quarter ellipse ends at a point without extending the
				// specimen past its authored length.
				scale := math.Sqrt(math.Max(0, 1-f*f))
				wide, high = wide*scale, high*scale
			}
			return wide, high
		}
	}
	last := stations[len(stations)-1]
	return last.Width, last.Height
}

// roundedSection uses finite harmonic slopes, clamped to each span, so an
// authored positive taper cannot become a negative radius or overshoot.
func roundedSection(stations []Station, i int, t float64, value func(Station) float64) float64 {
	slope := func(k int) float64 {
		if k == 0 || k == len(stations)-1 {
			return 0
		}
		a, b, c := stations[k-1], stations[k], stations[k+1]
		left := (value(b) - value(a)) / (b.At - a.At)
		right := (value(c) - value(b)) / (c.At - b.At)
		if math.IsInf(left, 0) || math.IsInf(right, 0) || left == 0 || right == 0 || math.Signbit(left) != math.Signbit(right) {
			return 0
		}
		v := 2 / (1/left + 1/right)
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return 0
		}
		return v
	}
	a, b := stations[i], stations[i+1]
	h := b.At - a.At
	u := t * t
	v := u * t
	result := (2*v-3*u+1)*value(a) + (v-2*u+t)*h*slope(i) + (-2*v+3*u)*value(b) + (v-u)*h*slope(i+1)
	return math.Max(math.Min(value(a), value(b)), math.Min(math.Max(value(a), value(b)), result))
}

// eyeNormal estimates the outward normal of the actual rounded head at the
// chosen eye station; the eye's center is set just beneath that surface.
func eyeNormal(stations []Station, at, level, side float64) maths.Vec {
	const sample = 1e-5
	lo, hi := math.Max(0, at-sample), math.Min(1, at+sample)
	w, h := section(stations, at)
	wl, hl := section(stations, lo)
	wh, hh := section(stations, hi)
	dw, dh := (wh-wl)/(hi-lo), (hh-hl)/(hi-lo)
	across := math.Sqrt(1 - level*level)
	return maths.V(-(dh*w*level*level + dw*h*across*across), w*level, side*h*across).Unit()
}
