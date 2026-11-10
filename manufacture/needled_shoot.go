package manufacture

import (
	_ "embed"
	"fmt"
	"math"
	"sync"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/scene/shape"
)

//go:embed needled-shoot.yml
var shootChoices []byte

var shootNumbers, shootNumbersError = numbers.Read("needled-shoot.yml", shootChoices)

// shootValue reads an authored model choice, including its provenance.
func shootValue(name string) float64 {
	if shootNumbersError != nil {
		panic(shootNumbersError)
	}
	value, err := shootNumbers.Get(name)
	if err != nil {
		panic(err)
	}
	return value.Value
}

// shootAxis is one continuing woody axis and its recursive parent.
type shootAxis struct {
	at, direction, normal maths.Vec
	length, radius        float64
	parent, order         int
}

// attachedNeedle keeps physical leaf dimensions separate from shoot size.
type attachedNeedle struct {
	root, tip, across, normal maths.Vec
	width, thickness          float64
	axis, rank                int
}

// shootPlan is the botanical scaffold. Detail changes only its surfaces.
type shootPlan struct {
	axes    []shootAxis
	needles []attachedNeedle
}

// shootPrototype is shared geometry, not a new copy for every branchlet.
type shootPrototype struct {
	wood, leaves *mesh.Mesh
	physical     maths.Box
	plan         shootPlan
}

type shootKey struct {
	style, bin, variant int
	detail              string
}

// shootKind exposes the same physical foliage used on trees as a specimen.
// It is useful for close views and for checking a recipe before scattering it.
type shootKind struct{ style int }

// made is the loose shoot at the default surface detail.
func (row shootKind) made(from seed) (good, error) { return row.madeAt(from, "quick") }

// madeAt changes only the shoot's surface detail, retaining physical leaves.
func (row shootKind) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	minimum, cell := shootValue("shoot-minimum-cell"), shootValue("shoot-cell-length")
	bin := int(math.Round(math.Log2(cell / minimum)))
	level := detail
	if level == "hull" {
		level = "quick"
	}
	shoot, err := prototype(row.style, bin, int(from.number%2), level)
	if err != nil {
		return good{}, err
	}
	dressed := redwoodRow().dressed
	if row.style == 2 {
		dressed = sequoiaRow().dressed
	}
	box := shoot.physical
	// A loose shoot lies horizontally, with its lowest point on the ground.
	space := maths.Subspace{Orientation: maths.About(maths.ZAxis, -math.Pi/2)}
	turned := transformed(box, space)
	space.Origin.Y = -turned.Min.Y
	space.Origin.X = -(turned.Min.X + turned.Max.X) / 2
	space.Origin.Z = -(turned.Min.Z + turned.Max.Z) / 2
	box = transformed(box, space)
	made := good{dressed: dressed, extent: &box}
	if detail == "hull" {
		made.parts = []part{{shape: crownShape{box}, of: leaf, space: maths.Subspace{}}}
		return made, nil
	}
	return made.withFoliageGroups(
		[]shape.MeshMember{{Mesh: shoot.wood, Space: space}},
		[]shape.MeshMember{{Mesh: shoot.leaves, Space: space}})
}

var shootPrototypes sync.Map

