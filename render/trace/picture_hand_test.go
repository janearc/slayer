package trace

import (
	"math"
	"sync"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// schlick is the share of light a glossy surface of gloss one mirrors
// when seen at cos to its normal, by the textbook form of schlick's
// fresnel: f0 + (1 - f0)(1 - cos)^5, with f0 the square of (n - 1) over
// (n + 1) and n the index of refraction, 1.5 for glass.
func schlick(cos float64) float64 {
	f0 := math.Pow((1.5-1)/(1.5+1), 2)
	return f0 + (1-f0)*math.Pow(1-cos, 5)
}

// mirrorFloor is a glossy floor of grey 0.5 under a sun straight overhead
// of one, so that what it gives back by itself is 0.5, and under a sky
// that says which way it was looked at.
func mirrorFloor(gloss float64) *thing.World {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.5), Gloss: gloss}})
	w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	w.Sky = compass{}
	return w
}

// a glossy floor mirrors the sky it faces by fresnel's share at the angle
// it is seen. the ray comes down at 0.6 across and 0.8 down, 0.8 to the
// normal, so schlick gives 0.04 plus 0.96 by 0.2 to the fifth.
//
// what is mirrored is the sky on the other side of the normal, along 0.6
// across and 0.8 up, which the compass sky gives as its own light. the
// floor's own 0.5 is kept by the rest: 0.5 times one less the share.
func TestAGlossyFloorMirrorsTheSkyTheOtherSideOfTheNormal(t *testing.T) {
	r := maths.Ray{At: maths.V(-3, 4, 0), Dir: maths.V(0.6, -0.8, 0)}
	share := schlick(0.8)
	if math.Abs(share-(0.04+0.96*math.Pow(0.2, 5))) > 1e-15 {
		t.Fatalf("the worked share is %v", share)
	}
	want := maths.Grey(0.5 * (1 - share)).Add(maths.RGB(0.6, 0.8, 0).Scale(share))
	colours(t, "a glossy floor", Trace(mirrorFloor(1), r, 0), want, 2e-6)
	// a gloss of a half mirrors half as much
	half := maths.Grey(0.5 * (1 - share/2)).Add(maths.RGB(0.6, 0.8, 0).Scale(share / 2))
	colours(t, "a half-glossy floor", Trace(mirrorFloor(0.5), r, 0), half, 2e-6)
}

// a dull surface asks the sky nothing: a floor that does not shine, under
// a sky that gives no number of light, is as lit as it was.
func TestADullFloorAsksTheSkyNothing(t *testing.T) {
	w := mirrorFloor(0)
	w.Sky = broken{}
	r := maths.Ray{At: maths.V(-3, 4, 0), Dir: maths.V(0.6, -0.8, 0)}
	colours(t, "a dull floor", Trace(w, r, 0), maths.Grey(0.5), 2e-6)
}

// broken is a sky that is no number of light in every direction.
type broken struct{}

func (broken) Light(maths.Vec) maths.Light { return maths.Grey(math.NaN()) }

// a mirror mirrors once: a glossy floor sees a glossy dark ball in its
// mirror, and the ball, seen in a mirror, does not mirror the sky in its
// turn, so the ball is black there and the floor is its own 0.5 less the
// share and nothing more.
//
// the ray comes down at 45 degrees, so the share is schlick's at the
// cosine of root a half; the mirrored ray runs up at 45 degrees the other
// way through the ball's middle. the same through a whole picture of one
// pixel, whose one ray is that one.
func TestAMirrorSeenInAMirrorMirrorsNothing(t *testing.T) {
	w := mirrorFloor(1)
	w.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5}, Material: &thing.Material{Gloss: 1},
		Motion: at(maths.V(0, 1, 0), 1)})
	want := maths.Grey(0.5 * (1 - schlick(math.Sqrt(0.5))))
	r := maths.Ray{At: maths.V(0, 2, 3), Dir: maths.V(0, -1, -1).Unit()}
	colours(t, "a mirror in a mirror, traced", Trace(w, r, 0), want, 2e-6)
	from := maths.V(0, 4, 5)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 0, 1))}
	f := Render(w, eye, film.Pinhole{Wide: 20 * maths.Degree}, 0, 1, 1)
	colours(t, "a mirror in a mirror, drawn", f.At(0, 0), want, 2e-6)
}

// probe is a texture that paints a quarter and keeps what it was asked:
// plainly, or for a pixel of a footprint.
type probe struct {
	mu     sync.Mutex
	plain  int
	within []asked
}

