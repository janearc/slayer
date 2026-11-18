package named

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

func TestOrbitalSurfaceTouchesNativeFloorAndRisesInward(t *testing.T) {
	ring := hoop{radius: orbitalRadius}
	floor, _, _, err := hoopMeshes(0, ring.radius, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, angle := range []float64{.0001, .7, 2.8, 5.4} {
		site := maths.V(ring.radius*math.Sin(angle), 0, ring.radius*math.Cos(angle))
		for _, east := range []float64{-90000, 0, 70000} {
			foot := ring.surfacePoint(site, maths.V(east, 0, 10000))
			if distance := math.Abs(floor.Distance(foot)); distance > 1e-8 {
				t.Fatalf("base is %g scene units off native floor", distance)
			}
			top := ring.surfacePoint(site, maths.V(east, 250000, 10000))
			if math.Abs(foot.Sub(top).Len()-.25) > 1e-9 {
				t.Fatal("250 km height lost")
			}
			if math.Hypot(top.X, top.Z) >= math.Hypot(foot.X, foot.Z) {
				t.Fatal("construction points out of inhabited floor")
			}
		}
	}
}

func TestOrbitalReliefTapersAwayFromBuildingsAndShore(t *testing.T) {
	seed := uint64(1203)
	areas := []orbitalFootprint{{radius: 1050000}}
	for i := 0; i < 4; i++ {
		areas = append(areas, orbitalFootprint{offset: orbitalDistrictOffset(seed, i), radius: 700000})
	}
	if orbitalReliefClearance(0, 0, areas) > 0 || orbitalReliefClearance(orbitalReliefHalf, 0, areas) > 0 {
		t.Fatal("building or patch edge has no clearance")
	}
	for i := 0; i < 4; i++ {
		p := orbitalDistrictOffset(seed, i)
		if orbitalReliefClearance(p.X, p.Z, areas) > 0 {
			t.Fatal("district footprint covered")
		}
	}
	if orbitalReliefHeight(2e6, 3e6, seed, -1, .2) != 0 || orbitalReliefHeight(2e6, 3e6, seed, 1e6, 0) != 0 {
		t.Fatal("relief did not return to ground at a boundary")
	}
	if orbitalReliefHeight(2e6, 3e6, seed, 1, .2) > .01 {
		t.Fatal("relief does not taper smoothly")
	}
	a := orbitalReliefHeight(2e6, 3e6, seed, 1e6, .2)
	b := orbitalReliefHeight(2.3e6, 3.2e6, seed, 1e6, .2)
	if a <= 0 || b <= 0 || math.Abs(a-b) < 1 {
		t.Fatal("relief is missing physical variation")
	}
}

func TestOrbitalReliefMeshLeavesBuildingAndWaterFootprintsOpen(t *testing.T) {
	ring := nested()[0]
	tiles := hexes.Band{Radius: ring.radius, Width: bandWidth, Around: 128}
	geography := biome.Make(biome.Fields{Tiling: tiles, Seed: 1806, Scale: floorScale, Sea: floorSea, Sprawl: floorSprawl})
	var centre maths.Vec
	found := false
	for _, feature := range geography.Features() {
		if feature.City && math.Abs(feature.At.Y) < bandWidth/2-6 {
			centre = feature.At
			found = true
			break
		}
	}
	if !found {
		t.Skip("small synthetic biome did not contain a city")
	}
	world := thing.NewWorld()
	paint := newOrbitalMetreFloor(ring.radius, 10, 1806)
	paint.geography = geography
	paint.macro = tiles
	areas := []orbitalFootprint{{radius: 1050000}, {offset: maths.V(0, 0, -1400000), radius: 905100}}
	offset, _, err := ring.districtLand(geography, centre, 1203, 0, 100000, areas)
	if err != nil || !ring.surfaceFootprintLand(geography, centre, offset, 100000) {
		t.Fatalf("could not place a dry district building: %v", err)
	}
	for _, area := range areas {
		if math.Hypot(offset.X-area.offset.X, offset.Z-area.offset.Z) < 100000+area.radius {
			t.Fatal("district building overlaps a reserved structure")
		}
	}
	if err := ring.surfaceRelief(world, geography, &thing.Material{Texture: paint}, centre, 1203, 24*time.Hour, areas); err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 1 {
		t.Fatal("land terrain not created")
	}
	land := world.Things[0].Shape.(*mesh.Mesh)
	points := land.Points()
	faces := land.Faces()
	if len(faces) < 100 {
		t.Fatal("terrain mesh is unexpectedly empty")
	}
	angle := math.Atan2(centre.X, centre.Z)
	for _, face := range faces {
		for _, index := range face {
			p := points[index]
			x := math.Remainder(math.Atan2(p.X, p.Z)-angle, 2*math.Pi) * ring.radius * 1e6
			z := (p.Y - centre.Y) * 1e6
			if orbitalReliefClearance(x, z, areas) <= 0 {
				t.Fatal("terrain overlapped a structure footprint")
			}
			base := ring.surfacePoint(centre, maths.V(x, 0, z))
			if geography.WaterAt(base) {
				t.Fatal("terrain covered an existing river, lake or ocean")
			}
		}
	}
}

func TestOrbitalTourMovesAndOffsetResumesSamePath(t *testing.T) {
	ring := nested()[0]
	frame := ring.surfaceFrame(maths.V(0, 0, ring.radius))
	frame.Scale = .3
	frame.Origin = ring.surfacePoint(maths.V(0, 0, ring.radius), maths.V(0, .45*frame.Scale*1e6, 0))
	site := orbitalSurfaceSite{frame: frame, motion: ring.turning(0)}
	first, err := site.tour(12*time.Second, 0)(0)
	if err != nil {
		t.Fatal(err)
	}
	middle, err := site.tour(12*time.Second, 0)(8 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := site.tour(12*time.Second, 8*time.Second)(0)
	if err != nil {
		t.Fatal(err)
	}
	if first.Origin.Near(middle.Origin, 1e-8) || !middle.Origin.Near(resumed.Origin, 1e-8) || middle.Orientation != resumed.Orientation {
		t.Fatal("tour did not move or resume at the same view")
	}
	boundary := 10*time.Second + 909090909*time.Nanosecond
	beforeCut, _ := site.tour(15*time.Second, 0)(boundary)
	afterCut, _ := site.tour(15*time.Second, boundary)(0)
	last, _ := site.tour(15*time.Second, boundary)(3636363636 * time.Nanosecond)
	if !beforeCut.Origin.Near(afterCut.Origin, 1e-9) || beforeCut.Orientation != afterCut.Orientation || afterCut.Origin.Near(last.Origin, 1e-6) {
		t.Fatal("film cut does not resume a moving tour")
	}
	for _, at := range []time.Duration{0, 3750 * time.Millisecond, 7500 * time.Millisecond, 11250 * time.Millisecond, 15 * time.Second} {
		eye, err := site.tour(15*time.Second, 0)(at)
		if err != nil {
			t.Fatal(err)
		}
		local := site.motion.At(at).In(eye.Origin)
		angle := math.Atan2(local.X, local.Z)
		x := angle * ring.radius * 1e6
		z := local.Y * 1e6
		altitude := (ring.surfaceRadius(angle) - math.Hypot(local.X, local.Z)) * 1e6
		if math.Abs(x) > 640000 || z < -2040000 || z > -760000 || altitude < 44000 || altitude > 76000 {
			t.Fatalf("tour leaves city or safe altitude at %v: x=%g z=%g height=%g", at, x, z, altitude)
		}
	}
}

func TestOrbitalCityTourFollowsParkPathAndResumes(t *testing.T) {
	ring := nested()[0]
	centre := maths.V(0, 0, ring.radius)
	site := orbitalSurfaceSite{ring: ring, centre: centre, motion: ring.turning(0), cityTour: true,
		landmark: maths.V(-900000, 100000, -1400000), landmarkRadius: 120000}
	boundary := 10*time.Second + 909090909*time.Nanosecond
	before, _ := site.tour(15*time.Second, 0)(boundary)
	after, _ := site.tour(15*time.Second, boundary)(0)
	if !before.Origin.Near(after.Origin, 1e-12) || before.Orientation != after.Orientation {
		t.Fatal("park tour changes pose at the film cut")
	}
	for _, at := range []time.Duration{0, 3750 * time.Millisecond, 7500 * time.Millisecond} {
		eye, err := site.tour(15*time.Second, 0)(at)
		if err != nil {
			t.Fatal(err)
		}
		local := site.motion.At(at).In(eye.Origin)
		angle := math.Atan2(local.X, local.Z)
		x := angle * ring.radius * 1e6
		z := (local.Y - centre.Y) * 1e6
		altitude := (ring.surfaceRadius(angle) - math.Hypot(local.X, local.Z)) * 1e6
		park := manufacture.DefaultClusteredCity().Parks[0]
		walk := 2 * float64(at) / float64(15*time.Second)
		walk = 1 - (1-walk)*(1-walk)
		path := manufacture.CityParkLongPathAt(park, .5+.0003*walk)
		ground := manufacture.CityParkTerrainHeight(park, path)
		if math.Abs(x-path.X*orbitalCityScale) > .01 || math.Abs(z-(path.Z*orbitalCityScale-1400000)) > .01 || math.Abs(altitude-(35+path.Y*orbitalCityScale)) > .01 || (path.Y-ground)*orbitalCityScale+25 < 25 {
			t.Fatalf("park camera left its path or clearance at %v: x=%g z=%g altitude=%g", at, x, z, altitude)
		}
	}
	last, _ := site.tour(15*time.Second, 0)(15 * time.Second)
	local := site.motion.At(15 * time.Second).In(last.Origin)
	angle := math.Atan2(local.X, local.Z)
	x, z := angle*ring.radius*1e6, (local.Y-centre.Y)*1e6
	altitude := (ring.surfaceRadius(angle) - math.Hypot(local.X, local.Z)) * 1e6
	distance := math.Hypot(x-site.landmark.X, z-site.landmark.Z)
	if distance < site.landmarkRadius+50000 || distance > site.landmarkRadius+120000 || math.Abs(altitude-160000) > .01 {
		t.Fatalf("tour did not safely approach its landmark: distance=%g altitude=%g", distance, altitude)
	}
}

func TestOrbitalDistrictFlybyIsOutsideCityAndFocalFootprints(t *testing.T) {
	ring := nested()[0]
	centre := maths.V(0, 0, ring.radius)
	placed := []orbitalFootprint{{radius: 1050000}, {offset: maths.V(0, 0, -1400000), radius: 905100}}
	const radius = 120000
	where, ok := ring.districtFlybyLand(nil, centre, radius, placed)
	if !ok || !ring.districtFits(nil, centre, where, radius, placed, true) {
		t.Fatal("no non-overlapping flyby landmark")
	}
	if math.Abs(where.X+225000) > 800000 || math.Abs(where.Z+1350000) > 400000 {
		t.Fatalf("flyby landmark is too far from the park corridor: %v", where)
	}
}

func TestOrbitalFlybyRejectsBlockerOnCurvedRoute(t *testing.T) {
	const radius = 120000
	target := maths.V(-900000, 100000, -1400000)
	park := manufacture.DefaultClusteredCity().Parks[0]
	from := manufacture.CityParkLongPathAt(park, .5003).Scale(orbitalCityScale).Add(orbitalCityOffset)
	end := target.Add(maths.V(radius+50000, 60000, -100000))
	control := target.Add(maths.V(radius+250000, 200000, -120000))
	blocker := orbitalBezier(from, from, control, end, .5)
	placed := []orbitalFootprint{{offset: maths.V(1e8, 0, 0), radius: 1}, {offset: orbitalCityOffset, radius: 905100}, {offset: blocker, radius: 1000}}
	if orbitalFlybyClear(target, radius, placed) {
		t.Fatal("accepted a landmark overlapping the actual curved camera route")
	}
	placed[2].offset.X += 2e6
	if !orbitalFlybyClear(target, radius, placed) {
		t.Fatal("rejected a clear camera route")
	}
}

func TestOrbitalCityTourClearsExistingBuildings(t *testing.T) {
	ring := nested()[0]
	centre := maths.V(0, 0, ring.radius)
	site := orbitalSurfaceSite{ring: ring, centre: centre, motion: ring.turning(0), cityTour: true,
		landmark: maths.V(-900000, 100000, -1400000), landmarkRadius: 120000}
	for _, seed := range []uint64{31, 1806} {
		plan, err := manufacture.PlanClusteredCity(manufacture.DefaultClusteredCity(), manufacture.Sow(seed))
		if err != nil {
			t.Fatal(err)
		}
		buildings := plan.Buildings()
		for i := 0; i <= 300; i++ {
			at := time.Duration(float64(15*time.Second) * float64(i) / 300)
			eye, err := site.tour(15*time.Second, 0)(at)
			if err != nil {
				t.Fatal(err)
			}
			local := site.motion.At(at).In(eye.Origin)
			angle := math.Atan2(local.X, local.Z)
			point := maths.V(angle*ring.radius*1e6/orbitalCityScale,
				((ring.surfaceRadius(angle)-math.Hypot(local.X, local.Z))*1e6-orbitalCityOffset.Y)/orbitalCityScale,
				(local.Y*1e6-orbitalCityOffset.Z)/orbitalCityScale)
			for _, building := range buildings {
				box := building.Bounds
				if point.X >= box.Min.X-.06 && point.X <= box.Max.X+.06 && point.Z >= box.Min.Z-.06 && point.Z <= box.Max.Z+.06 && point.Y >= box.Min.Y-.02 && point.Y <= box.Max.Y+.02 {
					t.Fatalf("tour intersects building seed %d at %v for city seed %d", building.Seed, at, seed)
				}
			}
		}
	}
}

func TestOrbitalDistrictFamilyVariesWithoutRepeatingCycle(t *testing.T) {
	previous := -1
	var choices []int
	for i := 0; i < 24; i++ {
		own := uint64(71 + (i+1)*100003)
		choice := orbitalDistrictFamily(own, i, previous, true)
		if choice == previous || choice < 0 || choice > 2 {
			t.Fatalf("repeated or invalid family at %d: %d", i, choice)
		}
		choices, previous = append(choices, choice), choice
	}
	periodic := true
	for i := 3; i < len(choices); i++ {
		periodic = periodic && choices[i] == choices[i-3]
	}
	if periodic {
		t.Fatalf("district still cycles through three silhouettes: %v", choices)
	}
}

func TestOrbitalCityAvoidsOtherGlobalStructures(t *testing.T) {
	ring := nested()[0]
	centre := maths.V(0, 0, ring.radius)
	own := orbitalFootprint{offset: centre, radius: 500000}
	if ring.cityOverlapsSites(centre, []orbitalFootprint{own}) {
		t.Fatal("focus rejected itself")
	}
	near := orbitalFootprint{offset: ring.surfacePoint(centre, maths.V(0, 0, -1400000)), radius: 100000}
	if !ring.cityOverlapsSites(centre, []orbitalFootprint{own, near}) {
		t.Fatal("city admitted an overlapping global site")
	}
	far := orbitalFootprint{offset: ring.surfacePoint(centre, maths.V(5000000, 0, 0)), radius: 100000}
	if ring.cityOverlapsSites(centre, []orbitalFootprint{own, far}) {
		t.Fatal("city rejected a distant global site")
	}
}

func TestOrbitalNightReversesSunAndKeepsFill(t *testing.T) {
	world := thing.NewWorld()
	nearView(world)
	if len(world.Lamps) != 2 {
		t.Fatal("missing daylight")
	}
	reverseOrbitalSun(world)
	sun, ok := world.Lamps[0].(film.Sun)
	if !ok || sun.Toward != maths.V(nearSunX, nearSunY, nearSunZ).Neg() {
		t.Fatal("night sun did not move across the sky")
	}
	if _, ok := world.Lamps[1].(film.Flood); !ok {
		t.Fatal("night lost its space fill")
	}
}

func TestOrbitalDistrictAddsDistinctMovingStructures(t *testing.T) {
	ring := nested()[0]
	world := thing.NewWorld()
	centre := maths.V(0, 0, ring.radius)
	if _, err := ring.surfaceDistrict(world, nil, centre, 71, 24*time.Hour, 3, true); err != nil {
		t.Fatal(err)
	}
	if len(world.Things) < 9 {
		t.Fatalf("three structures yielded only %d parts", len(world.Things))
	}
	lava, windows := false, false
	for _, part := range world.Things {
		if part.Motion == nil || part.At(time.Hour) != ring.turning(24*time.Hour).At(time.Hour) {
			t.Fatal("district part lost ring motion")
		}
		lava = lava || part.Material.Name == "landform authored caldera lava"
		windows = windows || part.Material.Name == "ancient living macro blue glazing" && part.Material.Shine != nil
	}
	if !lava || !windows {
		t.Fatalf("district lost volcanic lava or illuminated windows: %v %v", lava, windows)
	}
}

func TestOrbitalSurfaceNativeSpecimenAndCameraShareRingMotion(t *testing.T) {
	ring := nested()[0]
	world := thing.NewWorld()
	spec := manufacture.DefaultMacrostructure(manufacture.TwistedSpires)
	spec.Wildness = .85
	spec.LivingCover = true
	kind, err := manufacture.Macrostructure(spec)
	if err != nil {
		t.Fatal(err)
	}
	site := maths.V(0, 0, ring.radius)
	if err = ring.surfaceSpecimen(world, kind, 31, site, 24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 4 {
		t.Fatalf("got %d native material meshes", len(world.Things))
	}
	camera := orbitalSurfaceSite{frame: ring.surfaceFrame(site), motion: ring.turning(24 * time.Hour)}
	at := 6 * time.Hour
	eye, err := camera.camera(at)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range world.Things {
		if part.At(at) != camera.motion.At(at) {
			t.Fatal("part lost ring clock")
		}
	}
	localEye, _ := camera.camera(0)
	// The ring is already oriented at zero; compare by returning both views to ring space.
	if !camera.motion.At(at).In(eye.Origin).Near(camera.motion.At(0).In(localEye.Origin), 1e-9) {
		t.Fatal("camera is not carried by ring")
	}
}
