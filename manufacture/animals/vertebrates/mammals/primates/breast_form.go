package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// BreastForm adds independently adjustable left and right contours to a
// native female HumanSpec. The zero value keeps the reference contours.
// Torso.Left and Torso.Right still control width, projection, height and relief size.
type BreastForm struct {
	Left, Right BreastSideForm
}

// BreastSideForm describes one side in canonical metres and radians.
// Positive outward values point away from the sternum on either side.
// Fullness and lower-pole length run from -1 to 1 relative to the reference.
type BreastSideForm struct {
	// RootOutward moves the footprint along the rib cage by up to .024 m.
	RootOutward float64
	// UpperFullness and LowerFullness alter depth away from the centre.
	UpperFullness, LowerFullness float64
	// LowerPoleLength shortens or extends the lower contour.
	LowerPoleLength float64
	// AreolaOutward and AreolaUp move the relief and pigment relative to
	// the root. Their limits are .015 m laterally and .035 m vertically.
	AreolaOutward, AreolaUp float64
	// The papillary lean is relative to the local surface normal. Each
	// axis takes up to .44 radians, with travel limited by papilla radius.
	PapillaOutwardTilt, PapillaUpTilt float64
	// WithoutAreola clears local relief and pigment.
	WithoutAreola bool
}

func (form BreastForm) side(sign float64) BreastSideForm {
	if sign > 0 {
		return form.Left
	}
	return form.Right
}

func (form BreastForm) valid() bool {
	for _, side := range [...]BreastSideForm{form.Left, form.Right} {
		if !between(side.RootOutward, -.024, .024) ||
			!between(side.UpperFullness, -1, 1) || !between(side.LowerFullness, -1, 1) ||
			!between(side.LowerPoleLength, -1, 1) ||
			!between(side.AreolaOutward, -.015, .015) || !between(side.AreolaUp, -.035, .035) ||
			!between(side.PapillaOutwardTilt, -.44, .44) || !between(side.PapillaUpTilt, -.44, .44) {
			return false
		}
	}
	return true
}

// Geometry and material masks share these canonical coordinates through
// variations and poses.
func (form BreastSideForm) areolarChart(side Side, point maths.Vec, rootX, rootY float64) (u, v float64) {
	return (math.Abs(point.X) - (rootX + form.AreolaOutward)) / side.AreolaRadius,
		(point.Y - (rootY + form.AreolaUp) + .008) / (.93 * side.AreolaRadius)
}

func (form BreastSideForm) papillaLean(side Side, across, normal maths.Vec, sign, relief float64) maths.Vec {
	across = across.Unit()
	up := normal.Cross(across)
	lean := across.Scale(sign * math.Tan(form.PapillaOutwardTilt)).
		Add(up.Scale(math.Tan(form.PapillaUpTilt)))
	// Limit tangent travel at the configured height so the profile keeps a
	// smooth differential even for a narrow and prominent papilla.
	limit := min(1, .30*side.PapillaRadius/(side.PapillaProjection*max(1e-12, lean.Len())))
	return lean.Scale(relief * limit)
}
