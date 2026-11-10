package oceantex

import "errors"

// Tile is one square texture: a square of sea seen from a camera straight
// above its centre. For any other point of view, see View.
type Tile struct {
	// Size is the texture's width and height in texels, a power of two.
	Size int
	// Extent is the width of the square of sea, in metres. East is right
	// and north is up.
	Extent float64
	// Altitude is the camera's height above the sea, in metres.
	Altitude float64
	// Seconds is the time, for the waves' motion.
	Seconds float64
	// Seed picks the sea. The same seed is the same sea.
	Seed uint64
}

// Render is a tile of a scene, and the wave field under it.
//
// The tile is one patch of waves whose grid is the texture's own, so a
// tile a few metres across shows every ripple the spectrum has and a tile
// hundreds of kilometres across shows none, only the glint they add up
// to. It tiles: its left edge meets its right.
func Render(scene Scene, tile Tile) (*Texture, *Field, error) {
	if !(tile.Altitude > 0) || !finite(tile.Altitude) || !finite(tile.Seconds) {
		return nil, nil, errors.New("oceantex: altitude must be positive")
	}
	l, err := scene.lighting()
	if err != nil {
		return nil, nil, err
	}
	patch, err := NewPatch(scene.Sea, tile.Size, tile.Extent, tile.Seed)
	if err != nil {
		return nil, nil, err
	}
	field := patch.At(tile.Seconds)
	gotAlong, gotAcross := patch.Resolved()

	n := tile.Size
	out := &Texture{Width: n, Height: n,
		Light: make([][3]float64, n*n), Exposure: 1}
	for iy := 0; iy < n; iy++ {
		for ix := 0; ix < n; ix++ {
			at := iy*n + ix
			x := (float64(ix)+0.5)/float64(n)*tile.Extent - tile.Extent/2
			y := (float64(iy)+0.5)/float64(n)*tile.Extent - tile.Extent/2
			eye := Vec{-x, -y, tile.Altitude}.unit()
			out.Light[at] = l.seen(eye, field.SlopeX[at], field.SlopeY[at],
				gotAlong, gotAcross)
		}
	}
	return out, field, nil
}
