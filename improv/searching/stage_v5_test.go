package searching

import (
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"testing"
)

// Rapid succession must still give every identity actual frames at proof fps.
func TestHeardMultitudeEveryBeingGetsFrames(t *testing.T) {
	counts := make([]int, 96)
	for frame := int(math.Ceil(76 * 240.0 / 66 * 6)); float64(frame)/6 < 86*240.0/66; frame++ {
		counts[crowdIndex(float64(frame)/6)]++
	}
	for i, n := range counts {
		if n < 2 {
			t.Fatal("unreadable or absent being", i, n)
		}
	}
}

// The returning light changes both the sky and illumination from several
// directions. A lone luminous disc or exposure change cannot satisfy this.
func TestHeardEarthResponseLightsTheScene(t *testing.T) {
	s := &heardStage{world: thing.NewWorld(), burstLamps: []int{1, 2, 3}}
	for i := 0; i < 4; i++ {
		s.world.Lamp(film.Sun{})
	}
	p := &StagePlan{Burst: true, Sun: [2]float64{-15, -15}}
	if err := s.updateGround(p, State{At: 243.636363636}, 0); err != nil {
		t.Fatal(err)
	}
	before := s.world.Lamps[0].(film.Sun).Strength.Luminance()
	if err := s.updateGround(p, State{At: 246}, .5); err != nil {
		t.Fatal(err)
	}
	sum := 0.
	for _, lamp := range s.world.Lamps {
		level := lamp.(film.Sun).Strength.Luminance()
		if level < .1 {
			t.Fatal("missing collective light")
		}
		sum += level
	}
	if sum < before*10 {
		t.Fatal("ground did not brighten")
	}
	old := heardSky{}
	newSky := heardSky{burst: 1}
	a, b := 0., 0.
	for i := 0; i < 2000; i++ {
		az := float64(i) * 2.3999632297
		y := (float64(i) + .5) / 2000
		r := math.Sqrt(1 - y*y)
		d := maths.V(r*math.Cos(az), y, r*math.Sin(az))
		a += old.Light(d).Luminance()
		b += newSky.Light(d).Luminance()
	}
	if b < a*10 {
		t.Fatal("stellar sky did not brighten broadly", a, b)
	}
}
