// Package kelp builds a checked kelp specimen or forest in moving water.
package kelp

import (
	"math"

	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

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

// Members holds the three shared-geometry groups and their profile colours.
type Members struct {
	Stipes, Blades, Bladders              []shape.MeshMember
	StipeColour, BladeColour, FloatColour [3]float64
}

type kelpMembers struct{ stipes, blades, bladders []shape.MeshMember }

// Grow builds placements for the specimen at the given time in seconds.
func (specimen Specimen) Grow(from branching.Seed, clock float64) (Members, error) {
	profile := specimen.profile
	book := &profile.book
	kit, err := profile.kit()
	if err != nil {
		return Members{}, err
	}
	places, plants := []maths.Vec{{}}, []branching.Seed{from}
	if specimen.forest {
		places = kelpLayout(book, from)
		plants = plants[:0]
		for index := range places {
			plants = append(plants, from.Sprout(kelpPlantRow, index))
		}
	}
	var made kelpMembers
	for index, at := range places {
		plant := kelpPlant{book: book, kit: kit, from: plants[index], at: at, water: book.Conditions[specimen.water], clock: clock}
		plant.grow(&made)
	}
	result := Members{Stipes: made.stipes, Blades: made.blades, Bladders: made.bladders}
	copy(result.StipeColour[:], book.Looks["stipe"].Colour)
	copy(result.BladeColour[:], book.Looks["blade"].Colour)
	copy(result.FloatColour[:], book.Looks["float"].Colour)
	return result, nil
}

// kelpLayout is where a forest's plants stand: a grid as square as the
// count allows, each plant strayed from its place by the seed. Places
// are at the floor, y nought, about the origin.
func kelpLayout(book *kelpBook, from branching.Seed) []maths.Vec {
	count := book.count("plants", from, 0)
	side := int(math.Ceil(math.Sqrt(float64(count))))
	gap := book.pick("spacing", from, 0)
	places := make([]maths.Vec, count)
	for index := range places {
		east := float64(index%side) - float64(side-1)*kelpHalf
		north := float64(index/side) - float64(side-1)*kelpHalf
		places[index] = maths.V(
			(east+kelpJitter*from.Swing(kelpPlantRow, index, 0))*gap, 0,
			(north+kelpJitter*from.Swing(kelpPlantRow, index, 1))*gap)
	}
	return places
}
