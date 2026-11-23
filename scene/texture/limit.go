package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Detail is how much of a texture's fine detail a pixel can show, from
// nought to one, for detail of a size that is a feature and a pixel that
// is a footprint wide at the surface, both in the same units. A pixel
// that is half a feature wide or less shows it all; one that is a whole
// feature wide or more shows none, and the detail is better painted as
// its average, which is what a far surface of grain does in a photograph.
//
// Between the two it fades, so a surface does not shimmer as it recedes.
// A footprint that is not over nought is no pixel at all: all of it.
func Detail(footprint, feature float64) float64 {
	if !(footprint > nought) || !(feature > nought) {
		return unity
	}
	shown := feature / detailFull
	return unity - math.Min(unity, math.Max(nought,
		(footprint-shown)/(feature-shown)))
}

// detailFull is defined by sampling: two pixels across a feature, the
// least that shows a pattern's period, show it whole.
const detailFull = 2

// lineCover is defined: a line darkens both sides of an edge, so it
// covers two widths of itself for each spacing of tiles, on average.
const lineCover = 2

// kept is the near paint where the pixel can show its detail and the
// average where it cannot, blended by how much detail it shows.
func kept(near, average maths.Light, shown float64) maths.Light {
	return near.Scale(shown).Add(average.Scale(unity - shown))
}

// Within is the dust's surface at the point for a pixel footprint wide:
// patches that are narrower than a pixel fade to the average of rock and
// dust.
func (dusted Dusted) Within(point maths.Vec, footprint float64) maths.Light {
	share := math.Min(unity, math.Max(nought, dusted.Share))
	average := dusted.Rock.Scale(unity - share).Add(dusted.Dust.Scale(share))
	return kept(dusted.At(point), average, Detail(footprint, dusted.Patch))
}

// Within is the wood at the point for a pixel footprint wide: rings
// narrower than a pixel fade to the average of early and late wood.
func (wood Wood) Within(point maths.Vec, footprint float64) maths.Light {
	share := wood.lateShare()
	average := wood.Early.Scale(unity - share).Add(wood.Late.Scale(share))
	shown := Detail(footprint, wood.ring())
	return kept(wood.At(point), average, shown)
}

// lines is the share of a tiling's surface its lines cover on average.
func (tiled Tiled) lines() float64 {
	return math.Min(unity, lineCover*tiled.width()/tiled.Tiling.Spacing())
}

// Within is the tiles at the point for a pixel footprint wide: lines
// narrower than a pixel fade to a share of the tile's paint, the average
// of the line and the tile.
func (tiled Tiled) Within(point maths.Vec, footprint float64) maths.Light {
	if tiled.Paints == nil || tiled.Tiling == nil || tiled.Lines <= nought {
		return tiled.At(point)
	}
	paint := tiled.Paints.Paint(tiled.Tiling.Nearest(point))
	average := paint.Scale(unity - tiled.Lines*tiled.lines())
	shown := Detail(footprint, tiled.width())
	return kept(tiled.At(point), average, shown)
}

// Within is the tiles of the palette at the point for a pixel footprint
// wide, as Tiled's.
func (hexed Hexed) Within(point maths.Vec, footprint float64) maths.Light {
	return Tiled{Tiling: hexed.Tiling, Paints: hexed, Lines: hexed.Lines,
		Width: hexed.Width}.Within(point, footprint)
}

// Within is the wire at the point for a pixel footprint wide: lines
// narrower than a pixel fade to the share of the surface they cover.
func (wire Wire) Within(point maths.Vec, footprint float64) maths.Light {
	if wire.Tiling == nil {
		return noPaint
	}
	width := wire.lineWidth()
	covered := math.Min(unity, lineCover*width/wire.Tiling.Spacing())
	average := wire.Light.Scale(covered)
	return kept(wire.At(point), average, Detail(footprint, width))
}

// wireShare is how many lines of the default width fit in a tiling's
// spacing, as At draws it.
var wireShare = pickedNumber("wire-lines-in-spacing")
