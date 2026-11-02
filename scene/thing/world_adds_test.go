package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// a new world holds nothing: no things, volumes, fields or lamps, no
// sky, clear air and no pixel of its own.
func TestANewWorldIsEmpty(t *testing.T) {
	w := NewWorld()
	if w == nil {
		t.Fatal("NewWorld gave no world")
	}
	if len(w.Things) != 0 || len(w.Volumes) != 0 || len(w.Fields) != 0 || len(w.Lamps) != 0 {
		t.Errorf("a new world holds %d things, %d volumes, %d fields, %d lamps",
			len(w.Things), len(w.Volumes), len(w.Fields), len(w.Lamps))
	}
	if w.Sky != nil || w.Haze != (Haze{}) || w.Pixel != 0 {
		t.Error("a new world has a sky, haze or pixel")
	}
	if NewWorld() == w {
		t.Error("NewWorld gave the same world twice")
	}
}

// Add puts a copy of the thing in the world, in order, and hands back the
// one the world holds: changing it changes the world's, and not the
// thing it was made from.
func TestAddKeepsTheThingAndGivesItBack(t *testing.T) {
	w := NewWorld()
	chalk := &Material{Name: "chalk"}
	first := Thing{Material: chalk}
	a := w.Add(first)
	b := w.Add(Thing{Attrs: Attrs{Ghost: true}})
	if len(w.Things) != 2 || w.Things[0] != a || w.Things[1] != b {
		t.Fatalf("the world holds %d things, not a then b", len(w.Things))
	}
	if a == b {
		t.Fatal("two things are the one thing")
	}
	if a.Material != chalk || b.Material != nil || !b.Ghost() || a.Ghost() {
		t.Errorf("the things are not as added: %+v then %+v", *a, *b)
	}
	a.Attrs.Hidden = true
	if !w.Things[0].Hidden() {
		t.Error("changing the thing handed back did not change the world's")
	}
	if first.Hidden() {
		t.Error("changing the thing handed back changed the one it was made from")
	}
}

// AddVolume and AddField do as Add does, each into its own list.
func TestAddVolumeAndAddFieldKeepAndGiveBack(t *testing.T) {
	w := NewWorld()
	vol := Volume{Material: &Material{Name: "smoke"}}
	v := w.AddVolume(vol)
	v2 := w.AddVolume(Volume{Attrs: Attrs{Hidden: true}})
	fld := Field{Mass: 1, Reach: 5}
	f := w.AddField(fld)
	f2 := w.AddField(Field{Mass: 2})
	if len(w.Volumes) != 2 || w.Volumes[0] != v || w.Volumes[1] != v2 || v == v2 {
		t.Fatalf("the world holds %d volumes, not v then v2", len(w.Volumes))
	}
	if len(w.Fields) != 2 || w.Fields[0] != f || w.Fields[1] != f2 || f == f2 {
		t.Fatalf("the world holds %d fields, not f then f2", len(w.Fields))
	}
	if len(w.Things) != 0 || len(w.Lamps) != 0 {
		t.Error("a volume or a field went into another list")
	}
	if v.Material.Name != "smoke" || v.Hidden() || !v2.Hidden() {
		t.Errorf("the volumes are not as added: %+v then %+v", *v, *v2)
	}
	if f.Mass != 1 || f.Reach != 5 || f2.Mass != 2 {
		t.Errorf("the fields are not as added: %+v then %+v", *f, *f2)
	}
	v.Attrs.Hidden = true
	f.Mass = 9
	if !w.Volumes[0].Hidden() || w.Fields[0].Mass != 9 {
		t.Error("changing what was handed back did not change the world's")
	}
	if vol.Hidden() || fld.Mass != 1 {
		t.Error("changing what was handed back changed what it was made from")
	}
}

// Lamp adds a lamp, after the ones there.
func TestLampAddsALamp(t *testing.T) {
	w := NewWorld()
	one := Sun(maths.Up, maths.Grey(1))
	two := Sun(maths.XAxis, maths.Grey(2))
	w.Lamp(one)
	if len(w.Lamps) != 1 || w.Lamps[0] != one {
		t.Fatalf("the world holds %d lamps, not the one", len(w.Lamps))
	}
	w.Lamp(two)
	if len(w.Lamps) != 2 || w.Lamps[0] != one || w.Lamps[1] != two {
		t.Fatalf("the world holds %d lamps, not one then two", len(w.Lamps))
	}
}

// the sun lamp is film's sun, toward and as strong as given, and as wide
// as the sun looks from the earth: twice the arctangent of its radius
// over its distance. the sun's nominal radius is 695700 km (iaus, 2015)
// and an astronomical unit is 149597870.7 km, which is 0.5329 degrees.
func TestSunIsAsWideAsTheSunLooks(t *testing.T) {
	const radius, au = 695700.0, 149597870.7
	want := 2 * math.Atan(radius/au)
	toward, strength := maths.V(0, 1, 2), maths.RGB(3, 4, 5)
	sun, ok := Sun(toward, strength).(film.Sun)
	if !ok {
		t.Fatalf("Sun gave %T, not film's sun", Sun(toward, strength))
	}
	if sun.Toward != toward || sun.Strength != strength {
		t.Errorf("the sun is toward %v at %v", sun.Toward, sun.Strength)
	}
	// the code says 0.53 degrees, which is within a hundredth of a
	// degree of it; the changes that matter are by whole factors
	if d := math.Abs(sun.Size-want) / maths.Degree; d > 0.01 {
		t.Errorf("the sun is %v radians wide, want %v: %v degrees out", sun.Size, want, d)
	}
}