// prototype uses fixed physical size bins: scaling a twig must not turn
// a centimetre needle into a metre leaf. The variants are seeded microforms.
func prototype(style, bin, variant int, detail string) (*shootPrototype, error) {
	maximumBin := int(math.Ceil(math.Log2(
		shootValue("shoot-cell-length") / shootValue("shoot-minimum-cell"))))
	if style < 1 || style > spruceShootStyle || bin < 0 || bin > maximumBin ||
		variant < 0 || variant > 1 ||
		(detail != "clumps" && detail != "quick" && detail != "demo") {
		return nil, fmt.Errorf("manufacture: unsupported shoot prototype key")
	}

	key := shootKey{style, bin, variant, detail}
	if known, ok := shootPrototypes.Load(key); ok {
		return known.(*shootPrototype), nil
	}
	length := shootValue("shoot-minimum-cell") * math.Pow(2, float64(bin))
	plan := planShoot(style, length, Sow(uint64(variant)))
	var wood, leaves, physical surface
	// Both redwood ranks and all three sequoia rows land on tube vertices.
	// Finer cuts retain those vertices; leaf bases stay attached. Six is
	// the smallest common ring for two ranks and three rows. Spruce uses
	// eight vertices for four ranks at alternating half-rank phases.
	coarse := 6
	if style == spruceShootStyle {
		coarse = int(shootValue("spruce-wood-sides"))
	}
	fine := cut{around: coarse}
	if detail == "demo" {
		fine.around = 2 * coarse
	}
	for _, axis := range plan.axes {
		tube, err := newTube(measure.Metres(axis.length),
			measure.Metres(axis.radius),
			measure.Metres(axis.radius*shootValue("shoot-tip-radius")))
		if err != nil {
			return nil, err
		}
		space := axisSpace(axis)
		wood = append(wood, tube.skin(fine).placed(space)...)
		physical = append(physical,
			tube.skin(cut{around: 2 * coarse}).placed(space)...)
	}
	for _, needle := range plan.needles {
		leaves = append(leaves, needle.skin(detail)...)
		// Bound all detail levels, including the quick tetrahedra.
		physical = append(physical, needle.skin("quick")...)
		physical = append(physical, needle.skin("demo")...)
	}
	if style == spruceShootStyle && detail == "clumps" {
		bounds := partBounds([]part{{shape: fixedSurface(physical), space: maths.Local()}})
		leaves = crownShape{box: bounds}.skin(cut{})
	}
	var meshes [2]*mesh.Mesh
	for index, skin := range []surface{wood, leaves} {
		made, err := skin.meshed()
		if err != nil {
			return nil, err
		}
		meshes[index] = made
	}
	bounds := partBounds([]part{{shape: fixedSurface(physical), space: maths.Subspace{}}})
	made := &shootPrototype{wood: meshes[0], leaves: meshes[1], physical: bounds, plan: plan}
	actual, _ := shootPrototypes.LoadOrStore(key, made)
	return actual.(*shootPrototype), nil
}

// fixedSurface lets the existing bounds helper read already made vertices.
type fixedSurface surface

// skin supplies vertices already resolved by the shoot construction.
func (skin fixedSurface) skin(cut) surface { return surface(skin) }

// axisSpace retains a spray plane through successive lateral branches.
func axisSpace(axis shootAxis) maths.Subspace {
	side := axis.direction.Cross(axis.normal).Unit()
	return maths.Subspace{Origin: axis.at, Orientation: maths.Orientation{
		X: side, Y: axis.direction, Z: side.Cross(axis.direction),
	}}
}

// planShoot adapts ABOP chapter 2, figure 2.6: a continuing axis produces
// contracting lateral axes, alternating left and right in the spray plane.
// Child radii use Weber and Penn (1995), section 4.4. This is an adaptation,
// not their complete tree model. Choices live in needled-shoot.yml.
func planShoot(style int, length float64, from seed) shootPlan {
	plan := shootPlan{}
	var grow func(shootAxis)
	grow = func(axis shootAxis) {
		index := len(plan.axes)
		plan.axes = append(plan.axes, axis)
		plan.addNeedles(style, axis, index, from.sprout(index, 0))
		if axis.order >= int(shootValue("shoot-orders")) {
			return
		}
		count := int(shootValue("shoot-first-laterals"))
		if axis.order > 0 {
			count = int(shootValue("shoot-second-laterals"))
		}
		contraction := shootValue("shoot-internode-contraction")
		total := (1 - math.Pow(contraction, float64(count))) / (1 - contraction)
		position := 0.0
		space := axisSpace(axis)
		for node := range count {
			internode := math.Pow(contraction, float64(node))
			u := (position + internode/2) / total
			position += internode
			childLength := axis.length * shootValue("shoot-lateral-contraction") * (1 - u)
			if childLength < 2*needleSpacing(style) {
				continue
			}
			side := space.Orientation.X
			if style == 2 {
				side = maths.About(axis.direction, float64(node)*math.Pi*(3-math.Sqrt(5))).Out(side)
			} else if node%2 != 0 {
				side = side.Neg()
			}
			angle := (shootValue("shoot-lateral-angle") +
				shootValue("shoot-angle-variation")*from.swing(index, node, 0)) * maths.Degree
			direction := axis.direction.Scale(math.Cos(angle)).Add(
				side.Scale(math.Sin(angle))).Unit()
			// Twig radius follows the length ratio.
			radius := axis.radius * math.Pow(childLength/axis.length,
				shootValue("shoot-radius-power"))
			normal := axis.normal
			if style == 2 {
				normal = side.Cross(axis.direction).Unit()
			}
			grow(shootAxis{
				at:        axis.at.Add(axis.direction.Scale(u * axis.length)),
				direction: direction, normal: normal, length: childLength, radius: radius,
				parent: index, order: axis.order + 1,
			})
		}
	}
	grow(shootAxis{
		direction: maths.YAxis, normal: maths.ZAxis, length: length,
		radius: shootValue("shoot-base-radius") * length / shootValue("shoot-cell-length"), parent: -1,
	})
	return plan
}

