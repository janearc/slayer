package trace

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// Taper the slab approximation as the two normals diverge or the chord
// reaches its supported length.
const (
	minTissueOpposition  = .90
	fullTissueOpposition = .98
	tissueTaperStart     = .75
	// Limit lateral entry offsets to one tenth of the normal depth.
	tissueFootprintSlope = .10
)

func (scene *staged) validateThinTissue(band maths.Band, inside bool) error {
	for _, placed := range scene.crowd.things {
		material := placed.thing.Material
		if material == nil || !material.ThinTissue.Enabled() {
			continue
		}
		if band.Nanometres != 0 {
			return fmt.Errorf("trace: thin tissue material %q requires the visible RGB renderer", material.Name)
		}
		if inside || len(scene.fields) != 0 || scene.oceanSun != nil || scene.oceanSpotInscatter {
			return fmt.Errorf("trace: thin tissue material %q requires the ordinary air-side lighting path", material.Name)
		}
		skin, meshed := placed.thing.Shape.(*mesh.Mesh)
		limit := material.ThinTissue.MaxThickness() / placed.scale
		if !meshed || skin == nil || !skin.Closed() || placed.thing.Clear() ||
			placed.scale <= 0 || !finite(placed.scale) || limit <= 0 || !finite(limit) {
			return fmt.Errorf("trace: thin tissue material %q requires a closed shadow-casting mesh with positive scale", material.Name)
		}
		if material.Through != 0 || material.ClearMedium != nil || material.Metal != nil ||
			material.Fibre != nil || material.WakeFoam != nil || !finite(material.Gloss) ||
			material.Dense() < 1 || !finite(2*material.Dense()) {
			return fmt.Errorf("trace: thin tissue material %q requires opaque dielectric surface optics", material.Name)
		}
	}
	return nil
}

// thinTissueLight samples a footprint of entry points from a surface hit.
func (scene *staged) thinTissueLight(met hit, seen look, mirrorShare float64) maths.Light {
	if !met.on.meshed {
		return maths.Light{}
	}
	placed := &met.thing
	material := placed.thing.Material
	skin := placed.thing.Shape.(*mesh.Mesh)
	toEye := placed.at.Orientation.In(seen.ray.Dir.Neg())
	viewFace, viewNormal := tissueNormals(skin, met.on.met)
	viewed := viewNormal.Dot(toEye)
	if viewFace.Dot(toEye) <= 0 || viewed <= 0 {
		return maths.Light{}
	}
	index := material.Dense()
	// The reverse air-side direction gives the escaping ray's Fresnel
	// loss. Cap the remainder by the existing mirror budget.
	escape := min(1-maths.Fresnel(viewed, index), 1-mirrorShare)
	if escape <= 0 {
		return maths.Light{}
	}
	axis := viewNormal.Neg()
	limit := material.ThinTissue.MaxThickness() / placed.scale
	sample := func(toward maths.Vec) maths.Light {
		return scene.thinTissueColumn(placed, skin, met.on.met, axis, toward, limit, escape)
	}
	sum := sample(axis)
	if !skin.Smooth() {
		return sum
	}
	first, second := across(axis)
	tangents := [...]maths.Vec{first, first.Neg(), second, second.Neg()}
	for _, tangent := range tangents {
		toward := axis.Add(tangent.Scale(tissueFootprintSlope)).Unit()
		sum = sum.Add(sample(toward))
	}
	// Misses and occluded columns retain their zero share in the mean.
	return sum.Scale(1 / float64(1+len(tangents)))
}

// thinTissueColumn follows the first geometric crossing for each sample.
// Project slanted chords onto the axis for the slab thickness alone.
func (scene *staged) thinTissueColumn(placed *placedThing, skin *mesh.Mesh,
	from mesh.Met, axis, toward maths.Vec, limit, escape float64) maths.Light {
	profile := placed.thing.Material.ThinTissue
	projection := 1.0
	if toward != axis {
		projection = min(1, axis.Dot(toward))
	}
	chord, found := skin.InteriorChord(mesh.ChordQuery{
		From: from, Toward: toward, Limit: min(math.MaxFloat64, limit/projection),
	})
	if !found {
		return maths.Light{}
	}
	entryFace, entryNormal := tissueNormals(skin, chord.Exit)
	parallel := maths.Smooth(minTissueOpposition, fullTissueOpposition, axis.Dot(entryNormal))
	length := chord.Exit.Along()
	thickness := length * projection
	taper := 1 - maths.Smooth(tissueTaperStart, 1, thickness/limit)
	if parallel == 0 || taper == 0 {
		return maths.Light{}
	}
	through := profile.Transmittance(thickness * placed.scale)
	if through == (maths.Light{}) {
		return maths.Light{}
	}
	entry := placed.at.Out(chord.End)
	turn := placed.at.Orientation
	path := maths.Ray{At: placed.at.Out(chord.Start), Dir: turn.Out(toward.Unit())}
	if scene.crowd.blockedExcept(path, length*placed.scale, placed.thing) {
		return maths.Light{}
	}
	incident := scene.thinTissueEntry(entry,
		turn.Out(entryFace), turn.Out(entryNormal), placed.thing.Material.Dense())
	return incident.Times(through).Scale(profile.Share() * parallel * taper * escape)
}

// tissueNormals keeps geometric boundary checks tied to the triangle.
// Smooth mesh normals guide the optics when they point to its exterior.
func tissueNormals(skin *mesh.Mesh, met mesh.Met) (face, normal maths.Vec) {
	a, b, c := skin.Corners(met.Face())
	face = b.Sub(a).Cross(c.Sub(a)).Unit()
	if skin.Smooth() {
		normal = skin.NormalAt(met)
		if normal.Dot(face) > 0 {
			return face, normal
		}
	}
	return face, face
}

// thinTissueEntry returns incident irradiance over pi after the entry
// interface. The geometric face sets the ray origin and exterior hemisphere.
func (scene *staged) thinTissueEntry(point, face, normal maths.Vec, index float64) maths.Light {
	from := scene.lifted(point, face)
	var sum maths.Light
	var room [lampsEvery]lit
	worth := func(toward maths.Vec, light maths.Light) float64 {
		if face.Dot(toward) <= 0 {
			return 0
		}
		return light.Luminance() * max(0, normal.Dot(toward))
	}
	for _, one := range scene.asked(room[:], point, normal, worth) {
		cosine := normal.Dot(one.toward)
		if cosine <= 0 || face.Dot(one.toward) <= 0 || one.light == (maths.Light{}) {
			continue
		}
		visible := scene.unhidden(one.index, from, one.toward, one.far)
		coupling := cosine * (1 - maths.Fresnel(cosine, index))
		sum = sum.Add(one.light.Times(visible).Scale(one.counts * coupling / math.Pi))
	}
	if scene.lighting == nil {
		return sum
	}
	first, second := across(normal)
	for _, look := range skyWays {
		way := first.Scale(look.X).Add(second.Scale(look.Y)).Add(normal.Scale(look.Z))
		if face.Dot(way) <= 0 {
			continue
		}
		visible := scene.transmit(from, way, math.Inf(1))
		coupling := (1 - maths.Fresnel(look.Z, index)) / skyLooks
		sum = sum.Add(scene.lighting.Light(way).Times(visible).Scale(coupling))
	}
	return sum
}
