package searching

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"reflect"
	"testing"
)

func TestIndividualsHaveDifferentAnatomyNotOnlyPaint(t *testing.T) {
	seen := map[[32]byte]bool{}
	purposes := map[int]bool{}
	one, fifty, thin, broad, forward, reverse := false, false, false, false, false, false
	for id := 0; id < 3072; id++ {
		g := identityOf(id)
		purposes[g.Purpose] = true
		one = one || g.Rings == 1
		fifty = fifty || g.Rings == 50
		thin = thin || g.Width < .01
		broad = broad || g.Width > .8
		forward = forward || g.Speed > 1
		reverse = reverse || g.Speed < -1
		// Exclude color, emission and seed from this witness. There must still
		// be no duplicate anatomical blueprint.
		raw, _ := json.Marshal([]float64{float64(g.Purpose), float64(g.Rings), float64(g.Limbs), g.Radius, g.Width, g.Eccentricity})
		key := sha256.Sum256(raw)
		if seen[key] {
			t.Fatal("duplicate anatomical blueprint", id)
		}
		seen[key] = true
	}
	if len(purposes) != 8 || !one || !fifty || !thin || !broad || !forward || !reverse {
		t.Fatal("missing requested extremes")
	}
	// Check actual generated surfaces as well as parameters.
	geometry := map[[32]byte]bool{}
	for id := 0; id < 32; id++ {
		s := &heardStage{world: thing.NewWorld()}
		b := crowdBeing{identity: identityOf(id)}
		if err := s.buildIndividual(&b, false); err != nil {
			t.Fatal(err)
		}
		vertices := [][]maths.Vec{}
		for _, p := range b.parts {
			if m, ok := p.part.Shape.(*mesh.Mesh); ok {
				vertices = append(vertices, m.Points())
			}
		}
		raw, _ := json.Marshal(vertices)
		key := sha256.Sum256(raw)
		if len(vertices) == 0 || geometry[key] {
			t.Fatal("reused complete surface assembly", id)
		}
		geometry[key] = true
	}
}

func TestIndividualsSeekWithoutChangingPersonality(t *testing.T) {
	s := &heardStage{world: thing.NewWorld(), crowd: &multitude{}}
	for id := 0; id < 8; id++ {
		b := crowdBeing{identity: identityOf(id), scale: 1, members: 3, member: id % 3, system: id}
		if err := s.buildIndividual(&b, false); err != nil {
			t.Fatal(err)
		}
		s.crowd.beings = append(s.crowd.beings, b)
	}
	s.updateMultitude(290)
	poses := []maths.Subspace{}
	glows := []maths.Light{}
	for _, b := range s.crowd.beings {
		for _, p := range b.parts {
			poses = append(poses, p.part.At(0))
			glows = append(glows, p.part.Material.Glow)
		}
	}
	for _, at := range []float64{330, 278, 300, 290} {
		s.updateMultitude(at)
	}
	i := 0
	for _, b := range s.crowd.beings {
		for _, p := range b.parts {
			if !reflect.DeepEqual(poses[i], p.part.At(0)) || glows[i] != p.part.Material.Glow {
				t.Fatal("frame history changes individual", i)
			}
			i++
		}
	}
}

func TestZappyUsesRadiationVolumesRatherThanSolidShafts(t *testing.T) {
	s := &heardStage{world: thing.NewWorld()}
	if err := s.nativeCharacter(0, maths.V(2, 3, 4), 2); err != nil {
		t.Fatal(err)
	}
	if len(s.world.Volumes) != 2 {
		t.Fatal("missing expanding jets")
	}
	for _, p := range s.world.Things {
		if p.Material.Name == "zappy polar shaft" {
			t.Fatal("opaque shaft remains")
		}
	}
	s.updateSpace(&stagePlan{Kind: "zappy"}, State{At: 94, Beat: 12}, .5)
	for _, v := range s.world.Volumes {
		if v.At(0).Origin != maths.V(2, 3, 4) || v.At(0).Scale != 2 {
			t.Fatal("jet detached from character")
		}
		near := v.Medium.Density(maths.V(.2, 1, 0))
		far := v.Medium.Density(maths.V(.2, 3, 0))
		if near <= 0 || far <= 0 || v.Medium.Density(maths.V(0, 4, 0)) != 0 {
			t.Fatal("jet lacks expanding, bounded gas")
		}
	}
}
