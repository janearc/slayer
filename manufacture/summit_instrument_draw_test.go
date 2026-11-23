//go:build draw

package manufacture_test

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed testdata/summit-instruments.yaml
var summitContractRaw []byte

func TestSummitFamiliesKeepTheirSourceAperturesAndClosedGeometry(test *testing.T) {
	var data summitContract
	if err := yamlfile.Read("summit-instruments.yaml", summitContractRaw,
		&data); err != nil {
		test.Fatal(err)
	}
	for _, profile := range data.Profiles {
		probe := summitInstrument{summitProbe: summitProbe{contract: data},
			profile: profile}
		test.Run(profile.Kind, probe.check)
	}
}

// summitInstrument is a probe of the one profile a table row declares.
type summitInstrument struct {
	summitProbe
	profile summitProfile
}

type summitSpan struct{ radius, near, far float64 }

// check keeps the physical site, source aperture and array members at all details.
func (probe summitInstrument) check(test *testing.T) {
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
		if bounds.Min != probe.profile.Minimum || bounds.Max != probe.profile.Maximum {
			test.Fatal("summit physical envelope changed", detail, bounds)
		}
		parts, err := object.Things(maths.Local())
		if err != nil {
			test.Fatal(detail, err)
		}
		probe.parts(test, parts, bounds)
	}
}

// parts checks actual prototype geometry as well as its placed points.
func (probe summitInstrument) parts(test *testing.T, parts []thing.Thing,
	bounds maths.Box,
) {
	test.Helper()
	spans := make([]summitSpan, len(probe.profile.Centres))
	for index := range spans {
		spans[index].near = math.Inf(spacecraftPositiveSign)
		spans[index].far = math.Inf(-spacecraftPositiveSign)
	}
	for _, part := range parts {
		probe.part(test, part, bounds, spans)
	}
	if len(spans) != probe.profile.Copies {
		test.Fatal("wrong source inventory")
	}
	for _, span := range spans {
		probe.span(test, span)
	}
}

// part handles the ordinary single instrument and the shared array hierarchy.
func (probe summitInstrument) part(test *testing.T, part thing.Thing,
	bounds maths.Box, spans []summitSpan,
) {
	test.Helper()
	switch solid := part.Shape.(type) {
	case *mesh.Mesh:
		if probe.profile.Group {
			test.Fatal("array prototype was flattened")
		}
		probe.polygon(test, solid, maths.Local(), bounds)
		if part.Material.Name == probe.profile.Role {
			probe.measure(solid, maths.Local(), spans)
		}
	case *shape.MeshGroup:
		probe.group(test, solid, part.Material.Name, bounds, spans)
	default:
		test.Fatal("unexpected summit geometry", part.Shape)
	}
}

// group holds real pointer sharing and every member's transformed geometry.
func (probe summitInstrument) group(test *testing.T, group *shape.MeshGroup,
	role string, bounds maths.Box, spans []summitSpan,
) {
	test.Helper()
	members := group.Members()
	if !probe.profile.Group || len(members) != probe.profile.Copies {
		test.Fatal("wrong array member count", len(members))
	}
	first := members[spacecraftNear].Mesh
	for _, member := range members {
		if member.Mesh != first {
			test.Fatal("array copied its prototype")
		}
		polygon := member.Mesh.(*mesh.Mesh)
		probe.polygon(test, polygon, member.Space, bounds)
		if role == probe.profile.Role {
			probe.measure(polygon, member.Space, spans)
		}
	}
}

// measure assigns points to the independently declared pupils, not maker poses.
func (probe summitInstrument) measure(polygon *mesh.Mesh, pose maths.Subspace,
	spans []summitSpan,
) {
	for _, point := range polygon.Points() {
		point = pose.Out(point)
		index := probe.nearest(point)
		offset := point.Sub(probe.profile.Centres[index])
		along := offset.Dot(probe.contract.Axis)
		spans[index].near = min(spans[index].near, along)
		spans[index].far = max(spans[index].far, along)
		across := offset.Dot(probe.contract.Tangent)
		spans[index].radius = max(spans[index].radius, math.Hypot(offset.X, across))
	}
}

// nearest separates the twin pupils and the distant shared antennas.
func (probe summitInstrument) nearest(point maths.Vec) int {
	index, closest := spacecraftNear, math.Inf(spacecraftPositiveSign)
	for candidate, centre := range probe.profile.Centres {
		if distance := point.Sub(centre).Len(); distance < closest {
			index, closest = candidate, distance
		}
	}
	return index
}

// span holds source aperture and authored optical plane, including thickness.
func (probe summitInstrument) span(test *testing.T, span summitSpan) {
	test.Helper()
	if !(math.Abs(span.radius+span.radius-probe.profile.Aperture) <= probe.contract.Tolerance) ||
		!(math.Abs(span.near-probe.profile.Near) <= probe.contract.Tolerance) ||
		!(math.Abs(span.far-probe.profile.Far) <= probe.contract.Tolerance) {
		test.Fatal("wrong summit aperture or optical plane", probe.profile.Kind, span)
	}
}
