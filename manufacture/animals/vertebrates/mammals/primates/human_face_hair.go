package primates

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// Eyebrows grows one groom over the two brow ridges. Density is
// candidate roots per square metre. Length and Radius set the shaft
// scale in metres. Zero selects reference values for those three controls.
type Eyebrows struct {
	Seed                    uint64
	Density, Length, Radius float64
	Left, Right             BrowShape
	Material                *thing.Material
}

// BrowShape sets the rise of the whole brow and the height of its arch in
// metres relative to the reference. Fullness scales the width of the band.
// Zero Fullness uses the reference width.
type BrowShape struct {
	Rise, Arch, Fullness float64
}

func (shape BrowShape) resolved() (BrowShape, bool) {
	if shape.Fullness == 0 {
		shape.Fullness = 1
	}
	return shape, between(shape.Rise, -.004, .005) && between(shape.Arch, -.003, .004) &&
		between(shape.Fullness, .35, 1.6)
}

func (spec Eyebrows) layer() (hair.Layer, error) {
	if spec.Density == 0 {
		spec.Density = 1_650_000
	}
	if spec.Length == 0 {
		spec.Length = .0075
	}
	if spec.Radius == 0 {
		spec.Radius = .000048
	}
	left, leftOK := spec.Left.resolved()
	right, rightOK := spec.Right.resolved()
	if spec.Material == nil || !leftOK || !rightOK || !between(spec.Density, 1, 5_000_000) ||
		!between(spec.Length, .002, .015) || !between(spec.Radius, .000009, .00012) {
		return hair.Layer{}, fmt.Errorf("adult human: eyebrow hair requires a material and controls in the supported range")
	}
	spec.Left, spec.Right = left, right
	return hair.Layer{
		Seed: spec.Seed, Density: spec.Density, Segments: 6, Sides: 4,
		Material: spec.Material, SkinGap: .000045, Profile: spec.strand,
		FaceFilter: func(corners [3]maths.Vec) bool {
			return frontFaceBand(corners, 1.563, 1.599, .028)
		},
	}, nil
}

func (spec Eyebrows) strand(site hair.Site) hair.Strand {
	point := site.Chart
	x := math.Abs(point.X)
	shape, side := spec.Left, humanSides[0]
	if point.X < 0 {
		shape, side = spec.Right, humanSides[1]
	}
	start := .0135 + .00055*side.sign
	end := .0565 - .00075*side.sign
	if x <= start || x >= end || point.Z < .028 {
		return hair.Strand{}
	}
	along := (x - start) / (end - start)
	arch := bell((along - .61) / .30)
	grain := max(-1, min(1, noise.Gradient(spec.Seed, 203*point.X, 203*point.Y, 203*point.Z)))
	line := 1.5795 + shape.Rise + (.0043+shape.Arch)*arch - .0030*along*along
	line += .00038*grain - .00026*side.sign
	head := maths.Smooth(0, .21, along)
	tail := maths.Smooth(.51, 1, along)
	width := shape.Fullness * mix(.0027, .00130, tail) * mix(.29, 1, head)
	off := (point.Y - line) / width
	band := hairBand(along, 0, 1, .19) * (1 - maths.Smooth(.38, 1, math.Abs(off)))
	if band == 0 {
		return hair.Strand{}
	}
	// Inner hairs rise steeply. Both edges then lean toward the middle
	// of the brow while the outer tail combs down toward the temple.
	spread := hairRootDraw(spec.Seed+1, site)
	angle := mix(1.17, -.41, maths.Smooth(0, 1, along)) - .32*off + .39*(spread-.5)
	reach := mix(.48, .76, head) * mix(1, .48, tail)
	return hair.Strand{
		Coverage: band * (.88 + .09*grain),
		Flow:     site.ChartFlow(side.at(math.Cos(angle), math.Sin(angle), 0)),
		Length:   spec.Length * reach * mix(.67, 1.26, hairRootDraw(spec.Seed+2, site)),
		Radius:   spec.Radius * mix(.69, 1, hairRootDraw(spec.Seed+3, site)),
		RootLift: .21 + .12*spread,
		TipLift:  .06 + .15*spread,
		Sweep:    -side.sign * (.16 + .18*grain),
		Taper:    hair.ShaftTaper{Start: .23, TipShare: .03},
	}
}

// Eyelashes grows separate rows of upper and lower lashes in the same
// material. A nil row grows no lashes. The seed is split between the rows.
type Eyelashes struct {
	Seed         uint64
	Upper, Lower *LashLine
	Material     *thing.Material
}

