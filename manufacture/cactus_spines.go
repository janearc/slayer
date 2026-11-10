package manufacture

import (
	"math"
	"sync"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// cactusSpineCache has only the two supported physical-spine detail keys.
var cactusSpineCache sync.Map

// cactusSpine shares one physical cone without scaling it to plant size.
func cactusSpine(detail string) (*mesh.Mesh, error) {
	if known, found := cactusSpineCache.Load(detail); found {
		return known.(*mesh.Mesh), nil
	}
	length := measure.Metres(cactusValue("spine-length"))
	radius := measure.Metres(cactusValue("spine-radius"))
	cone, err := newTube(length, radius, measure.Metres(spaceZero))
	if err != nil {
		return nil, err
	}
	fine := cut{around: int(cactusValue("spine-sides-" + detail))}
	made, err := cone.skin(fine).meshed()
	if err != nil {
		return nil, err
	}
	actual, _ := cactusSpineCache.LoadOrStore(detail, made)
	return actual.(*mesh.Mesh), nil
}

// spines adds bounded instances to the upright runs, leaving bends and caps
// bare.
func (plant saguaro) spines(made good, columns []cactusColumn, detail string) (good, error) {
	if detail != "quick" && detail != "demo" {
		return made, nil
	}
	prototype, err := cactusSpine(detail)
	if err != nil {
		// Its caller refuses the incomplete cactus.
		//ninja:vow feint maker needs a value
		return good{}, err
	}
	var members []shape.MeshMember
	for _, column := range columns {
		members, err = column.spines(members, prototype)
		if err != nil {
			// Its caller refuses the incomplete cactus.
			//ninja:vow feint maker needs a value
			return good{}, err
		}
	}
	group, err := shape.NewMeshGroup(members)
	if err != nil {
		// Its caller refuses the incomplete cactus.
		//ninja:vow feint maker needs a value
		return good{}, err
	}
	made.groups = map[slot]*shape.MeshGroup{leaf: group}
	return made, nil
}

// spines places short radial cones at the pleat crests of an upright column.
func (column cactusColumn) spines(members []shape.MeshMember,
	prototype *mesh.Mesh,
) ([]shape.MeshMember, error) {
	stem := column.stem
	room := int(cactusValue("maximum-spines")) - len(members)
	ratio := stem.run / cactusValue("spine-spacing")
	if !positive(ratio) || ratio >= float64(room/stem.ribs+spaceOne) {
		return nil, errTooMany
	}
	count := int(ratio)
	for row := range count {
		share := float64(row+spaceOne) / float64(count+spaceOne)
		centre := maths.V(stem.bend, stem.bend+stem.run*share, spaceZero)
		for rib := range stem.ribs {
			angle := fullTurn * float64(rib) / float64(stem.ribs)
			sine, cosine := math.Sincos(angle)
			out := maths.V(cosine, spaceZero, -sine)
			root := centre.Add(out.Scale(stem.radius * cactusValue("spine-root-share")))
			local := maths.NewSubspace(root, maths.Upright(out), spaceOne)
			members = append(members, shape.MeshMember{Mesh: prototype, Space: local.Within(column.pose)})
		}
	}
	return members, nil
}
