package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// A city chunk is an authored preparation budget, below the ordinary flat
// specimen limit. Disjoint geometry shares a material mesh rather than one
// GPU acceleration structure per building, window, lot or road segment.
type cityBatch struct {
	limit     int
	placement CityBatchPlacementMode
	pending   map[slot]surface
	members   map[slot][]shape.MeshMember
	counts    map[slot]int
	trouble   error
	// modules keeps each street-level module's one mesh, made on first use.
	modules map[cityModule]*mesh.Mesh
}

// newCityBatch allocates material accumulators without geometry prototypes.
func newCityBatch() *cityBatch {
	return &cityBatch{limit: int(cityValue("chunk-faces")), pending: map[slot]surface{},
		members: map[slot][]shape.MeshMember{}, counts: map[slot]int{},
		modules: map[cityModule]*mesh.Mesh{}}
}

// add transforms a bounded building directly into material accumulators.
// Fusing its nested transforms avoids transient surfaces; the finished batch
// still owns the same unique material meshes and shares no building geometry.
func (batch *cityBatch) add(parts []part, place maths.Subspace) {
	for _, piece := range parts {
		if batch.placement == CityBatchNested {
			batch.skin(piece.of, piece.skin().placed(place))
			continue
		}
		skin := piece.shape.skin(piece.fine)
		batch.placedSkin(piece.of, skin, piece.space, place)
	}
}

// placedSkin applies the part and building frames in their original order
// while appending directly to the material accumulator. The intermediate
// surfaces made by part.skin and surface.placed are not retained by a batch.
func (batch *cityBatch) placedSkin(material slot, skin surface,
	local, parent maths.Subspace,
) {
	if batch.trouble != nil {
		// Match skin's early return after the old placement path had evaluated
		// both transforms for every corner.
		for _, triangle := range skin {
			for _, point := range triangle {
				_ = parent.Out(local.Out(point))
			}
		}
		return
	}
	if len(batch.pending[material])+len(skin) > batch.limit {
		batch.flush(material)
	}
	pending := batch.pending[material]
	for _, triangle := range skin {
		pending = append(pending, face{
			parent.Out(local.Out(triangle[0])),
			parent.Out(local.Out(triangle[1])),
			parent.Out(local.Out(triangle[2])),
		})
	}
	batch.pending[material] = pending
	batch.counts[material]++
}

// skin preserves the first failure and never splits an architectural solid.
func (batch *cityBatch) skin(material slot, skin surface) {
	if batch.trouble != nil {
		return
	}
	if len(batch.pending[material])+len(skin) > batch.limit {
		batch.flush(material)
	}
	batch.pending[material] = append(batch.pending[material], skin...)
	batch.counts[material]++
}

// flush validates every face with the ordinary native mesh constructor.
func (batch *cityBatch) flush(material slot) {
	if batch.trouble != nil || len(batch.pending[material]) == 0 {
		return
	}
	polygon, err := batch.pending[material].meshed()
	if err != nil {
		batch.trouble = err
		return
	}
	batch.members[material] = append(batch.members[material],
		shape.MeshMember{Mesh: polygon, Space: maths.Local()})
	batch.pending[material] = nil
}

// finish publishes complete material hierarchies or returns no specimen.
func (batch *cityBatch) finish() (good, error) {
	for material := range batch.pending {
		batch.flush(material)
	}
	if batch.trouble != nil {
		return good{}, batch.trouble
	}
	made := good{dressed: cityLooks(), groups: map[slot]*shape.MeshGroup{},
		groupParts: batch.counts}
	for material, members := range batch.members {
		group, err := shape.NewMeshGroup(members)
		if err != nil {
			return good{}, err
		}
		made.groups[material] = group
	}
	return made, nil
}
