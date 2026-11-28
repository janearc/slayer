package primates

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// HumanHair selects independently seeded hair populations.
// A nil population leaves its roots bare.
type HumanHair struct {
	Scalp     *ScalpHair
	Body      *BodyHair
	Vellus    *VellusHair
	Eyebrows  *Eyebrows
	Eyelashes *Eyelashes
}

// Hairline moves the reference cuts in the canonical human chart.
// Each value is in metres. Positive FrontRise and NapeRise expose
// more skin. TempleRecession raises the frontotemporal corners.
// Peak lowers the centre of the frontal line; Asymmetry raises one side.
type Hairline struct {
	FrontRise, NapeRise              float64
	TempleRecession, Peak, Asymmetry float64
}

// ScalpStyle selects a native scalp cut.
type ScalpStyle uint8

const (
	SweptCrop ScalpStyle = iota
	PartedMedium
)

// ScalpHair describes a scalp groom. Length is the scale of
// its longest guide, and Radius its widest shaft radius, both in metres.
// SideLength and NapeLength set the taper relative to the crown Length.
// Density is candidate shafts per square metre. Zero uses the reference
// values. Part shifts the part sideways. Curl sets the strength of the
// authored Wavelength in metres.
type ScalpHair struct {
	Style                              ScalpStyle
	Seed                               uint64
	Density, Length, Radius            float64
	SideLength, NapeLength, Wavelength float64
	Line                               Hairline
	Part, Curl                         float64
	// MediumSweep, from zero to one, shapes the face-framing sweep and
	// the opposing tuck. An offset Part chooses which side tucks.
	MediumSweep float64
	// CrownLift adds a second, lofted course to some of the medium crown
	// strands. Zero keeps the base cut. Full strength is one.
	CrownLift float64
	// ScalpField grows the medium cut along connected chart directions.
	ScalpField bool
	// FieldWisps adds an authored envelope to the field cut.
	FieldWisps       *ScalpWispShape
	CounterClockwise bool
	Material         *thing.Material
}

// ScalpWispShape controls guide spacing, tip spread and part reach in
// metres. Volume scales lift and shoulder offsets. Zero values select
// the wisp study defaults.
type ScalpWispShape struct {
	Spacing, TipSpread, PartReach, Volume float64
	// BlendPart carries a common rearward flow through the end of the part.
	BlendPart bool
}

// Groom grows selected populations on the measured rest geometry using
// the canonical anatomical chart. Use the same HumanPose's point order
// when posing the returned groom.
func (human *Human) Groom(style HumanHair) (*hair.Groom, error) {
	surface := hair.Surface{Points: human.points, Chart: human.chart, Faces: human.faces}
	layers := make([]hair.Layer, 0, 9)
	if style.Scalp != nil {
		scalp, err := style.Scalp.layers(surface)
		if err != nil {
			return nil, err
		}
		if first, end := human.earFaces[0], human.earFaces[1]; first < end {
			for i := range scalp {
				profile := scalp[i].Profile
				scalp[i].Profile = func(site hair.Site) hair.Strand {
					if site.Face >= first && site.Face < end {
						return hair.Strand{}
					}
					return profile(site)
				}
			}
		}
		layers = append(layers, scalp...)
	}
	if style.Body != nil {
		body, err := style.Body.layer()
		if err != nil {
			return nil, err
		}
		layers = append(layers, body)
	}
	if style.Vellus != nil {
		fine, err := style.Vellus.layer(human.digitHair)
		if err != nil {
			return nil, err
		}
		if fine != nil {
			layers = append(layers, *fine)
		}
	}
	if style.Eyebrows != nil {
		brows, err := style.Eyebrows.layer()
		if err != nil {
			return nil, err
		}
		layers = append(layers, brows)
	}
	if style.Eyelashes != nil {
		lashes, err := style.Eyelashes.layers(human.lashMargins)
		if err != nil {
			return nil, err
		}
		layers = append(layers, lashes...)
	}
	return hair.New(surface, layers)
}

func hairRootDraw(seed uint64, site hair.Site) float64 {
	return noise.Hash3(seed, int64(site.Face), int64(math.Float64bits(site.Weights[1])),
		int64(math.Float64bits(site.Weights[2])))
}

func (spec ScalpHair) layers(surface hair.Surface) ([]hair.Layer, error) {
	switch spec.Style {
	case SweptCrop:
		return spec.cropLayers()
	case PartedMedium:
		return spec.mediumLayers(surface)
	default:
		return nil, fmt.Errorf("adult human: unknown scalp style %d", spec.Style)
	}
}

func (spec ScalpHair) cropLayers() ([]hair.Layer, error) {
	if spec.Density == 0 {
		spec.Density = 670_000
	}
	if spec.Length == 0 {
		spec.Length = .058
	}
	if spec.Radius == 0 {
		spec.Radius = .000069
	}
	if spec.SideLength == 0 {
		spec.SideLength = .52 * spec.Length
	}
	if spec.NapeLength == 0 {
		spec.NapeLength = .42 * spec.Length
	}
	if spec.Wavelength == 0 {
		spec.Wavelength = .055
	}
	if spec.Material == nil || !between(spec.Density, 20_000, 1_300_000) ||
		!between(spec.Length, .005, .090) || !between(spec.Radius, .000015, .00015) ||
		!between(spec.SideLength, .002, .090) || !between(spec.NapeLength, .002, .090) ||
		!between(spec.Wavelength, .018, .14) ||
		!between(spec.Part, -.044, .044) || !between(spec.Curl, 0, 1) || !spec.Line.valid() {
		return nil, fmt.Errorf("adult human: scalp hair requires a material and controls in the supported range")
	}
	clumps := hair.Clumps{Seed: noise.Mix(spec.Seed), Spacing: .017}
	finest := 7
	if spec.Curl > 0 {
		turns := min(2.4, max(spec.Length, spec.SideLength, spec.NapeLength)/spec.Wavelength)
		finest = max(finest, int(math.Ceil(6*turns)))
	}
	layer := func(kind scalpPopulation, seed uint64, density float64, segments, sides int) hair.Layer {
		return hair.Layer{
			Seed: seed, Density: density * spec.Density, Segments: segments, Sides: sides,
			Material: spec.Material, FaceFilter: scalpFace, SkinGap: .00018,
			Profile: func(site hair.Site) hair.Strand { return spec.strand(site, kind, clumps) },
		}
	}
	return []hair.Layer{
		layer(scalpMain, spec.Seed, 1, finest, 4),
		layer(scalpFringe, noise.Mix(spec.Seed^0x7363616c702d6564), .30, 4, 3),
		layer(scalpLoose, noise.Mix(spec.Seed^0x686169722d746970), .012, finest, 3),
	}, nil
}

type scalpPopulation uint8

const (
	scalpMain scalpPopulation = iota
	scalpMargin
	scalpFringe
	scalpLoose
)

func scalpFace(points [3]maths.Vec) bool {
	return max(points[0].Y, points[1].Y, points[2].Y) >= 1.512
}

func (spec ScalpHair) partDistance(point maths.Vec) float64 {
	bend := .011 * maths.Smooth(-.035, .080, point.Z)
	irregular := noise.Gradient(spec.Seed+11, point.Z*111, 0, 0)
	line := spec.Part + bend + .004*math.Sin(36*(point.Z+.018)) + .0032*irregular
	return point.X - line
}

func (line Hairline) valid() bool {
	return between(line.FrontRise, -.018, .027) && between(line.NapeRise, -.015, .025) &&
		between(line.TempleRecession, 0, .032) && between(line.Peak, 0, .018) &&
		between(line.Asymmetry, -.012, .012)
}
