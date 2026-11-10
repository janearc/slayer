//go:build darwin && arm64

package manufacture

import (
	"bufio"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

// This is an authored CPU catalogue compatibility record for Darwin/ARM64.
// It holds geometry, normals, prototype sharing, poses, physical bounds and
// materials across the assembly refactor. It is not a physical validation
// of plants or an exact-pixel GPU comparison. The original 144 entries and
// their provenance are unchanged; additions have separate files and producer
// records, captured after their independent geometry checks.

func referenceWrite(h hash.Hash, values any) {
	if err := binary.Write(h, binary.LittleEndian, values); err != nil {
		panic(err)
	}
}
func referenceVector(h hash.Hash, point maths.Vec) {
	referenceWrite(h, [3]float64{point.X, point.Y, point.Z})
}
func referenceSpace(h hash.Hash, pose maths.Subspace) {
	referenceVector(h, pose.Origin)
	referenceWrite(h, pose.Scale)
	for _, axis := range []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis} {
		referenceVector(h, pose.Orientation.Out(axis))
	}
}
func referenceGeometry(h hash.Hash, solid shape.Shape, seen map[shape.Shape]int) {
	if index, exists := seen[solid]; exists {
		referenceWrite(h, int64(index))
		return
	}
	index := len(seen)
	seen[solid] = index
	referenceWrite(h, int64(-1-index))
	switch made := solid.(type) {
	case *mesh.Mesh:
		referenceWrite(h, byte(1))
		referenceWrite(h, int64(made.FaceCount()))
		for i := range made.FaceCount() {
			a, b, c := made.Corners(i)
			referenceVector(h, a)
			referenceVector(h, b)
			referenceVector(h, c)
			normals, smooth := made.CornerNormals(i)
			referenceWrite(h, smooth)
			if smooth {
				for _, normal := range normals {
					referenceVector(h, normal)
				}
			}
		}
	case *shape.Mesh:
		referenceWrite(h, byte(2))
		referenceWrite(h, int64(len(made.Faces)))
		for _, face := range made.Faces {
			for _, corner := range face {
				referenceVector(h, made.Vertices[corner])
			}
		}
		referenceWrite(h, made.Smooth)
		referenceWrite(h, int64(len(made.Normals)))
		for _, normal := range made.Normals {
			referenceVector(h, normal)
		}
	case *shape.MeshGroup:
		members := made.Members()
		referenceWrite(h, byte(3))
		referenceWrite(h, int64(len(members)))
		for _, member := range members {
			referenceSpace(h, member.Space)
			referenceGeometry(h, member.Mesh, seen)
		}
	default:
		panic(fmt.Sprintf("unrecorded manufactured shape %T", solid))
	}
}

//go:embed testdata/geometry-records.yaml
var geometryCountsRaw []byte

// referenceReason is defined by the four fields before a refusal's reason.
const referenceReason = 4

// referenceEpoch is defined as the stationary specimen's zero time.
const referenceEpoch = 0

// referenceAbsent and referencePresent are defined material record tags.
const referenceAbsent, referencePresent = 0, 1

// referenceMaterialFields is defined by RGB, glow RGB, gloss and index.
const referenceMaterialFields = 8

