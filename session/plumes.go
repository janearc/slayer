package session

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// plumes lays the set's plumes in the world, each mouth's plume carried
// down the set's wind, after the things they leave from are placed.
func (stage *staging) plumes(read set.Set) error {
	from, speed := read.Wind()
	downwind := skyward(from, windHorizon).Neg()
	turn := maths.Rotation(maths.YAxis, math.Atan2(-downwind.Z, downwind.X))
	_, grounded := read.Ground()
	for _, plume := range read.Plumes() {
		if err := stage.plume(plume, turn, speed, grounded); err != nil {
			return fmt.Errorf("%s: plume %s: %w", stage.given.set,
				plume.Name(), err)
		}
	}
	return nil
}

// plume lays one plume's mouths: one at its origin, or many laid by its
// seed within its spread round the origin, each standing the offset's
// height over the ground under it where the set has ground.
func (stage *staging) plume(plume set.Plume, turn maths.Orientation,
	wind float64, grounded bool) error {
	object, east, up, south := plume.Origin()
	origin := maths.V(east, up, south)
	if object != "" {
		at, found := stage.where(object)
		if !found {
			return fmt.Errorf("object %q has no placed origin", object)
		}
		origin = origin.Add(at)
	}
	count, spread, seed := plume.Mouths()
	if count == 1 {
		return stage.mouth(plume, origin, turn, wind)
	}
	places := rand.New(rand.NewPCG(seed, uint64(count)))
	floor := terrainFloor{window: stage.ground, lowest: stage.lowest}
	for range count {
		at := strewn(places, origin, spread)
		if grounded {
			at.Y = floor.height(at.X, at.Z) + up
		}
		if err := stage.mouth(plume, at, turn, wind); err != nil {
			return err
		}
	}
	return nil
}

// strewn is a point drawn evenly over the disc of a radius round a middle.
func strewn(places *rand.Rand, middle maths.Vec, radius float64) maths.Vec {
	distance := radius * math.Sqrt(places.Float64())
	angle := fullCircle * places.Float64()
	return middle.Add(maths.V(distance*math.Cos(angle), 0,
		distance*math.Sin(angle)))
}

// fullCircle is the angle of a whole turn, in radians. Defined.
const fullCircle = 2 * math.Pi

// mouth lays the plume from one mouth at a point, turned down the wind:
// the manufactory's plume of what the set says, staged into the world.
func (stage *staging) mouth(plume set.Plume, at maths.Vec,
	turn maths.Orientation, wind float64) error {
	return plumeOf(plume, at, turn, wind).Staged(stage.world)
}

// plumeOf is the manufactory's plume for one mouth of a set's plume
// at a point: what leaves the mouth, the particles' optics, the reach and
// the lid the set says.
func plumeOf(plume set.Plume, at maths.Vec, turn maths.Orientation,
	wind float64) manufacture.Plume {
	mouth, exit, hot, air := plume.Stack()
	rate, extinction, albedo, asymmetry, exponent := plume.Particles()
	lid, lidded := plume.Lid()
	return manufacture.Plume{Name: plume.Name(), At: at, Turn: turn,
		Wind: wind, Reach: plume.Reach(), Mouth: mouth, Exit: exit,
		Hot: hot, Air: air, Lid: lid, Lidded: lidded, Rate: rate,
		Extinction: extinction, Asymmetry: asymmetry, Exponent: exponent,
		Albedo: albedo}
}
