package oakland_test

import (
	"crypto/sha256"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oakland"
	"github.com/janearc/slayer/scene/thing"
)

const humanSource = "testdata/two-metre-human.gltf"

func policyFor(path string) oakland.Policy {
	return oakland.Policy{Root: path, TargetHeight: 2, HeightUnit: "m",
		Limits: oakland.DefaultLimits()}
}

func TestReadHumanKeepsClosureAndSharedGeometry(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	asset, err := oakland.Read(humanSource, policyFor(root))
	if err != nil {
		t.Fatal(err)
	}
	closure := asset.SourceClosure()
	if len(closure) != 2 || closure[0].Path() != "testdata/two-metre-human.bin" ||
		closure[1].Path() != humanSource {
		t.Fatalf("source closure = %#v", closure)
	}
	for _, source := range closure {
		relative := filepath.FromSlash(source.Path())
		data, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatal(err)
		}
		if source.Size() != int64(len(data)) || source.SHA256() != sha256.Sum256(data) {
			t.Fatalf("source %q does not describe the bytes decoded", source.Path())
		}
	}
	instances := asset.Instances()
	if len(instances) != 8 {
		t.Fatalf("human has %d instances, want 8", len(instances))
	}
	if instances[4].Geometry() != instances[5].Geometry() {
		t.Fatal("same-size arms did not retain a shared immutable mesh")
	}
	if instances[4].Placement().Origin == instances[5].Placement().Origin {
		t.Fatal("shared arm mesh lost its distinct node placements")
	}
	if instances[0].Geometry() == nil || instances[0].Geometry().FaceCount() != 12 {
		t.Fatalf("human geometry = %#v", instances[0].Geometry())
	}
	if asset.OptionsKey() == ([32]byte{}) {
		t.Fatal("asset options key is empty")
	}
	if slots := asset.MaterialSlots(); len(slots) != 1 || slots[0] != "material/0/human-fixture-slot" {
		t.Fatalf("material slots = %#v", slots)
	}
}

