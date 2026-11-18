package searching

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// summitDetritus scatters authored volcanic debris across the summit surface
// below the DEM's one-arcsecond scale.
// The fragments are authored volcanic detritus, not surveyed boulder sites.
// Their coordinates are disjoint, so each material can retain one complete
// mesh without welding touching construction components.
func (s *heardStage) summitDetritus() error {
	var fragments [4]siteBlocks
	clear := func(x, z float64) bool {
		a, b := maths.V(-28.669, 0, 18.299), maths.V(22.005, 0, -49.972)
		p := maths.V(x, 0, z)
		edge := b.Sub(a)
		u := math.Max(0, math.Min(1, p.Sub(a).Dot(edge)/edge.Dot(edge)))
		if p.Sub(a.Add(edge.Scale(u))).Len() < 48 {
			return false
		}
		for _, site := range maunaKeaNeighbours {
			if math.Hypot(x-site.x, z-site.z) < site.radius+13 {
				return false
			}
		}
		if s.summitGround != nil && s.summitGround.nearRoad(p, 6) {
			return false
		}
		return true
	}
	for i := range 18000 {
		id := int64(i)
		a := float64(i) * 2.399963229728653
		radius := 620 * math.Sqrt(noise.Hash(2871, id))
		x, z := radius*math.Cos(a), radius*math.Sin(a)
		if !clear(x, z) {
			continue
		}
		size := .07 + 1.25*math.Pow(noise.Hash(2872, id), 5)
		height := size * (.3 + .65*noise.Hash(2873, id))
		y := s.ground(x, z) - height*.15
		batch := &fragments[i%4]
		first := len(batch.vertices)
		turn := noise.Hash(2874, id) * 2 * math.Pi
		for j := range 5 {
			angle := turn + float64(j)*2*math.Pi/5
			r := size * (.6 + .5*noise.Hash3(2875, id, int64(j), 0))
			batch.vertices = append(batch.vertices, maths.V(x+r*math.Cos(angle), y, z+r*math.Sin(angle)))
		}
		top, bottom := len(batch.vertices), len(batch.vertices)+1
		batch.vertices = append(batch.vertices, maths.V(x+size*.17, y+height, z-size*.1), maths.V(x, y-height*.3, z))
		for j := range 5 {
			next := (j + 1) % 5
			batch.faces = append(batch.faces, [3]int{first + j, top, first + next}, [3]int{first + next, bottom, first + j})
		}
	}
	for i, colour := range []maths.Light{maths.RGB(.045, .05, .06), maths.RGB(.12, .1, .09), maths.RGB(.25, .14, .1), maths.RGB(.38, .36, .34)} {
		if err := fragments[i].place(s, &thing.Material{Name: "authored Maunakea volcanic debris and fractured lava", Colour: colour}); err != nil {
			return err
		}
	}
	return nil
}
