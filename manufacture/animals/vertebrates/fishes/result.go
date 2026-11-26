package fishes

import (
	"math"

	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Group is one school prototype shared by its placements.
type Group struct {
	Mesh     *shape.MeshGroup
	Material *thing.Material
}

// Result holds a single fish's native parts, or ordered school groups.
type Result struct {
	Things    []thing.Thing
	Bounds    maths.Box
	HasBounds bool
	Groups    []Group
}

// Recipe owns the checked species and its requested representation.
type Recipe struct {
	name    string
	profile fishSpecies
	options FishOptions
}

// Build constructs this fish from the caller's repeatable seed.
func (row *Recipe) Build(from branching.Seed) (Result, error) {
	if row.options.School > 0 {
		return row.buildSchool(from)
	}
	body, err := buildFish(fishPlan{name: row.name, species: row.profile, from: from})
	if err != nil {
		return Result{}, err
	}
	if row.options.Bones {
		return body.bonesGood()
	}
	pose := body.stroke(body.plan.figure("amplitude"))
	if row.options.Glide {
		pose = body.stroke(body.plan.figure("glide"))
	}
	return body.fleshGood(pose, true)
}

const (
	picking = 0
	growing = 1
)

type fishRange struct{ least, most float64 }

func (of fishRange) pick(share float64) float64 { return of.least + (of.most-of.least)*share }
func (of fishRange) count(share float64) int {
	wide := of.most - of.least + 1
	return int(of.least + math.Min(math.Floor(share*wide), wide-1))
}
func positive(n float64) bool { return n > 0 && !math.IsInf(n, 0) }
func share(n float64) bool    { return n >= 0 && n <= 1 }
func shares(v []float64) bool { return len(v) == 3 && share(v[0]) && share(v[1]) && share(v[2]) }

// Embedded returns an independent copy of the default species book.
func Embedded() []byte { return append([]byte(nil), fishRaw...) }
