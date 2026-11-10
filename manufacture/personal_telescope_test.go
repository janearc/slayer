package manufacture_test

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed testdata/personal-telescopes.yaml
var personalContractRaw []byte

type personalContract struct {
	Seed          uint64
	Tolerance     float64
	MaximumFaces  int `yaml:"maximum-faces"`
	Axis, Tangent maths.Vec
	Profiles      []personalProfile
}

type personalProfile struct {
	Kind, Role               string
	Roles                    []string
	Aperture, Near, Far      float64
	Minimum, Maximum, Centre maths.Vec
}

type personalProbe struct {
	profile  personalProfile
	contract personalContract
}

func TestPersonalTelescopesKeepTheirRequestedAperturesAndClosedParts(
	test *testing.T,
) {
	var data personalContract
	if err := yamlfile.Read("personal-telescopes.yaml", personalContractRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	for _, profile := range data.Profiles {
		probe := personalProbe{profile: profile, contract: data}
		test.Run(profile.Kind, probe.check)
	}
}

// check holds physical envelopes and optical faces at every detail.
func (probe personalProbe) check(test *testing.T) {
	test.Helper()
	row, err := manufacture.KindNamed(probe.profile.Kind)
	if err != nil {
		test.Fatal(err)
	}
	for _, detail := range manufacture.Complexities() {
		object, err := manufacture.MakeWithComplexity(row,
			manufacture.Sow(probe.contract.Seed), detail)
		if err != nil {
			test.Fatal(detail, err)
		}
		bounds := object.Bounds()
		if bounds.Min.Sub(probe.profile.Minimum).Len() >
			probe.contract.Tolerance ||
			bounds.Max.Sub(probe.profile.Maximum).Len() >
				probe.contract.Tolerance {
			test.Fatal("instrument envelope changed", detail, bounds)
		}
		parts, err := object.Things(maths.Local())
		if err != nil {
			test.Fatal(detail, err)
		}
		limits := meshContract{Tolerance: probe.contract.Tolerance,
			MaximumFaces: probe.contract.MaximumFaces}
		limits.assertMeshes(test, parts, bounds)
		for _, role := range probe.profile.Roles {
			probe.checkRole(test, parts, role)
		}
	}
}

// checkRole measures the objective or primary independently of its maker.
func (probe personalProbe) checkRole(test *testing.T,
	parts []thing.Thing, role string,
) {
	test.Helper()
	found := false
	for _, part := range parts {
		if part.Material.Name != role {
			continue
		}
		if found {
			test.Fatal("duplicate optical role", role)
		}
		found = true
		if role == probe.profile.Role {
			probe.checkAperture(test, part.Shape.(*mesh.Mesh))
		}
	}
	if !found {
		test.Fatal("missing optical role", role)
	}
}

// checkAperture uses the expected physical plane and unit axes from the fixture.
func (probe personalProbe) checkAperture(test *testing.T, polygon *mesh.Mesh) {
	test.Helper()
	radius := float64(spacecraftNear)
	near := math.Inf(spacecraftPositiveSign)
	far := math.Inf(-spacecraftPositiveSign)
	for _, point := range polygon.Points() {
		offset := point.Sub(probe.profile.Centre)
		along := offset.Dot(probe.contract.Axis)
		near, far = min(near, along), max(far, along)
		across := offset.Dot(probe.contract.Tangent)
		radius = max(radius, math.Hypot(offset.X, across))
	}
	error := math.Abs(radius + radius - probe.profile.Aperture)
	if error > probe.contract.Tolerance ||
		math.Abs(near-probe.profile.Near) > probe.contract.Tolerance ||
		math.Abs(far-probe.profile.Far) > probe.contract.Tolerance {
		test.Fatal("wrong requested aperture or optical plane",
			probe.profile.Kind, radius, near, far)
	}
}
