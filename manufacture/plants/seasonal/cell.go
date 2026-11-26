package seasonal

import (
	"fmt"
	"math"
	"sync"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// plantBlade attaches a physical blade to a twig or rosette.
type plantBlade struct {
	shape broadLeaf
	pose  maths.Subspace
	dry   bool
}

// plantCellPlan is a physical twig system before surface-detail selection.
type plantCellPlan struct {
	wood   surface
	policy *Policy
	blades []plantBlade
	extent maths.Box
}

// plantCell holds finite, shared prototypes for a kind/detail/variant.
type plantCell struct {
	wood, green, dry *mesh.Mesh
	bounds           maths.Box
}

// plantCellShape contains only the inputs that change a foliage cell.
type plantCellShape struct {
	form, leafShape                                  string
	leafLength, leafWidth, leafThickness, lobeDepth  float64
	cellLength, twigLength, twigRadius, rosetteAngle float64
	lobes, twigCount, leafPairs, rosetteLeaves       int
}

// plantCellKey never includes placement, frame number or provenance.
type plantCellKey struct {
	shape   plantCellShape
	detail  string
	variant int
}

type plantCellStore struct{ cells sync.Map }

func (profile Profile) cellShape() plantCellShape {
	return plantCellShape{form: profile.Form, leafShape: profile.LeafShape,
		leafLength: profile.LeafLength, leafWidth: profile.LeafWidth,
		leafThickness: profile.LeafThickness, lobeDepth: profile.LobeDepth,
		cellLength: profile.CellLength, twigLength: profile.TwigLength,
		twigRadius: profile.TwigRadius, rosetteAngle: profile.RosetteAngle,
		lobes: profile.Lobes, twigCount: profile.TwigCount,
		leafPairs: profile.LeafPairs, rosetteLeaves: profile.RosetteLeaves}
}

// cell reuses foliage within its profile book; releasing the book and its
// plants releases the cache. Standalone profiles without a book do not cache.
func (profile Profile) cell(detail string, variant int) (*plantCell, error) {
	if variant < spaceZero || variant >= int(profile.value("cell-variants")) ||
		(detail != "clumps" && detail != "quick" && detail != "demo") {
		return nil, fmt.Errorf("manufacture: unsupported plant cell key")
	}
	key := plantCellKey{shape: profile.cellShape(), detail: detail, variant: variant}
	if profile.cells != nil {
		if known, found := profile.cells.cells.Load(key); found {
			return known.(*plantCell), nil
		}
	}
	plan, err := profile.cellPlan(profile.policy.Sow(uint64(variant)))
	if err != nil {
		return nil, err
	}
	made, err := plan.meshes(detail)
	if err != nil {
		return nil, err
	}
	if profile.cells == nil {
		return made, nil
	}
	actual, _ := profile.cells.cells.LoadOrStore(key, made)
	return actual.(*plantCell), nil
}

// blade is an authored, closed leaf; its camber is a share of blade width.
func (profile Profile) blade() broadLeaf {
	return broadLeaf{policy: profile.policy, length: profile.LeafLength, width: profile.LeafWidth,
		thickness: profile.LeafThickness, camber: profile.LeafWidth * profile.value("leaf-camber"),
		outline: profile.LeafShape, lobes: profile.Lobes, depth: profile.LobeDepth}
}

// cellPlan keeps attachment geometry independent of how it will be drawn.
func (profile Profile) cellPlan(from branching.Seed) (plantCellPlan, error) {
	plan := plantCellPlan{policy: profile.policy}
	if err := plan.rod(maths.V(spaceZero, spaceZero, spaceZero),
		maths.YAxis.Scale(profile.CellLength), profile.TwigRadius); err != nil {
		return plan, err
	}
	if profile.Form == "rosette" {
		profile.rosette(&plan, from)
	} else {
		if err := profile.twigs(&plan, from); err != nil {
			return plan, err
		}
	}
	plan.physical()
	return plan, nil
}

// rod adds the actual woody support beneath the leaf attachments.
func (plan *plantCellPlan) rod(root, end maths.Vec, radius float64) error {
	run := end.Sub(root)
	length := measure.Metres(run.Len())
	foot := measure.Metres(radius)
	tip := measure.Metres(radius * plan.policy.Value("twig-taper"))
	if err := geometry.CheckTube(length.InMetres(), foot.InMetres(), tip.InMetres()); err != nil {
		return err
	}
	pose, err := branching.SpaceAlong(root, run)
	if err != nil {
		return err
	}
	sides := int(plan.policy.Value("twig-sides"))
	skin := append(geometry.TubeWall[face](length.InMetres(), foot.InMetres(), tip.InMetres(), sides, false), geometry.TubeCaps[face](length.InMetres(), foot.InMetres(), tip.InMetres(), sides, false)...)
	plan.wood = append(plan.wood, geometry.Placed(skin, pose)...)
	return nil
}

// twigs is a continuing axis with contracting lateral axes and attached leaves.
func (profile Profile) twigs(plan *plantCellPlan, from branching.Seed) error {
	for index := range profile.TwigCount {
		share := float64(index+spaceOne) / float64(profile.TwigCount+spaceOne)
		root := maths.YAxis.Scale(profile.CellLength * share)
		angle := float64(index) * profile.value("twig-divergence-deg") * maths.Degree
		sine, cosine := math.Sincos(angle)
		run := profile.TwigLength * (spaceOne - profile.value("twig-contraction")*share)
		direction := maths.V(cosine, profile.value("twig-rise"), -sine).Unit()
		end := root.Add(direction.Scale(run))
		// The original twig builder fused this product and addition.
		end.Y = math.FMA(direction.Y, run, root.Y)
		if err := plan.rod(root, end, profile.TwigRadius); err != nil {
			return err
		}
		profile.twigLeaves(plan, from.Sprout(index, spaceZero), root, direction, run)
	}
	return nil
}

// twigLeaves roots paired blades in the twig, rather than floating in a crown.
func (profile Profile) twigLeaves(plan *plantCellPlan, from branching.Seed,
	root, direction maths.Vec, length float64,
) {
	frame := maths.Upright(direction)
	blade := profile.blade()
	for rank := range profile.LeafPairs {
		share := float64(rank+spaceOne) / float64(profile.LeafPairs+spaceOne)
		at := root.Add(direction.Scale(length * share))
		for _, side := range []float64{-spaceOne, spaceOne} {
			local := maths.V(side, profile.value("leaf-lift"),
				from.Swing(growing, rank, spaceZero)*profile.value("leaf-turn-variation")).Unit()
			along := frame.Out(local)
			pose := maths.NewSubspace(at, maths.Upright(along), spaceOne)
			plan.blades = append(plan.blades, plantBlade{shape: blade, pose: pose})
		}
	}
}

// rosette attaches stiff green blades and a separate downward old-leaf skirt.
func (profile Profile) rosette(plan *plantCellPlan, from branching.Seed) {
	for index := range profile.RosetteLeaves {
		turn := fullTurn * float64(index) / float64(profile.RosetteLeaves)
		jitter := from.Swing(growing, index, spaceZero) * profile.value("rosette-angle-variation")
		profile.rosetteBlade(plan, turn, profile.RosetteAngle+jitter, false)
		profile.rosetteBlade(plan, turn, profile.value("skirt-angle-deg")+jitter, true)
	}
}

// rosetteBlade uses the same physical blade dimensions for both age roles.
func (profile Profile) rosetteBlade(plan *plantCellPlan,
	turn, degrees float64, dry bool,
) {
	sinTurn, cosTurn := math.Sincos(turn)
	sinTilt, cosTilt := math.Sincos(degrees * maths.Degree)
	along := maths.V(sinTilt*cosTurn, cosTilt, -sinTilt*sinTurn)
	at := maths.YAxis.Scale(profile.CellLength)
	if dry {
		at = at.Sub(maths.YAxis.Scale(profile.value("skirt-drop")))
	}
	pose := maths.NewSubspace(at, maths.Upright(along), spaceOne)
	plan.blades = append(plan.blades, plantBlade{shape: profile.blade(), pose: pose, dry: dry})
}

// physical encloses every blade and twig, without building a finer leaf mesh.
func (plan *plantCellPlan) physical() {
	plan.extent = maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)), Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, tri := range plan.wood {
		plan.extent = geometry.GrowBox(plan.extent, tri[:])
	}
	for _, blade := range plan.blades {
		plan.extent = enclosing(plan.extent, transformed(blade.shape.bounds(), blade.pose))
	}
}