type asked struct {
	at        maths.Vec
	footprint float64
}

func (p *probe) At(maths.Vec) maths.Light {
	p.mu.Lock()
	p.plain++
	p.mu.Unlock()
	return maths.Grey(0.25)
}

func (p *probe) Within(at maths.Vec, footprint float64) maths.Light {
	p.mu.Lock()
	p.within = append(p.within, asked{at, footprint})
	p.mu.Unlock()
	return maths.Grey(0.25)
}

// pinholeAngle is the angle between the rays of the middle pixel and the
// next, in a picture w by h of a pinhole that sees wide across. the rays
// are (u, v, -1), with u and v the pixel's place on a plane one away,
// and the angle is the arctangent of the cross over the dot.
func pinholeAngle(wide, pixel float64, w, h int) float64 {
	half := math.Tan(wide / 2)
	aspect := float64(w) * pixel / float64(h)
	ray := func(x, y int) maths.Vec {
		return maths.V((2*(float64(x)+0.5)/float64(w)-1)*half,
			(1-2*(float64(y)+0.5)/float64(h))*half/aspect, -1)
	}
	a, b := ray(w/2, h/2), ray(w/2+1, h/2)
	return math.Atan2(a.Cross(b).Len(), a.Dot(b))
}

// the width of a pixel is the angle between the middle pixel's ray and
// the next one's: 120 degrees across 16 pixels by 9 of three quarters of
// a square, worked from the pinhole's own geometry. it is what a surface
// takes as its pixel, and nothing else.
func TestAPixelIsAsWideAsTheMiddleOneBesideTheNext(t *testing.T) {
	lens := film.Pinhole{Wide: 120 * maths.Degree, Pixel: 0.75}
	for _, size := range [][2]int{{16, 9}, {15, 8}, {40, 22}} {
		got := pixelAngle(maths.Subspace{}, lens, size[0], size[1])
		want := pinholeAngle(120*maths.Degree, 0.75, size[0], size[1])
		if !near(got, want, 1e-12) {
			t.Errorf("%v: a pixel is %v wide, want %v", size, got, want)
		}
	}
}

// aimed is a sensor that sends the middle pixel along one way and every
// other along another.
type aimed struct{ middle, other maths.Vec }

func (a aimed) Ray(x, y, w, h int, _ maths.Band) maths.Ray {
	if x == w/2 {
		return maths.Ray{Dir: a.middle}
	}
	return maths.Ray{Dir: a.other}
}

// the angle between a pixel and its neighbour is zero for two rays the same way
// and a half turn for two opposite, even where the dot of their unit lengths
// comes out a hair past one or past minus one, which acos does not take.
//
// the three ways here were found by a search of twenty thousand. on arm64,
// where the compiler joins the multiplies and adds, the dot of each one's unit
// vector with itself is one and one part in 2^52, and with its opposite minus
// that.
//
// the clamp makes the zero and the half turn. elsewhere they may round inside,
// and the test holds all the same.
func TestAnAngleOfNoneAndOfAHalfTurnAreNotNoNumber(t *testing.T) {
	for _, way := range []maths.Vec{
		maths.V(0.45752021798627124, 1.9809907523197885, 1.9678084990486833),
		maths.V(0.4071213906625421, 1.6696092515947942, 1.0081101277800955),
		maths.V(1.346113138604105, 1.075754497434352, 1.7231637154292718)} {
		same := aimed{way, way}
		if got := pixelAngle(maths.Subspace{}, same, 8, 8); got != 0 {
			t.Errorf("two rays along %v are %v apart", way, got)
		}
		opposite := aimed{way, way.Neg()}
		if got := pixelAngle(maths.Subspace{}, opposite, 8, 8); !near(got, math.Pi, 1e-12) {
			t.Errorf("two rays, one along %v and one against, are %v apart", way, got)
		}
	}
}

