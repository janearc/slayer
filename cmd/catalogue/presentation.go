package main

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// addCove keeps the specimen and scale figure in metres. Only the studio
// expands with the specimen; neither the human nor the model is resized.
func addCove(world *thing.World, bounds maths.Box, camera maths.Vec,
	detail string,
) (maths.Box, int, error) {
	human, err := makeSpecimen("human-scale-2m", 1, detail)
	if err != nil {
		return bounds, 0, err
	}
	right := camera.Cross(maths.YAxis).Unit()
	back := maths.YAxis.Cross(right)
	half := bounds.Max.Sub(bounds.Min).Scale(.5)
	centre := bounds.Min.Add(bounds.Max).Scale(.5)
	clearance := math.Abs(right.X)*half.X + math.Abs(right.Z)*half.Z + .75
	at := centre.Add(right.Scale(clearance))
	at.Y = bounds.Min.Y
	parts, err := human.Things(maths.NewSubspace(at, maths.Identity, 1))
	if err != nil {
		return bounds, 0, err
	}
	faces := 0
	for _, part := range parts {
		look := *part.Material
		look.Name = "two-metre scale human"
		part.Material = &look
		world.Add(part)
		faces += storedFaces(part.Shape)
	}
	humanBounds := human.Bounds()
	shown := unionBounds(bounds, maths.Box{Min: humanBounds.Min.Add(at), Max: humanBounds.Max.Add(at)})
	radius := max(2, shown.Max.Sub(shown.Min).Len()*.5)
	floor := maths.V(centre.X, bounds.Min.Y, centre.Z)
	count, err := coveSurfaces(world, floor, right, back, radius)
	return shown, faces + count, err
}

// unionBounds frames both physical objects without including the backdrop.
func unionBounds(a, b maths.Box) maths.Box {
	return maths.Box{Min: maths.V(min(a.Min.X, b.Min.X), min(a.Min.Y, b.Min.Y), min(a.Min.Z, b.Min.Z)),
		Max: maths.V(max(a.Max.X, b.Max.X), max(a.Max.Y, b.Max.Y), max(a.Max.Z, b.Max.Z))}
}

// coveSurfaces makes a floor tangent to a quarter-circle sweep and back wall.
// Its dimensions and neutral studio reflectances are presentation choices.
func coveSurfaces(world *thing.World, floor, right, back maths.Vec,
	radius float64,
) (int, error) {
	point := func(x, y, z float64) maths.Vec {
		return floor.Add(right.Scale(x)).Add(maths.YAxis.Scale(y)).Add(back.Scale(z))
	}
	width, start, curve := radius*8, radius*2, radius*2
	vertices := []maths.Vec{point(-width, 0, -width), point(width, 0, -width),
		point(-width, 0, start), point(width, 0, start)}
	ground, err := mesh.New(vertices, [][3]int{{0, 2, 1}, {1, 2, 3}}, nil, false)
	if err != nil {
		return 0, err
	}
	world.Add(thing.Thing{Shape: ground, Motion: thing.Still(maths.Local()),
		Material: &thing.Material{Name: "cove checkerboard",
			Texture: texture.Checker(max(1, radius/12), maths.Grey(.85), maths.Grey(.55))}})
	var sweep []maths.Vec
	var normals []maths.Vec
	const steps = 48
	for index := range steps + 1 {
		angle := math.Pi * .5 * float64(index) / steps
		y, z := curve*(1-math.Cos(angle)), start+curve*math.Sin(angle)
		sweep = append(sweep, point(-width, y, z), point(width, y, z))
		normal := maths.YAxis.Scale(math.Cos(angle)).Sub(back.Scale(math.Sin(angle)))
		normals = append(normals, normal, normal)
	}
	sweep = append(sweep, point(-width, width, start+curve), point(width, width, start+curve))
	normals = append(normals, back.Neg(), back.Neg())
	var triangles [][3]int
	for index := range steps + 1 {
		first := index * 2
		triangles = append(triangles, [3]int{first, first + 2, first + 1}, [3]int{first + 1, first + 2, first + 3})
	}
	wall, err := mesh.New(sweep, triangles, normals, true)
	if err != nil {
		return 0, err
	}
	world.Add(thing.Thing{Shape: wall, Motion: thing.Still(maths.Local()),
		Material: thing.Diffuse("white infinity cove", maths.Grey(.9))})
	return ground.FaceCount() + wall.FaceCount(), nil
}