func TestInstantiateBindsExplicitNativeMaterials(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	asset, err := oakland.Read(humanSource, policyFor(root))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := asset.Instantiate(nil); err == nil {
		t.Fatal("missing material binding was accepted")
	}
	material := thing.Diffuse("human", maths.Light{R: 0.5, G: 0.4, B: 0.3})
	things, err := asset.Instantiate(map[string]*thing.Material{
		"material/0/human-fixture-slot": material,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(things) != 8 {
		t.Fatalf("instantiated %d things, want 8", len(things))
	}
	if _, err := asset.Instantiate(map[string]*thing.Material{
		"material/0/human-fixture-slot": material,
		"unused":                        material,
	}); err == nil {
		t.Fatal("extra material binding was accepted")
	}
	for i, made := range things {
		if made.Material != material {
			t.Fatalf("thing %d did not use its explicit native binding", i)
		}
	}
}

func TestOptionsKeyNamesNormalizationIntent(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	metres := policyFor(root)
	feet := metres
	feet.HeightUnit = "ft"
	otherHeight := metres
	otherHeight.TargetHeight = 1
	first, err := oakland.Read(humanSource, metres)
	if err != nil {
		t.Fatal(err)
	}
	second, err := oakland.Read(humanSource, feet)
	if err != nil {
		t.Fatal(err)
	}
	third, err := oakland.Read(humanSource, otherHeight)
	if err != nil {
		t.Fatal(err)
	}
	if first.OptionsKey() == second.OptionsKey() || first.OptionsKey() == third.OptionsKey() {
		t.Fatal("asset options key ignored normalization intent")
	}
}

func TestHeightNormalizationGroundsTheLowestPoint(t *testing.T) {
	root := t.TempDir()
	document := humanDocument(t)
	for _, raw := range document["nodes"].([]any) {
		node := raw.(map[string]any)
		translation := node["translation"].([]any)
		translation[1] = translation[1].(float64) + 7
	}
	writeHuman(t, root, document)
	asset, err := oakland.Read("human.gltf", oakland.Policy{Root: root,
		TargetHeight: 1, HeightUnit: "m", Limits: oakland.DefaultLimits()})
	if err != nil {
		t.Fatal(err)
	}
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, instance := range asset.Instances() {
		placement := instance.Placement()
		for _, point := range instance.Geometry().Points() {
			world := placement.Out(point)
			minY = math.Min(minY, world.Y)
			maxY = math.Max(maxY, world.Y)
		}
	}
	if math.Abs(minY) > 1e-12 || math.Abs(maxY-1) > 1e-12 {
		t.Fatalf("normalized vertical bounds = [%g, %g], want [0, 1]", minY, maxY)
	}
	record, has := asset.HeightNormalization()
	sourceMin, sourceMax := record.SourceBoundsY()
	if !has || sourceMin != 7 || sourceMax != 9 || record.ScaleFactor() != 0.5 ||
		record.VerticalOffset() != -3.5 || record.TargetHeight() != 1 || record.HeightUnit() != "m" {
		t.Fatalf("normalization record = %#v, bounds [%g, %g]", record, sourceMin, sourceMax)
	}
}

func TestReadBoundsUniqueBakedAndNormalizedVertexWork(t *testing.T) {
	for _, test := range []struct {
		name   string
		height float64
	}{
		{name: "baked variants"},
		{name: "normalization work", height: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			limits := oakland.DefaultLimits()
			limits.Vertices = 24
			limits.Indices = 36
			unit := ""
			if test.height > 0 {
				unit = "m"
			}
			_, err := oakland.Read(humanSource, oakland.Policy{Root: ".",
				TargetHeight: test.height, HeightUnit: unit, Limits: limits})
			if err == nil {
				t.Fatal("many unique geometry variants exceeded the aggregate work budget")
			}
			want := "baked geometry"
			if test.height > 0 {
				want = "normalization work"
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("refusal = %v, want %q budget", err, want)
			}
		})
	}
}

func TestOptionsKeyNamesChangedSourceClosure(t *testing.T) {
	root := t.TempDir()
	writeHuman(t, root, humanDocument(t))
	policy := oakland.Policy{Root: root, Limits: oakland.DefaultLimits()}
	first, err := oakland.Read("human.gltf", policy)
	if err != nil {
		t.Fatal(err)
	}
	document := humanDocument(t)
	materials := document["materials"].([]any)
	materials[0].(map[string]any)["name"] = "changed-fixture-slot"
	changed, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "human.gltf"), changed, 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := oakland.Read("human.gltf", policy)
	if err != nil {
		t.Fatal(err)
	}
	if first.OptionsKey() == second.OptionsKey() {
		t.Fatal("asset options key ignored changed source closure bytes")
	}
}

func TestReadEmbeddedGLB(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	asset, err := oakland.Read("testdata/embedded-triangle.glb", oakland.Policy{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	instances := asset.Instances()
	if len(instances) != 1 || instances[0].Geometry() == nil ||
		instances[0].Geometry().FaceCount() != 1 {
		t.Fatalf("embedded GLB instances = %#v", instances)
	}
	closure := asset.SourceClosure()
	if len(closure) != 1 || closure[0].Path() != "testdata/embedded-triangle.glb" {
		t.Fatalf("GLB source closure = %#v", closure)
	}
}

func TestReadRefusesUnsupportedAndMalformedDocuments(t *testing.T) {
	root := t.TempDir()
	raw, err := os.ReadFile(humanSource)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yamlfile.Read(humanSource, raw, &document); err != nil {
		t.Fatal(err)
	}
	document["extensionsRequired"] = []any{"KHR_draco_mesh_compression"}
	unsupported, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readTemporary(t, root, "unsupported.gltf", unsupported); err == nil {
		t.Fatal("requested extension was accepted")
	}
	document["extensionsRequired"] = nil
	delete(document, "extensionsRequired")
	meshes := document["meshes"].([]any)
	primitive := meshes[0].(map[string]any)["primitives"].([]any)[0].(map[string]any)
	primitive["mode"] = "triangles"
	malformed, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readTemporary(t, root, "wrong-mode.gltf", malformed); err == nil {
		t.Fatal("wrongly typed primitive mode was accepted")
	}
}

func TestReadRefusesYAMLMasqueradingAsGLTF(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "not-json.gltf"),
		[]byte("asset:\n  version: '2.0'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := oakland.Read("not-json.gltf", oakland.Policy{Root: root}); err == nil {
		t.Fatal("YAML syntax was accepted as glTF JSON")
	}
}

func TestReadRefusesEscapingSymlinkAndBadLimits(t *testing.T) {
	root := t.TempDir()
	data, err := os.ReadFile(humanSource)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "human.gltf"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "human.bin")
	bytes, err := os.ReadFile("testdata/two-metre-human.bin")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "two-metre-human.bin")); err != nil {
		t.Fatal(err)
	}
	if _, err := oakland.Read("human.gltf", oakland.Policy{Root: root}); err == nil {
		t.Fatal("external buffer symlink escape was accepted")
	}
	invalid := oakland.Policy{Root: root, Limits: oakland.Limits{SourceBytes: -1}}
	if _, err := oakland.Read("human.gltf", invalid); err == nil {
		t.Fatal("negative resource limit was accepted")
	}
	tooSmall := oakland.DefaultLimits()
	tooSmall.SourceBytes = 16
	if _, err := oakland.Read("human.gltf", oakland.Policy{Root: root, Limits: tooSmall}); err == nil {
		t.Fatal("source closure larger than its byte bound was accepted")
	}
}