// needleSpacing is an authored spacing, not implied by a published length.
func needleSpacing(style int) float64 {
	if style == spruceShootStyle {
		return shootValue("spruce-needle-spacing")
	}
	if style == 2 {
		return shootValue("sequoia-scale-spacing")
	}
	return shootValue("redwood-needle-spacing")
}

// addNeedles uses redwood's flat ranks, sequoia's appressed scale rows, or
// spruce needles around the stem. Redwood/sequoia lengths are in research
// data. Spruce dimensions, rows and alternating phases are authored choices.
func (plan *shootPlan) addNeedles(style int, axis shootAxis, index int, from seed) {
	rows, prefix := 2, "redwood-needle"
	// SI definitions: centimetres and millimetres converted to metres.
	low, high := research.Value("redwood-needle-minimum")/100, research.Value("redwood-needle-maximum")/100
	if style == 2 {
		rows, prefix = int(shootValue("sequoia-scale-rows")), "sequoia-scale"
		low, high = research.Value("sequoia-scale-minimum")/1000, research.Value("sequoia-scale-maximum")/1000
	}
	if style == spruceShootStyle {
		rows, prefix = int(shootValue("spruce-needle-rows")), "spruce-needle"
		low, high = shootValue(prefix+"-minimum"), shootValue(prefix+"-maximum")
	}
	count := max(1, int(axis.length/needleSpacing(style)))
	space := axisSpace(axis)
	angle := shootValue(prefix+"-angle") * maths.Degree
	for node := range count {
		u := (float64(node) + 0.5) / float64(count)
		for rank := range rows {
			out := space.Orientation.X
			if style == 2 || style == spruceShootStyle {
				phase := 0.0
				if style == spruceShootStyle && node%2 != 0 {
					phase = math.Pi / float64(rows)
				}
				out = maths.Rotation(axis.direction,
					fullTurn*float64(rank)/float64(rows)+phase).Out(out)
			} else if rank != 0 {
				out = out.Neg()
			}
			// Follow the woody taper, including its nonzero tip.
			radius := axis.radius * (1 - (1-shootValue("shoot-tip-radius"))*u)
			root := axis.at.Add(axis.direction.Scale(u * axis.length)).Add(
				out.Scale(radius))
			way := out.Scale(math.Sin(angle)).Add(axis.direction.Scale(math.Cos(angle)))
			long := low + (high-low)*from.draw(node, rank, 0)
			normal := axis.normal
			if style == 2 || style == spruceShootStyle {
				normal = axis.direction.Cross(out).Unit()
			}
			plan.needles = append(plan.needles, attachedNeedle{
				root: root, tip: root.Add(way.Scale(long)),
				across: normal.Cross(way).Unit(), normal: normal,
				width: shootValue(prefix + "-width"), thickness: shootValue(prefix + "-thickness"), axis: index, rank: rank,
			})
		}
	}
}

