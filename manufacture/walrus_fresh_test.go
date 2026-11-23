package manufacture

import (
	"github.com/janearc/slayer/scene/shape"
	"math"
	"testing"
)

func TestWalrusFresh1Constructs(t *testing.T) {
	for _, yaw := range []float64{0, .8} {
		p, e := WalrusFresh1(yaw, 0)
		if e != nil {
			t.Fatal(e)
		}
		if len(p) != 59 {
			t.Fatalf("got %d parts", len(p))
		}
	}
}

func TestWalrusFresh1ContinuityAndMotion(t *testing.T) {
	rest, e := WalrusFresh1(0, 0)
	if e != nil {
		t.Fatal(e)
	}
	turn, e := WalrusFresh1(.8, 0)
	if e != nil {
		t.Fatal(e)
	}
	swim, e := WalrusFresh1(.28, math.Pi/2)
	if e != nil {
		t.Fatal(e)
	}
	loop, e := WalrusFresh1(0, 2*math.Pi)
	if e != nil {
		t.Fatal(e)
	}
	body := rest[0].Shape.(*shape.Mesh)
	bent := turn[0].Shape.(*shape.Mesh)
	if !body.Closed() {
		t.Fatal("trunk skin is open")
	}
	for _, i := range []int{2, 28, 29, 31, 57, 58} {
		if !rest[i].Shape.(*shape.Mesh).Closed() {
			t.Fatalf("piece %d is open", i)
		}
	}
	volume := func(m *shape.Mesh) float64 {
		v := 0.
		for _, f := range m.Faces {
			a, b, c := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
			v += a.Dot(b.Cross(c)) / 6
		}
		return math.Abs(v)
	}
	v0, v1 := volume(body), volume(bent)
	if math.Abs(v0-v1) > 1e-8 {
		t.Fatalf("trunk volumes %f and %f differ", v0, v1)
	}
	displacement := bent.Vertices[0].Sub(body.Vertices[0]).Len()
	if displacement < .4 {
		t.Fatalf("turn displacement %f", displacement)
	}
	a := rest[29].Shape.(*shape.Mesh)
	b := swim[29].Shape.(*shape.Mesh)
	c := loop[29].Shape.(*shape.Mesh)
	change := a.Vertices[17].Sub(b.Vertices[17]).Len()
	endpoint := a.Vertices[17].Sub(c.Vertices[17]).Len()
	if change < .01 || endpoint > 1e-8 {
		t.Fatalf("hind stroke %f, loop error %g", change, endpoint)
	}
	t.Logf("trunk vertices=%d faces=%d volume=%0.6f m3 at rest/turn; tail displacement=%0.3f m; hind vertex stroke=%0.3f m; loop error=%g m", len(body.Vertices), len(body.Faces), v0, displacement, change, endpoint)
	t.Log("Limitation: closed components are disjoint; closed trunk alone is not a continuous whole-animal skin")
}
