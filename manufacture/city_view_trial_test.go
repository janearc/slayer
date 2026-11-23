package manufacture

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestCityViewChoosesBeforeBuildingAndFailsClosed(t *testing.T) {
	near := CityBuilding{Bounds: maths.Box{Min: maths.V(-1, 0, -11), Max: maths.V(1, 4, -10)}}
	far := CityBuilding{Bounds: maths.Box{Min: maths.V(-1, 0, -101), Max: maths.V(1, 4, -100)}}
	view := cityDetailView{Eye: maths.Local(), Width: 1000, Wide: math.Pi / 2, SmallPixels: 25}
	if got := view.detail(near, "quick", true); got != "quick" {
		t.Fatalf("near = %s", got)
	}
	if got := view.detail(far, "quick", true); got != cityDistantLit {
		t.Fatalf("far = %s", got)
	}
	if got := view.detail(far, "demo", true); got != "demo" {
		t.Fatalf("street detail unexpectedly changed to %s", got)
	}
	if got := view.detail(far, "quick", false); got != "quick" {
		t.Fatalf("day detail unexpectedly changed to %s", got)
	}
	crossing := near
	crossing.Bounds.Max.Z = 1
	if got := view.detail(crossing, "quick", true); got != "quick" {
		t.Fatalf("camera-plane crossing = %s", got)
	}
	bad := view
	bad.Width = 0
	if got := bad.detail(far, "quick", true); got != "quick" {
		t.Fatalf("invalid view = %s", got)
	}
	bad = view
	bad.Wide = math.NaN()
	if got := bad.detail(far, "quick", true); got != "quick" {
		t.Fatalf("NaN view = %s", got)
	}
	bad = view
	bad.SmallPixels = math.Inf(1)
	if got := bad.detail(far, "quick", true); got != "quick" {
		t.Fatalf("invalid threshold = %s", got)
	}
}

func TestDistantNightProxyKeepsSeededLitGeometry(t *testing.T) {
	building := CityBuilding{Seed: 73, Kind: "city-tower", Place: maths.Local()}
	quick := districtBuildingParts(building, "quick", true)
	proxy := districtBuildingParts(building, cityDistantLit, true)
	hull := districtBuildingParts(building, "hull", true)
	byRole := func(parts []part, role slot) []part {
		var out []part
		for _, p := range parts {
			if p.of == role {
				out = append(out, p)
			}
		}
		return out
	}
	for _, role := range []slot{cityWarmWindow, cityCoolWindow} {
		a, b := byRole(quick, role), byRole(proxy, role)
		if len(a) == 0 || !reflect.DeepEqual(a, b) {
			t.Fatalf("seeded lit role %d differs", role)
		}
	}
	if len(byRole(proxy, cityGlass)) != 0 {
		t.Fatal("proxy still builds unlit panes")
	}
	for _, role := range []slot{0, 1, 2, 3, cityRoof} {
		if !reflect.DeepEqual(byRole(proxy, role), byRole(hull, role)) {
			t.Fatalf("silhouette role %d differs", role)
		}
	}
	if len(byRole(quick, cityGlass)) == 0 {
		t.Fatal("fixture has no unlit panes")
	}
}

func TestCityViewOptInMakesLessGeometryWithoutLosingLights(t *testing.T) {
	spec := DefaultClusteredCity()
	spec.Grid.BlocksX, spec.Grid.BlocksZ = 2, 2
	spec.Process.Width = spec.Grid.BlockLength.Times(2)
	spec.Process.Depth = spec.Grid.BlockLength.Times(2)
	spec.Count = 8
	spec.Parks, spec.Highways, spec.Trees = nil, nil, nil
	spec.Night = true
	full, err := (clusteredCityMaker{spec}).madeAt(Sow(73), "quick")
	if err != nil {
		t.Fatal(err)
	}
	view := cityDetailView{Eye: maths.NewSubspace(maths.V(0, 0, 10000), maths.Identity, 1), Width: 1000, Wide: math.Pi / 2, SmallPixels: 20}
	spec.DetailView = &view
	kind, err := ClusteredCity(spec)
	if err != nil {
		t.Fatal(err)
	}
	view.SmallPixels = 0
	got, err := kind.how.(clusteredCityMaker).madeAt(Sow(73), "quick")
	if err != nil {
		t.Fatal(err)
	}
	faces := func(good good, role slot) int {
		if g := good.groups[role]; g != nil {
			return g.FaceCount()
		}
		return 0
	}
	t.Logf("unlit pane faces: full=%d view=%d; warm=%d cool=%d", faces(full, cityGlass), faces(got, cityGlass), faces(got, cityWarmWindow), faces(got, cityCoolWindow))
	if faces(full, cityGlass) == 0 || faces(got, cityGlass) != 0 {
		t.Fatalf("unlit faces full=%d trial=%d", faces(full, cityGlass), faces(got, cityGlass))
	}
	for _, role := range []slot{cityWarmWindow, cityCoolWindow, cityRoof} {
		if faces(full, role) != faces(got, role) {
			t.Fatalf("role %d full=%d trial=%d", role, faces(full, role), faces(got, role))
		}
	}
}

