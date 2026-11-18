package manufacture

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// measuredSeeds is how many seeded buildings of a family are measured.
const measuredSeeds = 400

// gpuTriangleBytes is a pooled face as the GPU holds it: three float4
// corners in the Triangle buffer and three float4 vertices for Metal.
const gpuTriangleBytes = 96

// cityGeometryHash digests every group member's mesh and placement in
// slot order, so a level that must not change can be compared exactly.
func cityGeometryHash(made good) string {
	digest := sha256.New()
	slots := make([]int, 0, len(made.groups))
	for of := range made.groups {
		slots = append(slots, int(of))
	}
	sort.Ints(slots)
	word := func(value float64) {
		_ = binary.Write(digest, binary.LittleEndian, math.Float64bits(value))
	}
	vector := func(at maths.Vec) { word(at.X); word(at.Y); word(at.Z) }
	for _, of := range slots {
		group := made.groups[slot(of)]
		word(float64(of))
		for index := range group.MemberCount() {
			member := group.Member(index)
			vector(member.Space.Origin)
			word(member.Space.Scale)
			vector(member.Space.Orientation.Out(maths.XAxis))
			vector(member.Space.Orientation.Out(maths.YAxis))
			if solid, ok := member.Mesh.(*mesh.Mesh); ok {
				for _, point := range solid.Points() {
					vector(point)
				}
				for _, three := range solid.Faces() {
					word(float64(three[0]))
					word(float64(three[1]))
					word(float64(three[2]))
				}
			}
		}
	}
	return fmt.Sprintf("%x", digest.Sum(nil))[:16]
}

// cityCensus is a made city's members, unique meshes, the faces pooled
// once, the faces of meshes placed only once, and the faces placed.
type cityCensus struct {
	members, unique, stored, single, placed int
}

// census counts what the GPU pools once and what it places.
func census(made good) cityCensus {
	var counted cityCensus
	uses := map[shape.Shape]int{}
	for _, group := range made.groups {
		counted.members += group.MemberCount()
		counted.placed += group.FaceCount()
		for index := range group.MemberCount() {
			uses[group.Member(index).Mesh]++
		}
	}
	for _, skin := range made.skins() {
		counted.stored += len(skin)
		counted.single += len(skin)
	}
	counted.unique = len(uses)
	for one, used := range uses {
		if solid, ok := one.(*mesh.Mesh); ok {
			counted.stored += solid.FaceCount()
			if used == 1 {
				counted.single += solid.FaceCount()
			}
		}
	}
	return counted
}

// cityBuildingFaces is one seeded building made alone at a level: the
// faces only it has, and every face it shows, modules counted where
// they stand.
func cityBuildingFaces(family string, number uint64,
	detail string) (int, int) {
	own := sow(citySeed(number))
	design := cityBuildingDesign(family, own)
	batch := newCityBatch()
	if detail == cityStreetLevel {
		batch.detailed(design, own, maths.Local(), false)
	} else {
		batch.add(design.parts(detail), maths.Local())
	}
	made, err := batch.finish()
	if err != nil {
		return 0, 0
	}
	counted := census(made)
	return counted.single, counted.placed
}

// TestCityDetailMeasures logs faces per building and the whole city's
// pooled and placed faces at every city level. It asserts nothing but
// that each level makes; the numbers are the design note's evidence.
func TestCityDetailMeasures(t *testing.T) {
	if testing.Short() {
		t.Skip("measures the whole city")
	}
	for _, family := range []string{"city-rowhouse", "city-apartment",
		"city-tower"} {
		for _, detail := range cityLevels() {
			alone, total, most := 0, 0, 0
			for number := range uint64(measuredSeeds) {
				single, faces := cityBuildingFaces(family, number, detail)
				alone += single
				total += faces
				most = max(most, faces)
			}
			t.Logf("%s %s: mean %d faces its own, %d shown, most %d",
				family, detail, alone/measuredSeeds, total/measuredSeeds, most)
		}
	}
	night := DefaultClusteredCity()
	night.Night = true
	for _, detail := range cityLevels() {
		measureCity(t, "grid "+detail, cityMaker{DefaultCityGrid()}, detail)
		measureCity(t, "clustered day "+detail,
			clusteredCityMaker{DefaultClusteredCity()}, detail)
		measureCity(t, "clustered night "+detail,
			clusteredCityMaker{night}, detail)
	}
}

// cityLevelMaker is a city recipe that makes itself at a level.
type cityLevelMaker interface {
	madeAt(seed, string) (good, error)
}

// measureCity makes one city and logs its census and cost.
func measureCity(t *testing.T, name string, row cityLevelMaker,
	detail string) {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	made, err := row.madeAt(Sow(31), detail)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	took := time.Since(started)
	runtime.ReadMemStats(&after)
	counted := census(made)
	panes := 0
	for _, role := range []slot{cityGlass, cityWarmWindow, cityCoolWindow} {
		if group := made.groups[role]; group != nil {
			panes += group.FaceCount() / facesPerPane
		}
	}
	t.Logf("%s: %d members (%.0f MB placing them), %d unique meshes, "+
		"%d pooled faces (%.0f MB), %d placed faces, %d panes, "+
		"heap %+.0f MB, %v, hash %s", name, counted.members,
		float64(counted.members*gpuInstanceBytes)/1e6, counted.unique,
		counted.stored, float64(counted.stored*gpuTriangleBytes)/1e6,
		counted.placed, panes,
		(float64(after.HeapAlloc)-float64(before.HeapAlloc))/1e6,
		took.Round(time.Millisecond), cityGeometryHash(made))
	runtime.KeepAlive(made)
}

// gpuInstanceBytes is one placed member: a 112-byte kernel Instance and
// a 68-byte Metal user-ID instance descriptor, before either hierarchy.
const gpuInstanceBytes = 180

// facesPerPane is a window pane's two triangles; quick's glazing boxes
// count as six panes each by the same division.
const facesPerPane = 2
