package manufacture

import (
	"fmt"
	"math"
	"sync"

	"github.com/janearc/celestial/measure"
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
	wood   []part
	blades []plantBlade
	extent maths.Box
}

// plantCell holds finite, shared prototypes for a kind/detail/variant.
type plantCell struct {
	wood, green, dry *mesh.Mesh
	bounds           maths.Box
}

// plantCellKey never includes tree height, world position or a frame number.
type plantCellKey struct {
	kind, detail string
	variant      int
}

var plantCellCache sync.Map

// cell reuses physical-size foliage instead of scaling leaves with branch
// length.
func (profile plantProfile) cell(detail string, variant int) (*plantCell, error) {
	if variant < spaceZero || variant >= int(plantValue("cell-variants")) ||
		(detail != "clumps" && detail != "quick" && detail != "demo") {
		return nil, fmt.Errorf("manufacture: unsupported plant cell key")
	}
	key := plantCellKey{kind: profile.Kind, detail: detail, variant: variant}
	if known, found := plantCellCache.Load(key); found {
		return known.(*plantCell), nil
	}
	plan, err := profile.cellPlan(sow(uint64(variant)))
	if err != nil {
		return nil, err
	}
	made, err := plan.meshes(detail)
	if err != nil {
		return nil, err
	}
	actual, _ := plantCellCache.LoadOrStore(key, made)
	return actual.(*plantCell), nil
}

// blade is an authored, closed leaf; its camber is a share of blade width.
func (profile plantProfile) blade() broadLeaf {
	return broadLeaf{length: profile.LeafLength, width: profile.LeafWidth,
		thickness: profile.LeafThickness, camber: profile.LeafWidth * plantValue("leaf-camber"),
		outline: profile.LeafShape, lobes: profile.Lobes, depth: profile.LobeDepth}
}

// cellPlan keeps attachment geometry independent of how it will be drawn.
func (profile plantProfile) cellPlan(from seed) (plantCellPlan, error) {
	var plan plantCellPlan
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
	tube, err := newTube(length,
		measure.Metres(radius), measure.Metres(radius*plantValue("twig-taper")))
	if err != nil {
		return err
	}
	pose, err := spaceAlong(root, run)
	if err != nil {
		return err
	}
	fine := cut{around: int(plantValue("twig-sides"))}
	plan.wood = append(plan.wood, part{shape: tube, fine: fine, space: pose, of: body})
	return nil
}

// twigs is a continuing axis with contracting lateral axes and attached leaves.
func (profile plantProfile) twigs(plan *plantCellPlan, from seed) error {
	for index := range profile.TwigCount {
		share := float64(index+spaceOne) / float64(profile.TwigCount+spaceOne)
		root := maths.YAxis.Scale(profile.CellLength * share)
		angle := float64(index) * plantValue("twig-divergence-deg") * maths.Degree
		sine, cosine := math.Sincos(angle)
		run := profile.TwigLength * (spaceOne - plantValue("twig-contraction")*share)
		direction := maths.V(cosine, plantValue("twig-rise"), -sine).Unit()
		end := root.Add(direction.Scale(run))
		if err := plan.rod(root, end, profile.TwigRadius); err != nil {
			return err
		}
		profile.twigLeaves(plan, from.sprout(index, spaceZero), root, direction, run)
	}
	return nil
}

// twigLeaves roots paired blades in the twig, rather than floating in a crown.
func (profile plantProfile) twigLeaves(plan *plantCellPlan, from seed,
	root, direction maths.Vec, length float64,
) {
	frame := maths.Upright(direction)
	blade := profile.blade()
	for rank := range profile.LeafPairs {
		share := float64(rank+spaceOne) / float64(profile.LeafPairs+spaceOne)
		at := root.Add(direction.Scale(length * share))
		for _, side := range []float64{-spaceOne, spaceOne} {
			local := maths.V(side, plantValue("leaf-lift"),
				from.swing(growing, rank, spaceZero)*plantValue("leaf-turn-variation")).Unit()
			along := frame.Out(local)
			pose := maths.NewSubspace(at, maths.Upright(along), spaceOne)
			plan.blades = append(plan.blades, plantBlade{shape: blade, pose: pose})
		}
	}
}

// rosette attaches stiff green blades and a separate downward old-leaf skirt.
func (profile plantProfile) rosette(plan *plantCellPlan, from seed) {
	for index := range profile.RosetteLeaves {
		turn := fullTurn * float64(index) / float64(profile.RosetteLeaves)
		jitter := from.swing(growing, index, spaceZero) * plantValue("rosette-angle-variation")
		profile.rosetteBlade(plan, turn, profile.RosetteAngle+jitter, false)
		profile.rosetteBlade(plan, turn, plantValue("skirt-angle-deg")+jitter, true)
	}
}

// rosetteBlade uses the same physical blade dimensions for both age roles.
func (profile plantProfile) rosetteBlade(plan *plantCellPlan,
	turn, degrees float64, dry bool,
) {
	sinTurn, cosTurn := math.Sincos(turn)
	sinTilt, cosTilt := math.Sincos(degrees * maths.Degree)
	along := maths.V(sinTilt*cosTurn, cosTilt, -sinTilt*sinTurn)
	at := maths.YAxis.Scale(profile.CellLength)
	if dry {
		at = at.Sub(maths.YAxis.Scale(plantValue("skirt-drop")))
	}
	pose := maths.NewSubspace(at, maths.Upright(along), spaceOne)
	plan.blades = append(plan.blades, plantBlade{shape: profile.blade(), pose: pose, dry: dry})
}

// physical encloses every blade and twig, without building a finer leaf mesh.
func (plan *plantCellPlan) physical() {
	plan.extent = partBounds(plan.wood)
	for _, blade := range plan.blades {
		plan.extent = enclosing(plan.extent, transformed(blade.shape.bounds(), blade.pose))
	}
}

// meshes turns one physical plan into a detail-specific set of shared surfaces.
func (plan plantCellPlan) meshes(detail string) (*plantCell, error) {
	var wood, green, dry surface
	for _, piece := range plan.wood {
		wood = append(wood, piece.skin()...)
	}
	if detail == "clumps" {
		green = crownShape{plan.extent}.skin(cut{})
	} else {
		fine := cut{around: int(plantValue("leaf-sides-" + detail))}
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
	if made.wood, err = wood.meshed(); err != nil {
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
