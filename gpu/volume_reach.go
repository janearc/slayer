package gpu

import (
	"encoding/binary"
	"math"
)

// The box of a volume in the world, for the rays that miss it. A shadow
// ray used to take itself into every volume's own space and clip itself to
// the volume's shape before it could say it missed: with seventy volumes in
// a haze, seventy times for every ray.
//
// Each volume's row now holds a box in
// the world that its shape lies inside, and the kernel asks that box first
// (volume_missed in trace.metal), as the tracer does (trace/volumebox.go,
// meets). A ray that misses the box misses the shape, so it has no span,
// and the march that followed gave it no thickness.
//
// The box is read from the volume's row as the kernel's span reads it, one
// shape for each way span clips a ray, so the two stay together by what
// the row holds and not by the kinds that wrote it.

const (
	// volumeMargin is how much a box is grown on every side, as a share
	// of its longest side, so that a ray the kernel's float arithmetic
	// puts on the shape's face is not culled by the box.
	volumeMargin = 1e-3
	// volumeRounding is the same growth as a share of the farthest a
	// coordinate of the box lies from the world's origin, which is what
	// a float32 coordinate there is rounded by.
	volumeRounding = 1e-6
	// axisTolerance is how far a volume's axes may be from a turn, which
	// has them of length one and square to each other, before its box is
	// not taken, and every ray is let into it as before.
	axisTolerance = 1e-4
	// cylinderKind is the sub kind a boxed medium's info.x names for a
	// cylinder, and sphereKind for a ball: span clips them, as it does
	// not clip the cells' box.
	cylinderKind = 2
	sphereKind   = 1
)

// volumeNumber is the part-th number of a volume's float field.
func volumeNumber(row []byte, name string, part int) float64 {
	at := fieldOffset(volumeStruct, name) + part*wordBytes
	return float64(math.Float32frombits(binary.LittleEndian.Uint32(row[at:])))
}

// volumeWhole is the part-th number of a volume's int field.
func volumeWhole(row []byte, name string, part int) int {
	at := fieldOffset(volumeStruct, name) + part*wordBytes
	return int(int32(binary.LittleEndian.Uint32(row[at:])))
}

// volumeCorner is the volume's three numbers of a field, as a vector.
func volumeCorner(row []byte, name string) [3]float64 {
	return [3]float64{volumeNumber(row, name, 0),
		volumeNumber(row, name, 1), volumeNumber(row, name, 2)}
}

// volumeLocal is the box, in the volume's own space, that span clips a ray
// to: the cells' box for a boxed medium, a ring, a propeller and a fizz;
// a cylinder's; and the ball of its radius for every other volume.
func volumeLocal(row []byte) (low, high [3]float64) {
	kind, sub := volumeWhole(row, "info", 2), volumeWhole(row, "info", 0)
	switch {
	case kind == boxedKind && sub == cylinderKind:
		across, half := volumeNumber(row, "puff", 1),
			volumeNumber(row, "puff", 2)/2
		return [3]float64{-across, -half, -across},
			[3]float64{across, half, across}
	case kind == boxedKind && sub != sphereKind, kind == ringKind,
		kind == propellerKind, kind == fizzKind:
		return volumeCorner(row, "low"), volumeCorner(row, "high")
	}
	radius := volumeNumber(row, "puff", 0)
	return [3]float64{-radius, -radius, -radius},
		[3]float64{radius, radius, radius}
}

// volumeTurned is whether a volume's axes are a turn: each of length one,
// and square to each other, within axisTolerance, so that a point in the
// volume's space is taken to the world by the same axes as a way is taken
// from it.
func volumeTurned(axes [3][3]float64) bool {
	for first, one := range axes {
		for second, other := range axes[first:] {
			var along float64
			for part := range one {
				along += one[part] * other[part]
			}
			want := 0.0
			if second == 0 {
				want = 1
			}
			if math.Abs(along-want) > axisTolerance {
				return false
			}
		}
	}
	return true
}

// volumeReach is the box of a volume's row in the world, grown by the
// margins, and whether there is one: not for a volume whose axes are not
// a turn, whose scale is not above nought, or whose box is not finite.
func volumeReach(row []byte) (low, high [3]float64, walled bool) {
	origin := volumeCorner(row, "origin")
	scale := volumeNumber(row, "origin", 3)
	axes := [3][3]float64{volumeCorner(row, "x"), volumeCorner(row, "y"),
		volumeCorner(row, "z")}
	if !(scale > 0) || !volumeTurned(axes) {
		return low, high, false
	}
	near, far := volumeLocal(row)
	for part := range low {
		low[part], high[part] = math.Inf(1), math.Inf(-1)
	}
	for corner := range reachCorners {
		at := volumeOut(origin, scale, axes, [2][3]float64{near, far},
			corner)
		for part := range at {
			low[part], high[part] = min(low[part], at[part]),
				max(high[part], at[part])
		}
	}
	return reachGrown(low, high)
}

// reachCorners is how many corners a box has.
const reachCorners = 8

// volumeOut is a corner of the box whose low and high corners are in the
// volume's space, in the world: bit axis of corner chooses the high side
// on that axis.
func volumeOut(origin [3]float64, scale float64, axes [3][3]float64,
	sides [2][3]float64, corner int) [3]float64 {
	out := origin
	for part := range out {
		for axis := range axes {
			side := sides[corner>>axis&1]
			out[part] += scale * side[axis] * axes[axis][part]
		}
	}
	return out
}

// reachGrown is a box made larger by the margins on every side, and
// whether it is finite.
func reachGrown(low, high [3]float64) ([3]float64, [3]float64, bool) {
	var longest, farthest float64
	for part := range low {
		longest = max(longest, high[part]-low[part])
		farthest = max(farthest, math.Abs(low[part]), math.Abs(high[part]))
	}
	by := volumeMargin*longest + volumeRounding*farthest
	for part := range low {
		low[part], high[part] = low[part]-by, high[part]+by
		if math.IsInf(low[part], 0) || math.IsNaN(low[part]) ||
			math.IsInf(high[part], 0) || math.IsNaN(high[part]) {
			return low, high, false
		}
	}
	return low, high, true
}

// boxVolumes writes into each volume's row the box it keeps in the world:
// the low corner in world_low with a one in its w, the high corner in
// world_high. A volume with no box has nought in both and is let every
// ray, as before.
func (encoded *scene) boxVolumes() {
	size := volumeStruct.size()
	for first := 0; first+size <= len(encoded.volumes.bytes); first += size {
		row := encoded.volumes.bytes[first : first+size]
		low, high, walled := volumeReach(row)
		if !walled {
			continue
		}
		volumeStruct.patch(row, values{
			"world_low":  {low[0], low[1], low[2], 1},
			"world_high": {high[0], high[1], high[2]}})
	}
}
