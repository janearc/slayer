package named

import (
	"fmt"
	"io"
	"math"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/cast"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// the dressed ring's floor: the hexes round it, the seed, scale and sea
// of its biome, how far its land rises toward the axis, and the lines
// between its tiles, its gloss and index
const (
	dressAround            = 360
	dressSeed              = 1806
	dressScale, dressSea   = 2.2, 0.02
	dressRise              = 0.35
	dressLines             = 0.2
	dressGloss, dressIndex = 0.1, 1.5
)

// dressRing dresses the inner face alone: hexes round the inside in the
// earth's paints, with a little of the mirror left on top so the flare
// still shimmers across them; and says what it made.
//
// cast from the hex lattice itself, centre to centre, so the ground and
// its tiles agree; the land rises toward the axis by its height above the
// sea, the sea lies flat, and the paint is the same two fields read at
// each tile, so the biomes clump
func dressRing(world *thing.World, said io.Writer) error {
	tiles := hexes.Band{Radius: ringRadius - ringThick/2,
		Width: ringWidth, Around: dressAround}
	fields := biome.Fields{Tiling: tiles, Seed: dressSeed,
		Scale: dressScale, Sea: dressSea}
	painted := biome.Make(fields)
	fmt.Fprintf(said, "the world: %d rivers, %d lakes, %d volcanoes, "+
		"%d cities\n", painted.Rivers, painted.Lakes, painted.Volcanoes,
		painted.Cities)
	inward := func(point maths.Vec) maths.Vec {
		return maths.V(-point.X, 0, -point.Z)
	}
	rise := func(_ hexes.Tile, centre maths.Vec) float64 {
		return dressRise * fields.Above(centre)
	}
	floor, count, err := cast.Mesh(tiles, inward, rise, true)
	if err != nil {
		return err
	}
	fmt.Fprintf(said, "the floor: %d tiles cast into %d triangles\n",
		len(tiles.Tiles()), count)
	world.Add(thing.Thing{Shape: floor, Material: &thing.Material{
		Name: "ring floor",
		Texture: texture.Tiled{Tiling: tiles, Paints: painted,
			Lines: dressLines},
		Shine: texture.Tiled{Tiling: tiles, Paints: painted.Shine()},
		Gloss: dressGloss, Index: dressIndex}})
	return nil
}

// ringStrip is one strip of the ring's cross-section: a line across it,
// as distance out and height, and the way its face looks in those terms.
type ringStrip struct {
	line   []rz
	facing rz
}

// ringStrips are the ring's four strips round the axis: the inner face,
// the top rim, the outer face and the bottom rim, each cut into rows
// across.
func ringStrips() []ringStrip {
	inner, outer := ringRadius-ringThick/2, ringRadius+ringThick/2
	lo, hi := -ringWidth/2, ringWidth/2
	return []ringStrip{
		{stepped(rz{inner, lo}, rz{inner, hi}, ringRows), rz{-1, 0}},
		{stepped(rz{inner, hi}, rz{outer, hi}, 1), rz{0, 1}},
		{stepped(rz{outer, hi}, rz{outer, lo}, ringRows), rz{1, 0}},
		{stepped(rz{outer, lo}, rz{inner, lo}, 1), rz{0, -1}},
	}
}

// stepped is the line from one point of the cross-section to another in
// rows equal steps, both ends included.
func stepped(from, to rz, rows int) []rz {
	var line []rz
	for j := 0; j <= rows; j++ {
		share := float64(j) / float64(rows)
		line = append(line, rz{from.out + (to.out-from.out)*share,
			from.along + (to.along-from.along)*share})
	}
	return line
}

// ringSolid is the ring as a solid of triangles, in the strips keep keeps,
// each stepped round the axis around times. A strip has its own vertices,
// so its normals are its own:
//
// smooth round the ring, so the mirror bends each reflection as a curved
// surface would, and hard at the corners, where one face meets the next.
// Every triangle is turned to face away from the middle of the
// cross-section, so the solid is wound outward throughout.
func ringSolid(around int, keep func(strip int) bool) (*mesh.Mesh, error) {
	var verts, normals []maths.Vec
	var faces [][3]int
	for number, strip := range ringStrips() {
		if !keep(number) {
			continue
		}
		base := len(verts)
		verts, normals = strip.round(around, verts, normals)
		faces = strip.faces(around, base, verts, faces)
	}
	return mesh.New(verts, faces, normals, true)
}

// round is the strip's points at every step round, with its facing as
// their normals, added to verts and normals.
func (strip ringStrip) round(around int, verts,
	normals []maths.Vec) ([]maths.Vec, []maths.Vec) {
	for i := range around {
		angle := 2 * math.Pi * float64(i) / float64(around)
		cosine, sine := math.Cos(angle), math.Sin(angle)
		for _, point := range strip.line {
			verts = append(verts, maths.V(point.out*cosine,
				point.along, point.out*sine))
			normals = append(normals, maths.V(strip.facing.out*cosine,
				strip.facing.along, strip.facing.out*sine))
		}
	}
	return verts, normals
}

// faces are the strip's triangles, two to each quad between one step
// round and the next, each turned to face outward, added to faces.
func (strip ringStrip) faces(around, base int, verts []maths.Vec,
	faces [][3]int) [][3]int {
	count := len(strip.line)
	at := func(i, j int) int { return base + (i%around)*count + j }
	for i := range around {
		angle := 2 * math.Pi * (float64(i) + half) / float64(around)
		outward := maths.V(strip.facing.out*math.Cos(angle),
			strip.facing.along, strip.facing.out*math.Sin(angle))
		for j := 0; j+1 < count; j++ {
			quad := [4]int{at(i, j), at(i+1, j), at(i+1, j+1),
				at(i, j+1)}
			for _, corners := range [][3]int{{quad[0], quad[1], quad[2]},
				{quad[0], quad[2], quad[3]}} {
				first, second := verts[corners[0]], verts[corners[1]]
				third := verts[corners[2]]
				normal := second.Sub(first).Cross(third.Sub(first))
				if normal.Dot(outward) < 0 {
					corners[1], corners[2] = corners[2], corners[1]
				}
				faces = append(faces, corners)
			}
		}
	}
	return faces
}

// half is the middle of a step round, defined.
const half = 0.5
