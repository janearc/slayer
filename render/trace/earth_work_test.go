package trace

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// work is what a picture asked of the world, counted where the time goes:
// ray hits on shapes, densities asked of media, and paint asked of
// textures. the counts are the same on a busy machine and an idle one.
type work struct {
	hits, densities, paints atomic.Int64
}

// countedSphere is a sphere that counts the rays it is asked about.
type countedSphere struct {
	shape.Sphere
	n *atomic.Int64
}

func (c countedSphere) Hit(r maths.Ray, near, far float64) (float64, bool) {
	c.n.Add(1)
	return c.Sphere.Hit(r, near, far)
}

// countedMedium is a medium that counts the densities it is asked for. it
// says where a ray crosses it as the medium it wraps does.
type countedMedium struct {
	medium.Medium
	span medium.Spanner
	n    *atomic.Int64
}

func (c countedMedium) Density(p maths.Vec) float64 {
	c.n.Add(1)
	return c.Medium.Density(p)
}

func (c countedMedium) Span(r maths.Ray, near, far float64) (float64, float64,
	bool) {
	return c.span.Span(r, near, far)
}

// countedTexture is a texture that counts the paint it is asked for, by a
// footprint or not, and paints as the texture it wraps does.
type countedTexture struct {
	texture.Texture
	n *atomic.Int64
}

func (c countedTexture) At(p maths.Vec) maths.Light {
	c.n.Add(1)
	return c.Texture.At(p)
}

func (c countedTexture) Within(p maths.Vec, footprint float64) maths.Light {
	c.n.Add(1)
	return texture.Sample(c.Texture, p, footprint)
}

// counting wraps the world's shapes, media and textures so that they
// count, and so that they draw as they did. it refuses a part it cannot
// wrap without changing what it draws, so the count is of the picture.
func counting(t *testing.T, w *thing.World, n *work) {
	t.Helper()
	wrap := func(tx texture.Texture) texture.Texture {
		if tx == nil {
			return nil
		}
		if _, moves := tx.(texture.Timed); moves {
			t.Fatalf("a texture that moves, %T, cannot be counted as it is", tx)
		}
		return countedTexture{tx, &n.paints}
	}
	for _, th := range w.Things {
		sphere, ok := th.Shape.(shape.Sphere)
		if !ok {
			t.Fatalf("a %T, not a sphere, cannot be counted", th.Shape)
		}
		th.Shape = countedSphere{sphere, &n.hits}
		if m := th.Material; m != nil {
			m.Texture, m.Shine, m.Rough = wrap(m.Texture), wrap(m.Shine), wrap(m.Rough)
		}
	}
	for _, v := range w.Volumes {
		spanner, ok := v.Medium.(medium.Spanner)
		_, moves := v.Medium.(medium.Timed)
		_, optical := v.Medium.(gas.Optical)
		if !ok || moves || optical {
			t.Fatalf("a %T cannot be counted as it is", v.Medium)
		}
		v.Medium = countedMedium{v.Medium, spanner, &n.densities}
		if m := v.Material; m != nil {
			m.Texture, m.Shine, m.Rough = wrap(m.Texture), wrap(m.Shine), wrap(m.Rough)
		}
	}
}

// earthAt is the earth drawn small at a time of its day, counted: the
// frame, and the work it took.
func earthAt(t *testing.T, at time.Duration, count bool) ([]float32, *work) {
	t.Helper()
	w := thing.NewWorld()
	w.Lamps = nil
	w.Lamp(thing.Sun(maths.V(-1, 0.2, 0.4), maths.RGB(1.2, 1.2, 1.2)))
	e := template.Earth()
	e.Moons = nil
	e.Place(w, maths.Vec{}, 1/e.Radius, 8)
	n := &work{}
	if count {
		counting(t, w, n)
	}
	from := maths.V(0.1, 0.5, 3.1)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
	f := Render(w, eye, film.Pinhole{Wide: 40 * maths.Degree}, at, 32, 18)
	var flat []float32
	for _, p := range f.Pix {
		flat = append(flat, p.R, p.G, p.B, p.Cover, p.Depth)
	}
	return flat, n
}

// the earth costs about the same to draw at any time of its day: the
// calls to shapes, media and textures, early and late, are within a
// factor of two, in each of the three kinds. it counts work done and not
// time taken, so a busy machine cannot fail it.
//
// what it does not see is time spent outside those three calls. nothing
// of the earth moves by the time of day but its turn, so the same kinds
// of call are made on a different face of it.
//
// it also holds that counting changes nothing drawn: the counted earth
// is the earth, bit for bit.
func TestTheEarthAsksTheSameWorkAllDay(t *testing.T) {
	late := 4*time.Hour + 30*time.Minute
	bare0, _ := earthAt(t, 0, false)
	bare1, _ := earthAt(t, late, false)
	got0, early := earthAt(t, 0, true)
	got1, evening := earthAt(t, late, true)
	for i := range bare0 {
		if bare0[i] != got0[i] || bare1[i] != got1[i] {
			t.Fatalf("counting changed what is drawn, at number %d", i)
		}
	}
	for _, c := range []struct {
		kind        string
		early, late int64
	}{
		{"ray hits", early.hits.Load(), evening.hits.Load()},
		{"densities", early.densities.Load(), evening.densities.Load()},
		{"paints", early.paints.Load(), evening.paints.Load()},
	} {
		if c.early == 0 || c.late == 0 {
			t.Errorf("the earth asked for no %s: %d and %d", c.kind, c.early, c.late)
			continue
		}
		if c.late > 2*c.early || c.early > 2*c.late {
			t.Errorf("the earth asks %d %s at nought and %d four and a half "+
				"hours on", c.early, c.kind, c.late)
		}
	}
}