func TestAuthoredCatalogueMatchesItsGeometryRecords(t *testing.T) {
	file, err := os.Open("testdata/geometry-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	additions, err := os.Open("testdata/spacecraft-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer additions.Close()
	ground, err := os.Open("testdata/ground-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer ground.Close()
	survey, err := os.Open("testdata/survey-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer survey.Close()
	large, err := os.Open("testdata/large-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer large.Close()
	civic, err := os.Open("testdata/civic-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer civic.Close()
	personal, err := os.Open("testdata/personal-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer personal.Close()
	yard, err := os.Open("testdata/yard-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer yard.Close()
	summit, err := os.Open("testdata/summit-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer summit.Close()
	launcher, err := os.Open("testdata/launcher-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer launcher.Close()
	vegetation, err := os.Open("testdata/vegetation-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer vegetation.Close()
	names := map[string]bool{}
	seen := map[string]bool{}
	scan := bufio.NewScanner(io.MultiReader(file, additions, ground, survey, large, civic,
		personal, yard, summit, launcher, vegetation))
	cases, geometries, refusals := 0, 0, 0
	for scan.Scan() {
		row := strings.Fields(scan.Text())
		if len(row) < 4 {
			t.Fatalf("invalid geometry reference row: %q", scan.Text())
		}
		name, seedText, level := row[0], row[1], row[2]
		key := strings.Join(row[:3], " ")
		if seen[key] {
			t.Fatalf("duplicate geometry reference %s", key)
		}
		seen[key] = true
		names[name] = true
		cases++
		seed, err := strconv.ParseUint(seedText, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		expected := row[3]
		if expected == "REFUSED" {
			refusals++
		} else {
			geometries++
		}
		t.Run(name+"_"+seedText+"_"+level, func(t *testing.T) {
			referenceSpecimen(t, name, seed, level, expected,
				strings.Join(row[referenceReason:], " "))
		})
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	referencePopulation(t, names, cases, geometries, refusals)
}

// referencePopulation refuses incomplete records or uncatalogued kinds.
func referencePopulation(t *testing.T, names map[string]bool,
	cases, geometries, refusals int,
) {
	t.Helper()
	var expected struct{ Cases, Geometries, Refusals int }
	if err := yamlfile.Read("testdata/geometry-records.yaml", geometryCountsRaw, &expected); err != nil {
		t.Fatal(err)
	}
	if cases != expected.Cases || geometries != expected.Geometries || refusals != expected.Refusals {
		t.Fatalf("incomplete reference: %d cases, %d geometries, %d refusals", cases, geometries, refusals)
	}
	actual := KindNames()
	if len(actual) != len(names) {
		t.Fatal("catalogue and geometry reference have different kind counts")
	}
	for _, name := range actual {
		if !names[name] {
			t.Fatalf("kind %s has no reference", name)
		}
	}
}

// referenceSpecimen checks one authored shape and its material record.
func referenceSpecimen(t *testing.T, name string, seed uint64,
	level, expected, reason string,
) {
	t.Helper()
	kind, err := KindNamed(name)
	if err != nil {
		t.Fatal(err)
	}
	made, err := MakeWithComplexity(kind, Sow(seed), level)
	if expected == "REFUSED" {
		want := reason
		if err == nil || err.Error() != want {
			t.Fatalf("unsupported level: got %v, want %s", err, want)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	referenceFingerprint(t, made, expected)
}

// referenceFingerprint holds the authored geometry and display material bytes.
func referenceFingerprint(t *testing.T, made good, expected string) {
	t.Helper()
	// Preserve the original record's valid zero-value placement exactly.
	var placement maths.Subspace
	things, err := made.Things(placement)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := sha256.New()
	bounds := made.Bounds()
	referenceVector(fingerprint, bounds.Min)
	referenceVector(fingerprint, bounds.Max)
	count := int64(len(things))
	referenceWrite(fingerprint, count)
	shared := map[shape.Shape]int{}
	for _, thing := range things {
		referenceSpace(fingerprint, thing.At(referenceEpoch))
		referenceGeometry(fingerprint, thing.Shape, shared)
		material := thing.Material
		if material == nil {
			referenceWrite(fingerprint, byte(referenceAbsent))
			continue
		}
		referenceWrite(fingerprint, byte(referencePresent))
		nameLength := int64(len(material.Name))
		referenceWrite(fingerprint, nameLength)
		if _, err := fingerprint.Write([]byte(material.Name)); err != nil {
			t.Fatal(err)
		}
		values := [referenceMaterialFields]float64{
			material.Colour.R, material.Colour.G, material.Colour.B,
			material.Glow.R, material.Glow.G, material.Glow.B,
			material.Gloss, material.Index}
		referenceWrite(fingerprint, values)
	}
	if got := fmt.Sprintf("%x", fingerprint.Sum(nil)); got != expected {
		t.Fatalf("authored geometry changed: %s, recorded %s", got, expected)
	}
}
