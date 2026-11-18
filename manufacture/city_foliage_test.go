package manufacture

import (
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// Count actual native plant representations before choosing city preparation.
func TestCityFoliagePrototypeCosts(t *testing.T) {
	row, err := KindNamed("valley-oak")
	if err != nil {
		t.Fatal(err)
	}
	for prototype := range 4 {
		own := citySeed(uint64(31) ^ uint64(prototype) ^ 0x54524545)
		for _, detail := range []string{"hull", "clumps", "quick"} {
			made, err := MakeWithComplexity(row, Sow(own), detail)
			if err != nil {
				t.Fatal(err)
			}
			things, err := made.Things(maths.Local())
			if err != nil {
				t.Fatal(err)
			}
			stored, logical, members := 0, 0, 0
			for _, part := range things {
				if group, ok := part.Shape.(*shape.MeshGroup); ok {
					stored += group.StoredFaces()
					logical += group.FaceCount()
					members += len(group.Members())
				} else {
					count := part.Shape.(interface{ FaceCount() int }).FaceCount()
					stored += count
					logical += count
					members++
				}
			}
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			t.Logf("prototype %d %s: stored %d, expanded %d, members %d, heap %.1f MiB", prototype, detail, stored, logical, members, float64(memory.HeapAlloc)/(1<<20))
		}
	}
}

func TestCityFoliageKeepsPlantingAndSharesRealLeaves(t *testing.T) {
	from := Sow(31)
	spec := greenClusteredCity(false)
	spec.Trees.Complexity = "hull"
	before, err := PlanClusteredCity(spec, from)
	if err != nil {
		t.Fatal(err)
	}
	spec.Trees.Complexity = "quick"
	after, err := PlanClusteredCity(spec, from)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Trees(), after.Trees()) {
		t.Fatal("foliage detail moved trees or changed their seeds")
	}
	prototypes, err := treePrototypesAt(*spec.Trees, from, "quick")
	if err != nil {
		t.Fatal(err)
	}
	palette := looks{}
	prepared, err := prepareCityFoliage(prototypes, palette)
	if err != nil {
		t.Fatal(err)
	}
	unique := map[shape.Shape]bool{}
	cells, faces := 0, 0
	for _, parts := range prepared {
		for _, part := range parts {
			for _, member := range part.members {
				if _, nested := member.Mesh.(*shape.MeshGroup); nested {
					t.Fatal("nested native groups")
				}
				cells++
				if !unique[member.Mesh] {
					unique[member.Mesh] = true
					faces += member.Mesh.(interface{ FaceCount() int }).FaceCount()
				}
			}
		}
	}
	if cells < 1000 || faces < 60000 {
		t.Fatal("quick foliage was replaced with whole-crown proxies", cells, faces)
	}
	started := time.Now()
	made, err := after.treeGroups(good{dressed: looks{}, groups: map[slot]*shape.MeshGroup{}, groupParts: map[slot]int{}}, *spec.Trees, from)
	if err != nil {
		t.Fatal(err)
	}
	members, stored := 0, 0
	for _, group := range made.groups {
		members += len(group.Members())
		stored += group.StoredFaces()
	}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	t.Logf("%d unchanged trees: %d cell instances, %d unique foliage meshes, %d stored faces; group build %s; heap %.1f MiB", len(after.trees), members, len(unique), stored, time.Since(started), float64(memory.HeapAlloc)/(1<<20))
	if stored != faces {
		t.Fatal("foliage sharing was expanded", stored, faces)
	}
}

// anti-test of TestCityFoliageKeepsPlantingAndSharesRealLeaves
func TestCityFoliageHullIsExplicitAndInvalidDetailIsRefused(t *testing.T) {
	spec := DefaultCityTrees()
	spec.Complexity = "not-foliage"
	if err := spec.check(); err == nil {
		t.Fatal("unknown foliage detail admitted")
	}
	spec.Complexity = "hull"
	prototypes, err := treePrototypesAt(spec, Sow(31), spec.complexity())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareCityFoliage(prototypes, looks{})
	if err != nil {
		t.Fatal(err)
	}
	for _, parts := range prepared {
		if len(parts) != 2 {
			t.Fatal("hull materials changed")
		}
		for _, part := range parts {
			if len(part.members) != 1 {
				t.Fatal("explicit hull unexpectedly grew leaf cells")
			}
		}
	}
}
