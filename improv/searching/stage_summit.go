package searching

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// summitScenery leaves the high volcanic site bare of trees. Rock positions,
// colour and cloud weather are authored; clouds are not measured weather.
// Separate large cloud banks remain below the summit and follow the
// spherical horizon. There is no opaque flat deck spanning the scene.
func (s *heardStage) summitScenery(p *StagePlan) error {
	if !p.Interior {
		temp := &Set{World: s.world, prototypes: map[string][]thing.Thing{}}
		for i := range 128 {
			angle := float64(i) * 2.399963229728653
			radius := 85 + 19*math.Sqrt(float64(i+1))
			x, z := radius*math.Cos(angle), radius*math.Sin(angle)
			if math.Abs(z-33) < 12 {
				continue
			}
			if err := temp.scenery("boulder", uint64(900+i), maths.V(x, s.ground(x, z), z), .3+1.1*math.Abs(math.Sin(float64(i)*1.73))); err != nil {
				return err
			}
		}
	}
	if p.Starlight {
		return nil
	}

	// A connected, undulating marine cloud field follows the spherical
	// horizon far below the instruments. Noise shapes both its cloud tops
	// and clearings, rather than placing disconnected illuminated balls.
	weather := medium.NewWeather(.004, 850, .93, 1880, maths.V(4, 0, -2))
	weather.PatchScale = 2200
	weather.Asymmetry = .35
	base := func(x, z float64) float64 {
		return -3400 - (x*x+z*z)/(2*keckEarthRadius) + 260*noise.Fractal(1881, x/1800, 0, z/1800, 4)
	}
	clouds, err := medium.NewLayer(weather, maths.Box{Min: maths.V(-50000, -4300, -50000), Max: maths.V(50000, -2200, 50000)}, base)
	if err != nil {
		return err
	}
	s.world.AddVolume(thing.Volume{Medium: clouds, Material: &thing.Material{Name: "authored connected marine cloud field below Maunakea", Colour: maths.RGB(.86, .91, 1)}})
	return s.summitDetritus()
}
