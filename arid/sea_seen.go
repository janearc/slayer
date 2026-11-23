// This file is how the sea looks where a ray meets it: its slope as a
// pixel there can show it, the slope the pixel cannot, and which side of
// the surface the ray came from.

package arid

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
)

// SeaTally is what a draw's rays did at a sea: how many used their steps
// and ended on the mean plane, and how many met it where the bands' slope
// leaned the surface away from them, so that their shading normal had to
// be turned to face them.
type SeaTally struct {
	OnMean, Leaning int64
}

// seaClearance is how many touches from the surface a look leaving the
// sea starts: two, so that a point found anywhere within one touch of it
// is left by at least one. Derived from the search's touch.
const seaClearance = 2

// Clearance is how far from the surface, in metres, a look leaving the
// sea starts, along the side it leaves toward: the search takes a ray
// within seaTouch as met, so a look that starts nearer may meet the same
// surface again at once, and a bent look may begin on the wrong side.
func (sea *Sea) Clearance() float64 { return seaClearance * seaTouch }

// Tally is what the draw's rays have done at the sea so far.
func (sea *Sea) Tally() SeaTally {
	return SeaTally{OnMean: sea.onMean.Load(), Leaning: sea.leaning.Load()}
}

// Seen is the way the sea faces at point for a ray along dir that met it
// there, seen by a pixel width metres across, and the slope variance of
// the waves the pixel cannot show, which the glint's lobe takes in their
// place. It is the one normal a ray's shading, mirror and bend take.
//
// Each band is kept by the share of it the pixel shows
// (oceantex.PixelShare), its slope scaled by that share, so a band of
// waves finer than the pixel is not drawn as a pattern the pixel cannot
// hold; the band's slope variance less that share squared of it is lost.
//
// The normal faces the side of the surface the ray came from, a ray that
// began at from (sided). Where the slope leans the surface away from the
// ray it is counted.
func (sea *Sea) Seen(point, from maths.Vec, width float64) (maths.Vec,
	float64) {
	dir := point.Sub(from)
	eastward, northward, lost := sea.slopeWithin(point.X, point.Z, width)
	share, shareEast, shareSouth := sea.coast.meet(point.X, point.Z)
	open := sea.waves(point.X, point.Z)
	eastward = share*eastward + open*shareEast
	northward = share*northward - open*shareSouth
	normal := sea.sided(maths.V(-eastward, 1, northward).Unit(), point,
		from)
	if normal.Dot(dir) >= 0 {
		sea.leaning.Add(1)
	}
	return normal, share * share * lost
}

// slopeWithin is the surface's rise a metre east and a metre north at an
// east and a south, of the bands a pixel width metres across shows, and
// the slope variance of the share of each band it does not.
func (sea *Sea) slopeWithin(east, south, width float64) (eastward,
	northward, lost float64) {
	return sea.slopeFrom(0, east, south, width)
}

// slopeFrom is slopeWithin of the bands from index on, read at an east
// and a south: the band's own slope there by the share of it the pixel
// shows, and the shorter bands' slope where this band's water came from,
// brought through the inverse of this band's carrying, one less its
// stretch.
//
// That is the slope of waves(): a short wave on a long one's
// crest is drawn shorter and steeper, as the long wave's water gathering
// there makes it.
func (sea *Sea) slopeFrom(index int, east, south, width float64) (eastward,
	northward, lost float64) {
	if index == len(sea.bands) {
		return 0, 0, 0
	}
	band := sea.bands[index]
	x, y := band.Extent/2+east, band.Extent/2-south
	shown := oceantex.PixelShare(width, band.Extent/float64(band.N))
	lost = (1 - shown*shown) * sea.slopes[index]
	if shown > 0 {
		eastward = shown * smoothSample(band, band.SlopeX, x, y)
		northward = shown * smoothSample(band, band.SlopeY, x, y)
	}
	if index == len(sea.bands)-1 {
		return eastward, northward, lost
	}
	_, pushEast, pushNorth := carriedSample(band, x, y)
	innerEast, innerNorth, innerLost := sea.slopeFrom(index+1,
		east-pushEast, south+pushNorth, width)
	xx := 1 - smoothSample(band, band.StretchXX, x, y)
	xy := -smoothSample(band, band.StretchXY, x, y)
	yy := 1 - smoothSample(band, band.StretchYY, x, y)
	return eastward + xx*innerEast + xy*innerNorth,
		northward + xy*innerEast + yy*innerNorth, lost + innerLost
}

// smoothSample is a band's grid at an east and a north on its tile, read
// by cubic convolution (oceantex.Field.Smooth): it passes through every
// sample as the bilinear read does, and its slope is continuous across
// sample lines, where the bilinear read's kinks. A normal taken from it
// has no creases for a mirror to draw as corners.
func smoothSample(field *oceantex.Field, values []float64, east,
	north float64) float64 {
	return field.Smooth(values, east, north)
}

// cubicTaps is how many samples a side cubic convolution reads.
const cubicTaps = oceantex.CubicTaps

// cubicWeights are cubic convolution's four weights at a fraction t of
// the way from the second sample to the third (oceantex.CubicWeights).
func cubicWeights(t float64) [cubicTaps]float64 {
	return oceantex.CubicWeights(t)
}

// sided is normal, an upward normal, turned to the side of the surface a
// ray from from met it on at point: Hit's search does not cross the
// surface, so the ray meets it on the side it began. A ray that began on
// the surface is taken as above when it goes down. The gap where the
// search stopped is not used: it is within the search's touch, and its
// sign is not to be trusted there.
func (sea *Sea) sided(normal, point, from maths.Vec) maths.Vec {
	gap := from.Y - sea.height(from.X, from.Z)
	if gap > 0 || gap == 0 && point.Y < from.Y {
		return normal
	}
	return normal.Neg()
}
