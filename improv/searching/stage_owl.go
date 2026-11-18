package searching

import (
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// makeOWL places the open 100-metre concept beside its parked roll-away
// shelter. The yard, access ways and gantries are authored film scenery.
func (s *heardStage) makeOWL() error {
	if err := s.owlHousing(); err != nil {
		return err
	}
	for _, row := range []struct {
		name string
		at   maths.Vec
	}{
		{"owl", maths.Vec{}},
		{"mirror-gantry", maths.V(-90, 0, 25)}, {"mobile-crane", maths.V(80, 0, -10)},
		{"work-platform", maths.V(-70, 0, -25)}, {"access-stairs", maths.V(65, 0, -25)},
	} {
		if err := s.factory(row.name, row.at, 1, 0); err != nil {
			return err
		}
	}
	for _, x := range []float64{-115, 115} {
		for i := 0; i < 6; i++ {
			at := maths.V(x, 0, -40+float64(i)*30)
			if err := s.factory("equipment-cabinet", at, 1, 0); err != nil {
				return err
			}
			if _, err := s.characterRod(at.Add(maths.V(0, .4, -12)), at.Add(maths.V(0, .4, 12)), .18, "OWL service-way lighting", maths.RGB(.4, .16, .04)); err != nil {
				return err
			}
		}
	}
	if err := s.owlCampus(); err != nil {
		return err
	}
	// A cold low key and practical yard lamps distinguish white salts,
	// dark outcrops and the enclosure's service structure after sunset.
	s.world.Lamp(thing.Sun(maths.V(-.7, .4, -.6), maths.RGB(.3, .38, .5)))
	for _, x := range []float64{-140, 140} {
		s.world.Lamp(emit.Lit{At: maths.V(x, 20, 120), Strength: maths.RGB(800, 1100, 1600)})
	}
	return nil
}
