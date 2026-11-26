package manufacture

import (
	"sync"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture/plants/cacti"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// columnarCactus is authored desert scenery, not an age or water-storage model.
type columnarCactus struct {
	name        string
	recipe      *cacti.Recipe
	spinesCache *sync.Map
}

func (plant columnarCactus) value(name string) float64 { return plant.recipe.Value(name) }

type cactusColumn = cacti.Column

// cactusValue keeps all picked cactus controls in the shared Pzm number book.
func cactusValue(name string) float64 { return plantValue("cactus-" + name) }

// made supplies the ordinary quick specimen.
func (plant columnarCactus) made(from seed) (good, error) { return plant.madeAt(from, "quick") }

// madeAt keeps arms, caps and reserved spine extent stable across details.
func (plant columnarCactus) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, errGood
	}
	if err := plant.recipe.Validate(); err != nil {
		//ninja:vow feint maker needs a value; Things refuses this one
		return good{}, err
	}
	columns := plant.recipe.Columns(from)
	made := plant.body(columns, detail)
	return plant.spines(made, columns, detail)
}

// body builds the ordinary skin and reserves the finest physical envelope.
func (plant columnarCactus) body(columns []cactusColumn, detail string) good {
	made := good{dressed: looks{
		body: plain("cactus skin", plant.value("green-r"), plant.value("green-g"), plant.value("green-b")),
		leaf: plain("cactus spines", plant.value("spine-r"), plant.value("spine-g"), plant.value("spine-b")),
	}}
	first := columns[spaceZero]
	extent := transformed(first.Stem.Bounds(plant.value("spine-length")), first.Pose)
	for _, column := range columns {
		stem, sides := plant.recipe.Surface(column, detail)
		made.parts = append(made.parts, part{shape: cactusStem(stem), fine: cut{around: sides}, space: column.Pose, of: body})
		box := transformed(column.Stem.Bounds(plant.value("spine-length")), column.Pose)
		extent = enclosing(extent, box)
	}
	made.extent = &extent
	return made
}

// cactusStem carries a columnar cactus part through manufacture's mesh API.
type cactusStem cacti.Stem

func (stem cactusStem) skin(fine cut) surface {
	return cacti.Skin[face](cacti.Stem(stem), fine.around)
}

// cactusSpine shares one physical cone without scaling it to plant size.
func (plant columnarCactus) spine(detail string) (*mesh.Mesh, error) {
	key := struct{ name, detail string }{plant.name, detail}
	if known, found := plant.spinesCache.Load(key); found {
		return known.(*mesh.Mesh), nil
	}
	length := measure.Metres(plant.value("spine-length"))
	radius := measure.Metres(plant.value("spine-radius"))
	cone, err := newTube(length, radius, measure.Metres(spaceZero))
	if err != nil {
		return nil, err
	}
	fine := cut{around: int(plant.value("spine-sides-" + detail))}
	made, err := cone.skin(fine).meshed()
	if err != nil {
		return nil, err
	}
	actual, _ := plant.spinesCache.LoadOrStore(key, made)
	return actual.(*mesh.Mesh), nil
}

// spines adds bounded instances to the upright runs, leaving bends and caps
// bare.
func (plant columnarCactus) spines(made good, columns []cactusColumn, detail string) (good, error) {
	if detail != "quick" && detail != "demo" {
		return made, nil
	}
	prototype, err := plant.spine(detail)
	if err != nil {
		// Its caller refuses the incomplete cactus.
		//ninja:vow feint maker needs a value
		return good{}, err
	}
	var members []shape.MeshMember
	for _, column := range columns {
		members, err = column.Stem.Spines(members, prototype, column.Pose,
			plant.value("spine-spacing"), plant.value("spine-root-share"), int(plant.value("maximum-spines")))
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