// LashLine sets candidate roots per square metre of lid margin and
// rendered shaft dimensions in metres. Zero selects the row's reference
// density, length and radius. Curl adds bend from zero to one.
type LashLine struct {
	Density, Length, Radius, Curl float64
}

func (spec Eyelashes) layers(margins [2]lidMargin) ([]hair.Layer, error) {
	var layers []hair.Layer
	for index, line := range [...]*LashLine{spec.Upper, spec.Lower} {
		if line == nil {
			continue
		}
		upper := index == 0
		seed := noise.Mix(spec.Seed + uint64(index))
		layer, err := line.layer(spec.Material, seed, upper, margins)
		if err != nil {
			return nil, err
		}
		layers = append(layers, layer)
	}
	return layers, nil
}

func (line LashLine) layer(material *thing.Material, seed uint64, upper bool, margins [2]lidMargin) (hair.Layer, error) {
	standard := LashLine{Density: 1_650_000, Length: .0035, Radius: .000036}
	if upper {
		standard = LashLine{Density: 3_150_000, Length: .0062, Radius: .000048}
	}
	if line.Density == 0 {
		line.Density = standard.Density
	}
	if line.Length == 0 {
		line.Length = standard.Length
	}
	if line.Radius == 0 {
		line.Radius = standard.Radius
	}
	if material == nil || !between(line.Density, 1, 15_000_000) ||
		!between(line.Length, .0017, .016) || !between(line.Radius, .000015, .0001) ||
		!between(line.Curl, 0, 1) {
		return hair.Layer{}, fmt.Errorf("adult human: eyelash hair requires a material and controls in the supported range")
	}
	return hair.Layer{
		Seed: seed, Density: line.Density, Segments: 8, Sides: 4,
		Material: material, SkinGap: .000032,
		FaceFilter: func(corners [3]maths.Vec) bool {
			return frontFaceBand(corners, eyeHeight-.009, eyeHeight+.009, eyeDepth)
		},
		Profile: func(site hair.Site) hair.Strand {
			return line.strand(site, seed, upper, margins)
		},
	}, nil
}

func (line LashLine) strand(site hair.Site, seed uint64, upper bool, margins [2]lidMargin) hair.Strand {
	side := humanSides[0]
	if !margins[0].contains(site.Face) {
		if !margins[1].contains(site.Face) {
			return hair.Strand{}
		}
		side = humanSides[1]
	}
	point := site.Chart
	across := (side.sign*point.X - eyeHalfSpacing) / lidHalfWidth
	onUpper := point.Y > eyeHeight+lidSlant*across
	if upper != onUpper {
		return hair.Strand{}
	}
	span := hairBand(across, -.88, .91, .34)
	if span == 0 {
		return hair.Strand{}
	}
	vertical := -1.0
	curl := .12 + .34*line.Curl
	taper := .15
	if upper {
		vertical = 1
		curl = .21 + .57*line.Curl
		taper = .24
	}
	// The free length peaks just lateral to centre. Roots near the
	// inner corner are finer and turn less strongly toward the temple.
	reach := .40 + .60*bell((across-.12)/.66)
	direction := hairRootDraw(seed+1, site)
	length := hairRootDraw(seed+2, site)
	fineness := hairRootDraw(seed+3, site)
	bend := hairRootDraw(seed+4, site)
	turn := mix(-.17, .33, maths.Smooth(-.81, .90, across)) + .36*(direction-.5)
	root := mix(1.02, 1.42, direction)
	return hair.Strand{
		Coverage: span,
		Flow:     site.ChartFlow(side.at(turn, vertical, 0)),
		Length:   line.Length * reach * mix(.56, 1.13, length),
		Radius:   line.Radius * reach * mix(.70, 1.08, fineness),
		RootLift: root,
		TipLift:  root - curl*mix(.58, 1.25, bend),
		Sweep:    -side.sign * vertical * (.05 + .15*bend) * maths.Smooth(-.60, .87, across),
		Taper:    hair.ShaftTaper{Start: taper, TipShare: .025},
	}
}

func frontFaceBand(corners [3]maths.Vec, low, high, front float64) bool {
	return max(corners[0].Y, corners[1].Y, corners[2].Y) >= low &&
		min(corners[0].Y, corners[1].Y, corners[2].Y) <= high &&
		max(corners[0].Z, corners[1].Z, corners[2].Z) >= front
}