func TestCityViewUsesShotResolutionAndLens(t *testing.T) {
	b := CityBuilding{Bounds: maths.Box{Min: maths.V(-1, -1, -101), Max: maths.V(1, 1, -100)}}
	view := cityDetailView{Eye: maths.Local(), Width: 1000, Wide: math.Pi / 2, SmallPixels: 12}
	if got := view.detail(b, "quick", true); got != cityDistantLit {
		t.Fatalf("wide 1000px = %s", got)
	}
	view.Width = 4000
	if got := view.detail(b, "quick", true); got != "quick" {
		t.Fatalf("wide 4000px = %s", got)
	}
	view.Width = 1000
	view.Wide = math.Pi / 6
	if got := view.detail(b, "quick", true); got != "quick" {
		t.Fatalf("narrow lens = %s", got)
	}
	view.Wide = math.Pi / 2
	view.Eye = maths.NewSubspace(maths.V(0, 0, -99), maths.Identity, 1)
	if got := view.detail(b, "quick", true); got != "quick" {
		t.Fatalf("near eye = %s", got)
	}
}

func TestCityViewUnknownBoundsNeverReduceDetail(t *testing.T) {
	view := cityDetailView{Eye: maths.Local(), Width: 1000, Wide: math.Pi / 2, SmallPixels: 20}
	boxes := []maths.Box{
		{Min: maths.V(-1, -1, -11), Max: maths.V(1, 1, 1)},
		{Min: maths.V(-1, -1, 1), Max: maths.V(1, 1, 11)},
		{Min: maths.V(2, -1, -11), Max: maths.V(1, 1, -10)},
		{Min: maths.V(math.NaN(), -1, -11), Max: maths.V(1, 1, -10)},
	}
	for _, box := range boxes {
		if got := view.detail(CityBuilding{Bounds: box}, "quick", true); got != "quick" {
			t.Errorf("uncertain box %+v = %s", box, got)
		}
	}
	if got := view.detail(CityBuilding{Bounds: maths.Box{Min: maths.V(-1, -1, -101), Max: maths.V(1, 1, -100)}}, "hull", true); got != "hull" {
		t.Errorf("authored hull changed to %s", got)
	}
}

func TestCityViewRefusesInvalidFrames(t *testing.T) {
	box := CityBuilding{Bounds: maths.Box{Min: maths.V(-1, -1, -101), Max: maths.V(1, 1, -100)}}
	view := cityDetailView{Eye: maths.Local(), Width: 1000, Wide: math.Pi / 2, SmallPixels: 20}
	for _, turn := range []maths.Orientation{
		{X: maths.XAxis, Y: maths.YAxis},
		{X: maths.XAxis, Y: maths.XAxis, Z: maths.ZAxis},
		{X: maths.XAxis, Y: maths.YAxis, Z: maths.ZAxis.Scale(-1)},
		{X: maths.V(math.NaN(), 0, 0), Y: maths.YAxis, Z: maths.ZAxis},
	} {
		view.Eye.Orientation = turn
		if got := view.detail(box, "quick", true); got != "quick" {
			t.Errorf("invalid frame changed detail: %s", got)
		}
	}
	view.Eye.Orientation = maths.About(maths.ZAxis, math.Pi/4)
	if got := view.detail(box, "quick", true); got != cityDistantLit {
		t.Errorf("rotated frame: %s", got)
	}
}

func TestCityViewRefusesUnrepresentableProjection(t *testing.T) {
	view := cityDetailView{Eye: maths.Local(), Width: 1000, Wide: math.SmallestNonzeroFloat64, SmallPixels: 20}
	box := CityBuilding{Bounds: maths.Box{Min: maths.V(-1, -1, -11), Max: maths.V(1, 1, -10)}}
	if got := view.detail(box, "quick", true); got != "quick" {
		t.Fatalf("overflow projection changed detail to %s", got)
	}
}
