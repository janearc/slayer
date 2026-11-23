package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// worldAddsSun is the csv file of the sun the tests ask Sun for: the way
// it lies and how strong it is, in the three parts of each.
const worldAddsSun = "testdata/world_adds-sun.csv"

// The places of the first and second of the things a world holds.
const (
	worldAddsFirst  = 0 // defined: the first place in a list
	worldAddsSecond = 1 // defined: the second place in a list
)

// worldAddsInOrder is whether a list holds two things and no more, the
// first then the second.
func worldAddsInOrder[Held any](held []*Held, first, second *Held) bool {
	return len(held) == pair &&
		held[worldAddsFirst] == first && held[worldAddsSecond] == second
}

// worldAddsEmpty is whether a world holds no thing, volume, field or lamp.
func worldAddsEmpty(world *World) bool {
	return len(world.Things) == nought && len(world.Volumes) == nought &&
		len(world.Fields) == nought && len(world.Lamps) == nought
}

// a new world holds nothing: no things, volumes, fields or lamps, no
// sky, clear air and no pixel of its own.
func TestANewWorldIsEmpty(t *testing.T) {
	world := NewWorld()
	if world == nil {
		t.Fatal("NewWorld gave no world")
	}
	if !worldAddsEmpty(world) {
		t.Errorf("a new world holds %d things, %d volumes, %d fields, %d lamps",
			len(world.Things), len(world.Volumes), len(world.Fields),
			len(world.Lamps))
	}
	if world.Sky != nil || world.Haze != (Haze{}) || world.Pixel != nought {
		t.Error("a new world has a sky, haze or pixel")
	}
	if NewWorld() == world {
		t.Error("NewWorld gave the same world twice")
	}
}

// Add puts a copy of the thing in the world, in order, and hands back the
// one the world holds: changing it changes the world's, and not the
// thing it was made from.
func TestAddKeepsTheThingAndGivesItBack(t *testing.T) {
	world := NewWorld()
	chalk := &Material{Name: "chalk"}
	original := Thing{Material: chalk}
	added := world.Add(original)
	ghost := world.Add(Thing{Attrs: Attrs{Ghost: true}})
	if !worldAddsInOrder(world.Things, added, ghost) {
		t.Fatalf("the world holds %d things, not a then b", len(world.Things))
	}
	if added == ghost {
		t.Fatal("two things are the one thing")
	}
	if added.Material != chalk || ghost.Material != nil || !ghost.Attrs.Ghost ||
		added.Attrs.Ghost {
		t.Errorf("the things are not as added: %+v then %+v", *added, *ghost)
	}
	added.Attrs.Hidden = true
	if !world.Things[worldAddsFirst].Hidden() {
		t.Error("changing the thing handed back did not change the world's")
	}
	if original.Hidden() {
		t.Error("changing the thing handed back changed what it was made from")
	}
}

// AddVolume and AddField do as Add does, each into its own list.
func TestAddVolumeAndAddFieldKeepAndGiveBack(t *testing.T) {
	world := NewWorld()
	original := Volume{Material: &Material{Name: "smoke"}}
	volume := world.AddVolume(original)
	hidden := world.AddVolume(Volume{Attrs: Attrs{Hidden: true}})
	if !worldAddsInOrder(world.Volumes, volume, hidden) || volume == hidden {
		t.Fatalf("the world holds %d volumes, not v then v2",
			len(world.Volumes))
	}
	if volume.Material.Name != "smoke" || volume.Hidden() || !hidden.Hidden() {
		t.Errorf("the volumes are not as added: %+v then %+v", *volume, *hidden)
	}
	volume.Attrs.Hidden = true
	if !world.Volumes[worldAddsFirst].Hidden() {
		t.Error("changing what was handed back did not change the world's")
	}
	if original.Hidden() {
		t.Error("changing what was handed back changed what it was made from")
	}
	worldAddsKeepFields(t, world)
	if len(world.Things) != nought || len(world.Lamps) != nought {
		t.Error("a volume or a field went into another list")
	}
}

// worldAddsKeepFields adds two fields to a world and checks that it keeps
// them, in order, as it was given them, and hands back the ones it holds.
func worldAddsKeepFields(t *testing.T, world *World) {
	t.Helper()
	mass := picked(t, "world_adds-field-mass")
	original := Field{Mass: mass, Reach: picked(t, "world_adds-field-reach")}
	other := picked(t, "world_adds-second-field-mass")
	field := world.AddField(original)
	second := world.AddField(Field{Mass: other})
	if !worldAddsInOrder(world.Fields, field, second) || field == second {
		t.Fatalf("the world holds %d fields, not f then f2", len(world.Fields))
	}
	if field.Mass != mass || field.Reach != original.Reach ||
		second.Mass != other {
		t.Errorf("the fields are not as added: %+v then %+v", *field, *second)
	}
	field.Mass = picked(t, "world_adds-field-mass-changed")
	if world.Fields[worldAddsFirst].Mass != field.Mass {
		t.Error("changing what was handed back did not change the world's")
	}
	if original.Mass != mass {
		t.Error("changing what was handed back changed what it was made from")
	}
}

// Lamp adds a lamp, after the ones there.
func TestLampAddsALamp(t *testing.T) {
	world := NewWorld()
	strong := picked(t, "world_adds-lamp-strength-one")
	stronger := picked(t, "world_adds-lamp-strength-two")
	first := Sun(maths.Up, maths.Grey(strong))
	second := Sun(maths.XAxis, maths.Grey(stronger))
	world.Lamp(first)
	if len(world.Lamps) != unity || world.Lamps[worldAddsFirst] != first {
		t.Fatalf("the world holds %d lamps, not the one", len(world.Lamps))
	}
	world.Lamp(second)
	if len(world.Lamps) != pair || world.Lamps[worldAddsFirst] != first ||
		world.Lamps[worldAddsSecond] != second {
		t.Fatalf("the world holds %d lamps, not one then two",
			len(world.Lamps))
	}
}

// the sun lamp is film's sun, toward and as strong as given, and as wide
// as the sun looks from the earth: twice the arctangent of its radius
// over its distance. the sun's nominal radius is 695700 km (iaus, 2015)
// and an astronomical unit is 149597870.7 km, which is 0.5329 degrees.
func TestSunIsAsWideAsTheSunLooks(t *testing.T) {
	radius := picked(t, "world_adds-sun-radius-km")
	distance := picked(t, "world_adds-astronomical-unit-km")
	want := pair * math.Atan(radius/distance)
	sunRows := readSheet(t, worldAddsSun)
	way := sunRows.row(t, worldAddsSun, "toward")
	light := sunRows.row(t, worldAddsSun, "strength")
	toward := maths.V(cell(t, way, "x"), cell(t, way, "y"), cell(t, way, "z"))
	strength := maths.RGB(
		cell(t, light, "x"), cell(t, light, "y"), cell(t, light, "z"))
	sun, ok := Sun(toward, strength).(film.Sun)
	if !ok {
		t.Fatalf("Sun gave %T, not film's sun", Sun(toward, strength))
	}
	if sun.Toward != toward || sun.Strength != strength {
		t.Errorf("the sun is toward %v at %v", sun.Toward, sun.Strength)
	}
	// the code says 0.53 degrees, which is within a hundredth of a
	// degree of it; the changes that matter are by whole factors
	tolerance := picked(t, "world_adds-sun-width-tolerance-degrees")
	if out := math.Abs(sun.Size-want) / maths.Degree; out > tolerance {
		t.Errorf("the sun is %v radians wide, want %v: %v degrees out",
			sun.Size, want, out)
	}
}
