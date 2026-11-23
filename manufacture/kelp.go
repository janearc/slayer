package manufacture

import (
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// Giant kelp: a holdfast of fingers on the floor, stipes rising from it,
// at each node a blade with a ruffled edge and a gas bladder at its
// base, and at the surface the stipes laid along it as a canopy. Every
// figure is data, kelp.yaml, with its basis.
//
// A plant's own space is metres, y up, its holdfast at the origin on the
// floor, and the sea's surface at the file's depth above it. The plant is
// hung by its bladders: its stipes lean downstream with a current, and a
// swell's orbit takes the water's motion down the stipe, less with depth.
//
// The swell and the current come from a named condition in the file, by
// the kind's name; the swell moves with time (MakeAtTime), travelling
// toward plus x, and the current runs toward plus x. A set turns the
// thing by its heading to point them where it wants.
//
// A forest is the same plants by seed on a jittered grid. Every blade,
// bladder and stipe is a placement of a shared mesh, so what a forest
// stores in faces does not grow with its plants.

// The kinds' names. A name followed by a hyphen and a condition's name
// is the kind in that water; the bare name is KelpWater.
const (
	KelpKind       = "giant-kelp"
	KelpForestKind = "kelp-forest"
	KelpWater      = "moderate"
)

// kelpFloat is the matter a gas bladder is of, after body and leaf.
const kelpFloat slot = 2 // defined: the third matter a plant has

// the numbers of a plant's making
const (
	kelpPlantRow    = 7        // defined: the row a forest draws a plant's seed from
	kelpNodeStride  = 4096     // defined: more nodes than any stipe has
	kelpGolden      = 2.399963 // defined: the golden angle, in radians
	kelpGround      = 0.03     // picked: how far above the floor a finger starts
	kelpRise        = 0.5      // picked: the steepest a finger climbs, in radians
	kelpFingerLeast = 0.6      // picked: the shortest a finger is, over the longest
	kelpJitter      = 0.35     // picked: how far a plant strays from its grid place, over the gap
	kelpSmall       = 1e-6     // picked: a length below which a direction is none
	kelpAlong       = 0.3      // picked: how much of a blade's axis runs along its stipe
)

// kelpRow is a kind of kelp: a plant or a forest of them, in a water.
type kelpRow struct {
	water  string
	forest bool
}

// kelpRows are the kinds, a plant and a forest in each water the file
// names. A file that cannot be read stops the build.
func kelpRows() map[string]maker {
	if kelpError != nil {
		panic(kelpError)
	}
	rows := map[string]maker{KelpKind: kelpRow{water: KelpWater},
		KelpForestKind: kelpRow{water: KelpWater, forest: true}}
	for name := range kelpRead.Conditions {
		rows[KelpKind+"-"+name] = kelpRow{water: name}
		rows[KelpForestKind+"-"+name] = kelpRow{water: name, forest: true}
	}
	return rows
}

// made is the kelp at time nought.
func (row kelpRow) made(from seed) (good, error) {
	return row.madeAt(from, 0)
}

// madeAtTime is the kelp as the swell has it at a time. The detail is
// not read: there is one representation.
func (row kelpRow) madeAtTime(from seed, _ string,
	at time.Duration) (good, error) {
	return row.madeAt(from, at.Seconds())
}

// madeAt is the plant or the forest at a time in seconds.
func (row kelpRow) madeAt(from seed, clock float64) (good, error) {
	kit, err := kelpShared()
	if err != nil {
		return good{}, err
	}
	places, plants := []maths.Vec{{}}, []seed{from}
	if row.forest {
		places = kelpLayout(from)
		plants = plants[:0]
		for index := range places {
			plants = append(plants, from.sprout(kelpPlantRow, index))
		}
	}
	var made kelpMembers
	for index, at := range places {
		plant := kelpPlant{kit: kit, from: plants[index], at: at,
			water: kelpRead.Conditions[row.water], clock: clock}
		plant.grow(&made)
	}
	return made.good()
}

// kelpMembers are the placements of a plant, or of a forest, by matter.
type kelpMembers struct {
	stipes, blades, bladders []shape.MeshMember
}

// good is the placements as a good: one shared group for each matter.
func (made kelpMembers) good() (good, error) {
	book := kelpRead
	look := func(name string) *kelpLook {
		row := book.Looks[name]
		return &row
	}
	dressed := looks{
		body: plain("kelp stipe", look("stipe").Colour[0],
			look("stipe").Colour[1], look("stipe").Colour[2]),
		leaf: plain("kelp blade", look("blade").Colour[0],
			look("blade").Colour[1], look("blade").Colour[2]),
		kelpFloat: plain("kelp bladder", look("float").Colour[0],
			look("float").Colour[1], look("float").Colour[2]),
	}
	grown := good{dressed: dressed, groups: map[slot]*shape.MeshGroup{},
		groupParts: map[slot]int{}}
	for of, members := range map[slot][]shape.MeshMember{body: made.stipes,
		leaf: made.blades, kelpFloat: made.bladders} {
		group, err := shape.NewMeshGroup(members)
		if err != nil {
			return good{}, err
		}
		grown.groups[of], grown.groupParts[of] = group, len(members)
	}
	return grown, nil
}

// kelpLayout is where a forest's plants stand: a grid as square as the
// count allows, each plant strayed from its place by the seed. Places
// are at the floor, y nought, about the origin.
func kelpLayout(from seed) []maths.Vec {
	count := kelpRead.count("plants", from, 0)
	side := int(math.Ceil(math.Sqrt(float64(count))))
	gap := kelpRead.pick("spacing", from, 0)
	places := make([]maths.Vec, count)
	for index := range places {
		east := float64(index%side) - float64(side-1)*kelpHalf
		north := float64(index/side) - float64(side-1)*kelpHalf
		places[index] = maths.V(
			(east+kelpJitter*from.swing(kelpPlantRow, index, 0))*gap, 0,
			(north+kelpJitter*from.swing(kelpPlantRow, index, 1))*gap)
	}
	return places
}
