package gpu

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// fresh is a scene with nothing in it
func fresh() *scene { return newScene() }

// number is the place-th four-byte number of a buffer, as a float
func number(buffer packer, place int) float32 {
	word := binary.LittleEndian.Uint32(buffer.bytes[place*4:])
	return math.Float32frombits(word)
}

// puffs is count plain volumes in a world
func puffs(count int) *thing.World {
	world := thing.NewWorld()
	for range count {
		world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1}})
	}
	return world
}

func TestAsManyVolumesAsARayHoldsAreEncodedAndMoreAreRefused(t *testing.T) {
	encoded := fresh()
	if err := encoded.volumesOf(puffs(mostVolumes), 0); err != nil {
		t.Errorf("%d volumes were refused: %v", mostVolumes, err)
	}
	if encoded.counts.volumes != mostVolumes {
		t.Errorf("%d volumes encoded, want %d", encoded.counts.volumes,
			mostVolumes)
	}
	var refused unsupported
	more := puffs(mostVolumes + 1)
	if err := fresh().volumesOf(more, 0); !errors.As(err, &refused) {
		t.Errorf("%d volumes were not refused: %v", mostVolumes+1, err)
	}
}

func TestAHiddenVolumeAndOneWithNoMediumAreLeftOut(t *testing.T) {
	world := puffs(1)
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1},
		Attrs: thing.Attrs{Hidden: true}})
	world.AddVolume(thing.Volume{})
	encoded := fresh()
	if err := encoded.volumesOf(world, 0); err != nil {
		t.Fatal(err)
	}
	if encoded.counts.volumes != 1 {
		t.Errorf("%d volumes encoded, want the one shown",
			encoded.counts.volumes)
	}
}

func TestAVolumeWithNoScaleIsAtScaleOne(t *testing.T) {
	encoded := fresh()
	encoded.volume(maths.Subspace{}, haze{puff: medium.Puff{Radius: 1}}, 0)
	if got := number(encoded.volumes, 3); got != 1 {
		t.Errorf("a volume with no scale has scale %v, want 1", got)
	}
	encoded.volume(maths.Subspace{Scale: 3}, haze{}, 0)
	second := len(volumeStruct.fields) * 4
	if got := number(encoded.volumes, second+3); got != 3 {
		t.Errorf("a volume of scale 3 has scale %v, want 3", got)
	}
}

func TestOnlyAWispyVolumeIsBaked(t *testing.T) {
	encoded := fresh()
	first, side := encoded.bakeIfWispy(haze{})
	if first != 0 || side != 0 || len(encoded.bakes) != 0 {
		t.Errorf("a plain volume is baked: %d, %d, %v", first, side,
			encoded.bakes)
	}
	first, side = encoded.bakeIfWispy(haze{wispy: true})
	if first != 0 || side != bakeSide || len(encoded.bakes) != 1 {
		t.Errorf("a wispy volume bakes %d, %d, %v", first, side,
			encoded.bakes)
	}
	first, _ = encoded.bakeIfWispy(haze{wispy: true})
	if first != bakeSide*bakeSide*bakeSide {
		t.Errorf("the second bake begins at %d", first)
	}
}

func TestWithNoBakeSideNothingIsBaked(t *testing.T) {
	kept := bakeSide
	bakeSide = 0
	t.Cleanup(func() { bakeSide = kept })
	encoded := fresh()
	first, side := encoded.bakeIfWispy(haze{wispy: true})
	if first != 0 || side != 0 || len(encoded.bakes) != 0 {
		t.Errorf("a bake of no cells: %d, %d, %v", first, side,
			encoded.bakes)
	}
}

// meshOf is a triangle, smooth or not, with normals or without
func meshOf(t *testing.T, smooth, normals bool) *thing.Thing {
	t.Helper()
	var given []maths.Vec
	if normals {
		given = []maths.Vec{maths.V(0, 0, 1), maths.V(0, 0, 1),
			maths.V(0, 0, 1)}
	}
	mesh, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0),
		maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}}, given,
		smooth)
	if err != nil {
		t.Fatal(err)
	}
	return &thing.Thing{Shape: mesh}
}

// only a smooth mesh that has its normals is refused: the kernel has no
// smoothing yet, and a mesh with none to smooth with is flat
func TestOnlyASmoothMeshWithNormalsIsRefused(t *testing.T) {
	cases := []struct {
		smooth, normals, refused bool
	}{{true, true, true}, {true, false, false}, {false, true, false},
		{false, false, false}}
	for _, one := range cases {
		err := fresh().place(meshOf(t, one.smooth, one.normals), 0)
		got := err != nil
		if got != one.refused {
			t.Errorf("smooth %v, normals %v: refused %v, want %v",
				one.smooth, one.normals, got, one.refused)
		}
	}
}

func TestASphereIsAsWideAsItsSpaceMakesIt(t *testing.T) {
	encoded := fresh()
	encoded.sphere(maths.Subspace{Scale: 2}, 3, 0, false)
	encoded.sphere(maths.Subspace{}, 3, 0, false)
	if got := number(encoded.spheres, 3); got != 6 {
		t.Errorf("radius 3 at scale 2 is %v wide, want 6", got)
	}
	second := len(sphereStruct.fields) * 4
	if got := number(encoded.spheres, second+3); got != 3 {
		t.Errorf("a sphere with no scale is %v wide, want 3", got)
	}
}

// lens is the header's lens: half the tangent, the aspect, and the size
func lens(shape film.Pinhole, width, height int) []float32 {
	encoded := fresh()
	encoded.camera(maths.Subspace{}, shape, width, height)
	return []float32{number(encoded.header, 16), number(encoded.header, 17),
		number(encoded.header, 18), number(encoded.header, 19)}
}

func TestTheDefaultLensIsSixtyDegreesAndSquare(t *testing.T) {
	got := lens(film.Pinhole{}, 200, 100)
	if want := float32(math.Tan(30 * maths.Degree)); got[0] != want {
		t.Errorf("the default lens is %v, want %v", got[0], want)
	}
	if got[1] != 2 || got[2] != 200 || got[3] != 100 {
		t.Errorf("aspect, width, height: %v, want 2, 200, 100", got[1:])
	}
}

func TestAWiderPixelWidensTheAspect(t *testing.T) {
	got := lens(film.Pinhole{Wide: 1, Pixel: 2}, 200, 100)
	if got[1] != 4 || got[0] != float32(math.Tan(0.5)) {
		t.Errorf("a lens of 1 radian and pixels twice as wide: %v", got)
	}
}

func TestARefusalSaysWhatIsRefused(t *testing.T) {
	got := unsupported{"a plane"}.Error()
	if want := "gpu: cannot draw a plane yet"; got != want {
		t.Errorf("a refusal says %q, want %q", got, want)
	}
}