// a surface paints for the pixel asking: a lone ray asks for no
// footprint, and a picture asks for the pixel's width, in radians, times
// how far the surface is, in the surface's own units.
//
// the floor is painted a quarter by a texture, which it must be asked,
// so the light is a quarter of the sun. seen from 10 up, a pixel's
// footprint is its angle times the way to the floor point, and a floor
// doubled in size has half the footprint for the same ray.
//
// a world that sets the pixel sets the footprint.
func TestASurfaceIsAskedForTheFootprintOfThePixel(t *testing.T) {
	// a lone ray asks plainly
	tex := &probe{}
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Colour: maths.Grey(1), Texture: tex}})
	w.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	colours(t, "a lone ray", Trace(w, straightDown(), 0), maths.Grey(0.25), 2e-6)
	if tex.plain != 1 || len(tex.within) != 0 {
		t.Errorf("a lone ray asked plainly %d times and by footprint %d", tex.plain, len(tex.within))
	}

	from := maths.V(0.2, 10, 0.1)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 0, 0))}
	lens := film.Pinhole{Wide: 120 * maths.Degree, Pixel: 0.75}
	angle := pinholeAngle(120*maths.Degree, 0.75, 16, 9)
	for _, c := range []struct {
		name  string
		scale float64
		pixel float64
		want  float64
	}{
		{"a floor", 1, 0, angle},
		{"a floor doubled", 2, 0, angle},
		{"a floor in a world that sets the pixel", 1, 0.01, 0.01},
	} {
		tex := &probe{}
		w := thing.NewWorld()
		w.Pixel = c.pixel
		w.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{Colour: maths.Grey(1), Texture: tex},
			Motion: at(maths.Vec{}, c.scale)})
		f := Render(w, eye, lens, 0, 16, 9)
		if tex.plain != 0 || len(tex.within) != 16*9 {
			t.Errorf("%s: asked plainly %d times and by footprint %d", c.name, tex.plain, len(tex.within))
		}
		for _, a := range tex.within {
			// the world point is the local point times the scale, on the floor
			world := a.at.Scale(c.scale)
			dist := world.Sub(from).Len()
			if !near(a.footprint, dist*c.want/c.scale, 1e-12) {
				t.Fatalf("%s: a pixel at %v away asked for %v, want %v", c.name, dist, a.footprint,
					dist*c.want/c.scale)
			}
		}
		if f.At(8, 4).Cover != 1 {
			t.Errorf("%s: the floor is not there", c.name)
		}
	}
}

// every pixel of a picture is drawn: the first row and the first column
// of a picture of a floor are floor.
func TestEveryPixelOfAPictureIsDrawn(t *testing.T) {
	from := maths.V(0, 10, 0)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 0, 0))}
	f := Render(floor(), eye, film.Pinhole{Wide: 60 * maths.Degree}, 0, 7, 5)
	for y := 0; y < 5; y++ {
		for x := 0; x < 7; x++ {
			if p := f.At(x, y); p.Cover != 1 || math.IsInf(float64(p.Depth), 1) {
				t.Errorf("the pixel at %d, %d is %+v", x, y, p)
			}
		}
	}
}

// a picture sampled n by n a pixel is each of its pixels the average of
// the n by n fine pixels in its place, as light, with cover averaged too
// and the depth the nearest. it is worked by drawing the picture n times
// finer and averaging by hand. at one or less it is the plain picture.
func TestASampledPictureIsTheAverageOfAFinerOne(t *testing.T) {
	// seen from above, so that every pixel, the first row among them, is
	// floor or ball
	w := smoking()
	from := maths.V(0, 6, 2)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, centre)}
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	const width, height = 6, 4
	for _, n := range []int{2, 3} {
		got := Sampled(w, eye, lens, 0, width, height, n)
		fine := Render(w, eye, lens, 0, width*n, height*n)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				var r, g, b, cover float64
				depth := math.Inf(1)
				for sy := 0; sy < n; sy++ {
					for sx := 0; sx < n; sx++ {
						q := fine.At(x*n+sx, y*n+sy)
						r, g, b, cover = r+float64(q.R), g+float64(q.G), b+float64(q.B), cover+float64(q.Cover)
						depth = math.Min(depth, float64(q.Depth))
					}
				}
				k := 1 / float64(n*n)
				p := got.At(x, y)
				if !within(p.R, r*k, 1e-6) || !within(p.G, g*k, 1e-6) || !within(p.B, b*k, 1e-6) ||
					!within(p.Cover, cover*k, 1e-6) ||
					!(float64(p.Depth) == depth || near(float64(p.Depth), depth, 1e-5*depth)) {
					t.Fatalf("n=%d at %d, %d: %+v, want %v %v %v cover %v depth %v", n, x, y, p, r*k, g*k, b*k, cover*k, depth)
				}
			}
		}
	}
	plain := Render(w, eye, lens, 0, width, height)
	for _, n := range []int{1, 0, -2} {
		got := Sampled(w, eye, lens, 0, width, height, n)
		for i := range plain.Pix {
			if got.Pix[i] != plain.Pix[i] {
				t.Fatalf("n=%d: pixel %d is %+v, the plain picture's is %+v", n, i, got.Pix[i], plain.Pix[i])
			}
		}
	}
}
