package main

import (
	_ "embed"
	"errors"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

//go:embed testdata/construction.yaml
var constructionRaw []byte

type constructionContract struct {
	Kind, Detail string
	Seed         uint64
	Faces        int
	Budgets      []struct {
		Limit   int
		Allowed bool
	}
	UnknownKind   string    `yaml:"unknown-kind"`
	UnknownDetail string    `yaml:"unknown-detail"`
	BadDirection  maths.Vec `yaml:"bad-direction"`
}

func readConstruction(test *testing.T) constructionContract {
	test.Helper()
	var data constructionContract
	if err := yamlfile.Read("construction.yaml", constructionRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	return data
}

func TestCatalogueHonoursItsFaceLimitIncludingTheExactBoundary(test *testing.T) {
	data := readConstruction(test)
	plan := defaultPlan(test)
	plan.Seed = data.Seed
	for _, budget := range data.Budgets {
		plan.FaceLimit = budget.Limit
		object, err := plan.construct(entry{Kind: data.Kind}, data.Detail)
		if (err == nil) != budget.Allowed ||
			(object != nil) != budget.Allowed {
			test.Fatal("wrong face-budget decision", budget, object, err)
		}
		if object != nil && object.faces != data.Faces {
			test.Fatal("stored-face inventory changed", object.faces, data.Faces)
		}
	}
}

func TestCataloguePreservesUnknownKindAndDetailContext(test *testing.T) {
	data := readConstruction(test)
	plan := defaultPlan(test)
	object, err := plan.construct(entry{Kind: data.UnknownKind}, data.Detail)
	if object != nil || err == nil ||
		!strings.Contains(err.Error(), data.UnknownKind) {
		test.Fatal("lost the unknown kind", object, err)
	}
	object, err = plan.construct(entry{Kind: data.Kind}, data.UnknownDetail)
	if object != nil || err == nil ||
		!strings.Contains(err.Error(), data.UnknownDetail) {
		test.Fatal("lost the unsupported detail", object, err)
	}
}

func TestCataloguePreservesTheGeometryRefusal(test *testing.T) {
	data := readConstruction(test)
	row, err := manufacture.KindNamed(data.Kind)
	if err != nil {
		test.Fatal(err)
	}
	failed, err := manufacture.MakeWithComplexity(row,
		manufacture.Sow(data.Seed), data.UnknownDetail)
	if err == nil {
		test.Fatal("unsupported detail unexpectedly made geometry")
	}
	_, cause := failed.Things(maths.Local())
	if cause == nil {
		test.Fatal("failed construction exposed a usable source")
	}
	plan := defaultPlan(test)
	object, err := plan.compose(entry{Kind: data.Kind}, failed)
	if object != nil || !errors.Is(err, cause) {
		test.Fatal("replaced the geometry's cause", object, err, cause)
	}
}

func TestCataloguePreservesAnInvalidLampDirection(test *testing.T) {
	data := readConstruction(test)
	plan := defaultPlan(test)
	for index := range plan.Sources {
		plan.Sources[index].Toward = data.BadDirection
	}
	object, err := plan.construct(entry{Kind: data.Kind}, data.Detail)
	if object != nil || !errors.Is(err, emit.ErrDistant) {
		test.Fatal("lost invalid illumination", object, err)
	}
}