func TestNestedAffineTransformPreservesKnownNormalAndWinding(t *testing.T) {
	root := t.TempDir()
	document := humanDocument(t)
	document["scenes"] = []any{map[string]any{"nodes": []any{0}}}
	document["nodes"] = []any{
		map[string]any{"scale": []any{2, 1, 1}, "children": []any{1}},
		map[string]any{"mesh": 0, "translation": []any{0, 1, 0},
			"rotation": []any{float64(0), float64(0), math.Sin(math.Pi / 8), math.Cos(math.Pi / 8)}},
	}
	writeHuman(t, root, document)
	asset, err := oakland.Read("human.gltf", oakland.Policy{Root: root, Limits: oakland.DefaultLimits()})
	if err != nil {
		t.Fatal(err)
	}
	instances := asset.Instances()
	if len(instances) != 1 {
		t.Fatalf("nested scene instances = %d, want 1", len(instances))
	}
	transformed := instances[0]
	if transformed.Placement().Origin.Sub(maths.V(0, 1, 0)).Len() > 1e-12 {
		t.Fatalf("nested translation = %v", transformed.Placement().Origin)
	}
	// Parent non-uniform scale composed with child rotation creates shear.
	// Source point (1/2, 1/2, 1/2) becomes (0, sqrt(1/2), 1/2),
	// with the child translation retained by the placement.
	wantPoint := maths.V(0, math.Sqrt(0.5), 0.5)
	found := false
	for _, point := range transformed.Geometry().Points() {
		if point.Sub(wantPoint).Len() < 1e-12 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("baked geometry omitted known transformed point %v", wantPoint)
	}
	// An x normal under S(2,1,1)*Rz(45deg) transforms to (.4472,.8944,0).
	normals, smooth := transformed.Geometry().CornerNormals(10)
	if !smooth {
		t.Fatal("authored vertex normals were discarded")
	}
	wantNormal := maths.V(1/math.Sqrt(5), 2/math.Sqrt(5), 0)
	for _, normal := range normals {
		if normal.Sub(wantNormal).Len() > 1e-12 {
			t.Fatalf("transformed normal = %v, want %v", normal, wantNormal)
		}
	}
}

func TestReflectionReversesWindingAndSingularityRefuses(t *testing.T) {
	root := t.TempDir()
	document := humanDocument(t)
	nodes := document["nodes"].([]any)
	nodes[0].(map[string]any)["scale"] = []any{-1, 1, 1}
	document["scenes"] = []any{map[string]any{"nodes": []any{0}}}
	document["nodes"] = []any{nodes[0]}
	writeHuman(t, root, document)
	asset, err := oakland.Read("human.gltf", oakland.Policy{Root: root, Limits: oakland.DefaultLimits()})
	if err != nil {
		t.Fatal(err)
	}
	geometry := asset.Instances()[0].Geometry()
	faces := geometry.Faces()
	if faces[0][1] != 2 || faces[0][2] != 1 {
		t.Fatal("reflection kept the original face winding")
	}
	reflected := humanDocument(t)
	reflectedNodes := reflected["nodes"].([]any)
	reflectedNodes[0].(map[string]any)["scale"] = []any{0, 1, 1}
	reflected["scenes"] = []any{map[string]any{"nodes": []any{0}}}
	reflected["nodes"] = []any{reflectedNodes[0]}
	writeHuman(t, root, reflected)
	if _, err := oakland.Read("human.gltf", oakland.Policy{Root: root, Limits: oakland.DefaultLimits()}); err == nil {
		t.Fatal("singular node transform was accepted")
	}
}

