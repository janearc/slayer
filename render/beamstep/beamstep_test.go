package beamstep_test

import (
	_ "embed"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/beamstep"
	"github.com/janearc/slayer/scene/thing"
)

// oneCell is defined as an unchanged original march cell.
const oneCell = 1

//go:embed testdata/reference.yaml
var referenceRaw []byte

type cell struct {
	Enter  float64 `json:"enter"`
	Leave  float64 `json:"leave"`
	Splits int     `json:"splits"`
}

type reference struct {
	Source     emit.Lit `json:"source"`
	Radius     float64  `json:"radius"`
	Reach      float64  `json:"reach"`
	Divergence float64  `json:"divergence"`
	Queries    []struct {
		Ray   maths.Ray `json:"ray"`
		Far   float64   `json:"far"`
		Cells []cell    `json:"cells"`
	} `json:"queries"`
	Second    emit.Lit  `json:"second-source" yaml:"second-source"`
	MergedRay maths.Ray `json:"merged-ray" yaml:"merged-ray"`
	MergedFar float64   `json:"merged-far" yaml:"merged-far"`
	MergedGap cell      `json:"merged-gap" yaml:"merged-gap"`
}

func readReference(test *testing.T) reference {
	test.Helper()
	var fixture reference
	if err := yamlfile.Read("testdata/reference.yaml", referenceRaw, &fixture); err != nil {
		test.Fatal(err)
	}
	return fixture
}

func (fixture reference) beam(test *testing.T, source emit.Lit) *emit.Beam {
	test.Helper()
	beam, err := emit.NewBeam(source, measure.Metres(fixture.Radius), measure.Metres(fixture.Reach), measure.Degrees(fixture.Divergence))
	if err != nil {
		test.Fatal(err)
	}
	return beam
}

func TestBeamRefinementFollowsThisRaysBoundsAndOriginalCells(test *testing.T) {
	fixture := readReference(test)
	beam := fixture.beam(test, fixture.Source)
	plan := beamstep.Of([]thing.Lamp{beam})
	for _, query := range fixture.Queries {
		refine := plan.Ray(query.Ray, query.Far)
		for _, cell := range query.Cells {
			if got := refine.Splits(cell.Enter, cell.Leave); got != cell.Splits {
				test.Fatal("wrong conservative sampling", query.Ray, query.Far, cell, got)
			}
		}
	}
}

func TestBeamRefinementLeavesOrdinaryAndMissingLampsAlone(test *testing.T) {
	fixture := readReference(test)
	var absent *emit.Beam
	plan := beamstep.Of([]thing.Lamp{fixture.Source, absent})
	var missing *beamstep.Beams
	for _, query := range fixture.Queries {
		ordinary := plan.Ray(query.Ray, query.Far)
		empty := missing.Ray(query.Ray, query.Far)
		for _, cell := range query.Cells {
			if ordinary.Splits(cell.Enter, cell.Leave) != oneCell || empty.Splits(cell.Enter, cell.Leave) != oneCell {
				test.Fatal("ordinary lamp changed volume sampling")
			}
		}
	}
}

func TestBeamRefinementConservativelyMergesSeparatedIntervals(test *testing.T) {
	fixture := readReference(test)
	first := fixture.beam(test, fixture.Source)
	second := fixture.beam(test, fixture.Second)
	plan := beamstep.Of([]thing.Lamp{first, second})
	refine := plan.Ray(fixture.MergedRay, fixture.MergedFar)
	gap := fixture.MergedGap
	if refine.Splits(gap.Enter, gap.Leave) != gap.Splits {
		test.Fatal("merged sampling omitted the gap between beams")
	}
}
