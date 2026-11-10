package emptyspace

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// EmptyBox certifies zero support throughout a local-space query box.
// It says nothing about geometry, other volumes or fields omitted at Build.
// Queries extending outside the field are unknown: no addressing rule is
// assumed there, even when all stored samples are zero.
func (mask *Mask) EmptyBox(query maths.Box) (bool, error) {
	if mask == nil || !mask.ready || !validBox(query) {
		return false, fmt.Errorf("emptyspace: invalid mask or query bounds")
	}
	if query.Min.X < mask.bounds.Min.X || query.Min.Y < mask.bounds.Min.Y ||
		query.Min.Z < mask.bounds.Min.Z || query.Max.X > mask.bounds.Max.X ||
		query.Max.Y > mask.bounds.Max.Y || query.Max.Z > mask.bounds.Max.Z {
		return false, nil
	}
	if len(mask.words) == 0 {
		return true, nil
	}
	first := mask.brickOf(query.Min)
	last := mask.brickOf(query.Max)
	return mask.emptyBricks(first, last), nil
}

// brickOf maps a clipped physical location to its cell-centred grid brick.
func (mask *Mask) brickOf(point maths.Vec) [3]int {
	size := mask.bounds.Max.Sub(mask.bounds.Min)
	var brick [3]int
	for axis, coordinate := range []struct{ point, low, size float64 }{
		{point.X, mask.bounds.Min.X, size.X}, {point.Y, mask.bounds.Min.Y, size.Y},
		{point.Z, mask.bounds.Min.Z, size.Z},
	} {
		share := (coordinate.point - coordinate.low) / coordinate.size
		index := math.Floor(share * float64(mask.cells[axis]) / float64(mask.options.BrickSide))
		brick[axis] = int(max(0, min(float64(mask.bricks[axis]-1), index)))
	}
	return brick
}

// emptyBricks includes one adjacent brick against query-boundary rounding.
func (mask *Mask) emptyBricks(first, last [3]int) bool {
	for axis := range first {
		first[axis] = max(0, first[axis]-1)
		last[axis] = min(mask.bricks[axis]-1, last[axis]+1)
	}
	xCount, yCount, zCount := last[0]-first[0]+1, last[1]-first[1]+1, last[2]-first[2]+1
	if xCount > mask.options.QueryBricks/yCount/zCount {
		// Too much proof work means unknown, so the caller keeps
		// marching.
		return false
	}
	for z := first[2]; z <= last[2]; z++ {
		for y := first[1]; y <= last[1]; y++ {
			for x := first[0]; x <= last[0]; x++ {
				index := (z*mask.bricks[1]+y)*mask.bricks[0] + x
				if mask.words[index/64]&(uint64(1)<<uint(index%64)) != 0 {
					return false
				}
			}
		}
	}
	return true
}

// PacketEmpty bounds every member's complete straight segment, not a pilot.
// Rays and distances must already use the field's local coordinates/units.
func (mask *Mask) PacketEmpty(rays []maths.Ray, begin, end float64) (bool, error) {
	if mask == nil || !mask.ready || len(rays) < 1 || len(rays) > mask.options.Rays ||
		!finite(begin) || !finite(end) || begin < 0 || end < begin {
		return false, fmt.Errorf("emptyspace: invalid packet or finite forward interval")
	}
	bounds := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)),
		Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, ray := range rays {
		if !finiteVector(ray.At) || !finiteVector(ray.Dir) || ray.Dir == (maths.Vec{}) {
			return false, fmt.Errorf("emptyspace: packet contains an invalid ray")
		}
		for _, point := range [2]maths.Vec{ray.Point(begin), ray.Point(end)} {
			if !finiteVector(point) {
				return false, fmt.Errorf("emptyspace: packet endpoint overflow")
			}
			bounds.Min = maths.V(min(bounds.Min.X, point.X), min(bounds.Min.Y, point.Y), min(bounds.Min.Z, point.Z))
			bounds.Max = maths.V(max(bounds.Max.X, point.X), max(bounds.Max.Y, point.Y), max(bounds.Max.Z, point.Z))
		}
	}
	return mask.EmptyBox(bounds)
}
