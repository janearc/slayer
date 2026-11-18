package searching

import (
	"github.com/janearc/slayer/scene/city"
)

// griffithHill and losAngelesBasin delegate to the shared scene constructor
// so an offshore set can use the exact same seeded city and ridge geometry.
func (s *heardStage) griffithHill() error {
	s.ground = city.GriffithGround
	return city.New(s.world, s.ground, true).GriffithHill()
}

// losAngelesBasin builds the Los Angeles basin on the shared city scene.
func (s *heardStage) losAngelesBasin() error {
	return city.New(s.world, s.ground, true).LosAngelesBasin()
}

// griffith adds the Griffith architecture to the stage.
func (s *heardStage) griffith() error { return s.griffithArchitecture() }