// skin resolves a needle as an open diamond, a closed tetrahedron, or a
// thin closed diamond. No representation contains a green woody sleeve.
func (needle attachedNeedle) skin(detail string) surface {
	middle := needle.root.Add(needle.tip).Scale(0.5)
	left, right := middle.Add(needle.across.Scale(needle.width/2)), middle.Sub(needle.across.Scale(needle.width/2))
	if detail == "clumps" {
		return surface{{needle.root, right, left}, {left, right, needle.tip}}
	}
	var skin surface
	if detail == "quick" {
		tip := needle.tip.Add(needle.normal.Scale(needle.thickness / 2))
		skin = surface{{needle.root, left, right}, {needle.root, tip, left}, {left, tip, right}, {right, tip, needle.root}}
	} else {
		top, bottom := middle.Add(needle.normal.Scale(needle.thickness/2)), middle.Sub(needle.normal.Scale(needle.thickness/2))
		ring := []maths.Vec{needle.root, left, needle.tip, right}
		for i, point := range ring {
			next := ring[(i+1)%4]
			skin = append(skin, face{top, point, next}, face{bottom, next, point})
		}
	}
	// The same construction serves both ranks and all three scale rows.
	volume := 0.0
	origin := skin[0][0]
	for _, face := range skin {
		volume += face[0].Sub(origin).Dot(face[1].Sub(origin).Cross(face[2].Sub(origin)))
	}
	if volume < 0 {
		for i := range skin {
			skin[i][1], skin[i][2] = skin[i][2], skin[i][1]
		}
	}
	return skin
}

// recursiveFoliage keeps the existing seeded woody scaffold and replaces
// terminal sleeves with repeated, recursively branched physical shoots.
func recursiveFoliage(made good, from seed, style int, detail string) (good, error) {
	var woodParts []part
	var wood, leaves []shape.MeshMember
	var bounds *maths.Box
	for index, piece := range made.parts {
		if piece.of != leaf {
			woodParts = append(woodParts, piece)
			continue
		}
		// Terminal foliage has a role, independent of part order.
		if !piece.terminal {
			continue
		}
		tube, ok := piece.shape.(tube)
		if !ok {
			return good{}, fmt.Errorf("manufacture: foliage needs a shoot axis")
		}
		length := tube.length.InMetres()
		cell := shootValue("shoot-cell-length")
		for node := 0; float64(node)*cell < length; node++ {
			remaining := math.Min(cell, length-float64(node)*cell)
			minimum := shootValue("shoot-minimum-cell")
			bin := max(0, int(math.Ceil(math.Log2(remaining/minimum))))
			variant := int(from.draw(index, node, 0) * 2)
			level := detail
			if detail == "hull" {
				level = "quick"
			}
			shoot, err := prototype(style, bin, variant, level)
			if err != nil {
				return good{}, err
			}
			// Adjacent sprays may roll into different planes.
			roll := from.swing(index, node, 1) *
				shootValue("shoot-roll") * maths.Degree
			space := maths.Subspace{
				Origin:      maths.V(0, float64(node)*cell, 0),
				Orientation: maths.About(maths.YAxis, roll),
			}.Within(piece.space)
			wood = append(wood, shape.MeshMember{Mesh: shoot.wood, Space: space})
			leaves = append(leaves, shape.MeshMember{Mesh: shoot.leaves, Space: space})
			physical := transformed(shoot.physical, space)
			if bounds == nil {
				bounds = &physical
			} else {
				joined := enclosing(*bounds, physical)
				bounds = &joined
			}
		}
	}
	if len(leaves) == 0 {
		return good{}, fmt.Errorf("manufacture: tree has no terminal shoots")
	}
	base := partBounds(woodParts)
	box := enclosing(base, *bounds)
	made.parts, made.extent = woodParts, &box
	if detail == "hull" {
		made.parts = []part{woodParts[0], {shape: crownShape{*bounds}, of: leaf, space: maths.Subspace{}}}
		return made, nil
	}
	var bark surface
	for _, part := range woodParts {
		bark = append(bark, part.skin()...)
	}
	barkMesh, err := bark.meshed()
	if err != nil {
		return good{}, err
	}
	wood = append(wood, shape.MeshMember{Mesh: barkMesh})
	return made.withFoliageGroups(wood, leaves)
}

// withFoliageGroups validates both materials before returning the specimen.
// A failed group leaves the caller with no partially assembled good.
func (made good) withFoliageGroups(wood, leaves []shape.MeshMember) (good, error) {
	groups := make(map[slot]*shape.MeshGroup, 2)
	for _, part := range []struct {
		of      slot
		members []shape.MeshMember
	}{{body, wood}, {leaf, leaves}} {
		group, err := shape.NewMeshGroup(part.members)
		if err != nil {
			return good{}, err
		}
		groups[part.of] = group
	}
	made.groups = groups
	return made, nil
}