func TestReadRefusesNodeCycle(t *testing.T) {
	root := t.TempDir()
	document := humanDocument(t)
	document["nodes"] = []any{map[string]any{"mesh": 0, "children": []any{0}}}
	document["scenes"] = []any{map[string]any{"nodes": []any{0}}}
	writeHuman(t, root, document)
	if _, err := oakland.Read("human.gltf", oakland.Policy{Root: root, Limits: oakland.DefaultLimits()}); err == nil {
		t.Fatal("node cycle was accepted")
	}
}

func TestMultipleScenesRequireExplicitSelection(t *testing.T) {
	root := t.TempDir()
	document := humanDocument(t)
	document["scenes"] = []any{
		map[string]any{"nodes": []any{0}},
		map[string]any{"nodes": []any{1}},
	}
	writeHuman(t, root, document)
	if _, err := oakland.Read("human.gltf", oakland.Policy{Root: root, Limits: oakland.DefaultLimits()}); err == nil {
		t.Fatal("glTF default scene made an ambiguous scene selection implicit")
	}
	selected := 1
	asset, err := oakland.Read("human.gltf", oakland.Policy{Root: root, Scene: &selected,
		Limits: oakland.DefaultLimits()})
	if err != nil {
		t.Fatal(err)
	}
	if len(asset.Instances()) != 1 {
		instances := len(asset.Instances())
		t.Fatalf("selected scene has %d instances, want one", instances)
	}
}

func TestReadRefusesPathTraversalBufferURI(t *testing.T) {
	root := t.TempDir()
	document := humanDocument(t)
	buffers := document["buffers"].([]any)
	buffers[0].(map[string]any)["uri"] = "../human.bin"
	writeHuman(t, root, document)
	if _, err := oakland.Read("human.gltf", oakland.Policy{Root: root, Limits: oakland.DefaultLimits()}); err == nil {
		t.Fatal("buffer URI path traversal was accepted")
	}
}

func TestReadRefusesOversizedAccessorBeforeAllocation(t *testing.T) {
	root := t.TempDir()
	document := humanDocument(t)
	accessors := document["accessors"].([]any)
	accessors[0].(map[string]any)["count"] = oakland.DefaultLimits().Vertices + 1
	writeHuman(t, root, document)
	if _, err := oakland.Read("human.gltf", oakland.Policy{Root: root, Limits: oakland.DefaultLimits()}); err == nil {
		t.Fatal("accessor above the vertex bound was accepted")
	}
}

func TestReadRefusesInvalidGLBFraming(t *testing.T) {
	root := t.TempDir()
	raw, err := os.ReadFile("testdata/embedded-triangle.glb")
	if err != nil {
		t.Fatal(err)
	}
	raw[8] ^= 0x01
	if err := os.WriteFile(filepath.Join(root, "bad.glb"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := oakland.Read("bad.glb", oakland.Policy{Root: root, Limits: oakland.DefaultLimits()}); err == nil {
		t.Fatal("malformed GLB chunk padding was accepted")
	}
}

func readTemporary(t *testing.T, root, name string, raw []byte) (*oakland.Asset, error) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return oakland.Read(name, oakland.Policy{Root: root})
}

func humanDocument(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(humanSource)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yamlfile.Read(humanSource, raw, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func writeHuman(t *testing.T, root string, document map[string]any) {
	t.Helper()
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	buffer, err := os.ReadFile("testdata/two-metre-human.bin")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "human.gltf"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "two-metre-human.bin"), buffer, 0o600); err != nil {
		t.Fatal(err)
	}
}