// meshes turns one physical plan into a detail-specific set of shared surfaces.
func (plan plantCellPlan) meshes(detail string) (*plantCell, error) {
	var green, dry surface
	if detail == "clumps" {
		green = geometry.Crown[face](plan.extent)
	} else {
		fine := cut{around: int(plan.policy.Value("leaf-sides-" + detail))}
		for _, blade := range plan.blades {
			skin := blade.shape.skin(fine).placed(blade.pose)
			if blade.dry {
				dry = append(dry, skin...)
			} else {
				green = append(green, skin...)
			}
		}
	}
	made := &plantCell{bounds: plan.extent}
	var err error
	if made.wood, err = plan.wood.meshed(); err != nil {
		return nil, err
	}
	if made.green, err = green.meshed(); err != nil {
		return nil, err
	}
	if len(dry) != spaceZero {
		if made.dry, err = dry.meshed(); err != nil {
			return nil, err
		}
	}
	if made.green == nil {
		return nil, fmt.Errorf("manufacture: empty plant foliage cell")
	}
	return made, nil
}

// CellBounds returns the physical extent of the requested cached cell.
func (profile Profile) CellBounds(detail string, variant int) (maths.Box, error) {
	cell, err := profile.cell(detail, variant)
	if err != nil {
		return maths.Box{}, err
	}
	return cell.bounds, nil
}
