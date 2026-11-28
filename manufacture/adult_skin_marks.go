package manufacture

import (
	"math"

	"github.com/janearc/slayer/bioforms/pigment"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

func adultSkinMarks(from seed, scarRake bool) (pigment.Paint, error) {
	clusters := pigment.Mottle{
		Seed: noise.Mix(from.number ^ 0x636c757374657273), Scale: .022, Strength: 1,
	}
	region := pigment.MaskFunc(func(rest maths.Vec, footprint float64) float64 {
		coverage := freckledFace(rest, footprint)
		if coverage == 0 {
			return 0
		}
		return coverage * min(1, max(0, .60+.90*clusters.At(rest, footprint)))
	})
	freckles, err := pigment.NewScatter(pigment.Scatter{
		Seed:    noise.Mix(from.number ^ 0x667265636b6c6573),
		Spacing: .0025, Radius: .00100, Density: .67, Feather: .72,
		Region: region,
	})
	if err != nil {
		return pigment.Paint{}, err
	}
	darkerFreckles, err := pigment.NewScatter(pigment.Scatter{
		Seed:    noise.Mix(from.number ^ 0x7374726f6e676572),
		Spacing: .0052, Radius: .00150, Density: .48, Feather: .57,
		Region: region,
	})
	if err != nil {
		return pigment.Paint{}, err
	}
	blemishes, err := pigment.NewScatter(pigment.Scatter{
		Seed:    noise.Mix(from.number ^ 0x626c656d697368),
		Spacing: .017, Radius: .0050, Density: .46, Feather: .80,
		Region: pigment.MaskFunc(blemishedFace),
	})
	if err != nil {
		return pigment.Paint{}, err
	}
	layers := []pigment.Layer{
		{Style: pigment.Blemish, Mask: blemishes, Colour: maths.RGB(.43, .20, .17), Opacity: .31},
		{Style: pigment.Freckle, Mask: freckles, Colour: maths.RGB(.64, .52, .43), Opacity: .49},
		{Style: pigment.Freckle, Mask: darkerFreckles, Colour: maths.RGB(.54, .41, .32), Opacity: .73},
	}
	if scarRake {
		mark, err := adultCheekScar(from)
		if err != nil {
			return pigment.Paint{}, err
		}
		layers = append(layers, mark)
	}
	return pigment.NewPaint(layers)
}

// Centre and axes register the scratches with the outward curve of the
// left cheek in the canonical adult chart.
func adultCheekScar(from seed) (pigment.Layer, error) {
	mask, err := pigment.NewScarRake(pigment.ScarRake{
		Centre: maths.V(.046, 1.529, .062),
		Across: maths.V(.70, -.40, -.55), Along: maths.V(.32, .91, -.25),
		Seed: noise.Mix(from.number ^ 0x636865656b736361), Count: 3,
		Length: .035, Spacing: .0048, Radius: .0016, Depth: .015,
	})
	return pigment.Layer{Style: pigment.Scar, Mask: mask, Colour: maths.RGB(.56, .26, .22), Opacity: .78}, err
}

// Place the field across the cheeks, nose and outboard forearms in the
// native adult chart. The seed supplies the individual marks.
func freckledFace(rest maths.Vec, _ float64) float64 {
	x, y, z := math.Abs(rest.X), rest.Y, rest.Z
	front := maths.Smooth(.032, .059, z)
	cheek := maths.Smooth(1.518, 1.535, y) * (1 - maths.Smooth(1.549, 1.571, y))
	bridge := (1 - maths.Smooth(.014, .036, x)) * maths.Smooth(1.530, 1.543, y) *
		(1 - maths.Smooth(1.562, 1.585, y))
	forearm := maths.Smooth(.241, .268, x) * maths.Smooth(.820, .861, y) *
		(1 - maths.Smooth(1.051, 1.109, y))
	return max(front*max(cheek, bridge), .44*forearm)
}

func blemishedFace(rest maths.Vec, _ float64) float64 {
	x, y, z := math.Abs(rest.X), rest.Y, rest.Z
	cheek := maths.Smooth(.022, .043, x) * maths.Smooth(1.510, 1.529, y) *
		(1 - maths.Smooth(1.550, 1.580, y))
	forehead := maths.Smooth(1.583, 1.604, y) * (1 - maths.Smooth(1.625, 1.654, y))
	return maths.Smooth(.034, .064, z) * max(cheek, .65*forehead)
}
