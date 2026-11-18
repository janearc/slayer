package door

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/job"
)

func TestImportReceiptUsesFrozenNestedSourceClosure(t *testing.T) {
	directory := t.TempDir()
	assetDir := filepath.Join(directory, "assets")
	if err := os.MkdirAll(assetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	assetDir, err := filepath.EvalSymlinks(assetDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"two-metre-human.gltf", "two-metre-human.bin"} {
		data, err := os.ReadFile(filepath.Join("..", "oakland", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(assetDir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gltfPath := filepath.Join(assetDir, "two-metre-human.gltf")
	gltf, err := os.ReadFile(gltfPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yamlfile.Read(gltfPath, gltf, &document); err != nil {
		t.Fatal(err)
	}
	for _, rawNode := range document["nodes"].([]any) {
		node := rawNode.(map[string]any)
		translation := node["translation"].([]any)
		translation[1] = translation[1].(float64) + 7
	}
	gltf, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gltfPath, gltf, 0o600); err != nil {
		t.Fatal(err)
	}
	setText := `scene:
  world:
    materials:
      matte:
        colour: {srgb_linear: [0.3, 0.4, 0.5]}
        source: eye
    imports:
      human:
        file: assets/two-metre-human.gltf
        height: {value: 1, unit: m}
        at: [1, 2, 3]
        materials: {material/0/human-fixture-slot: matte}
        source: "Self-authored CC0 mannequin"
        license: CC0-1.0
`
	setPath := filepath.Join(directory, "set.yaml")
	if err := os.WriteFile(setPath, []byte(setText), 0o600); err != nil {
		t.Fatal(err)
	}
	jobPath := filepath.Join(directory, "job.yaml")
	jobText := `"$schema": https://github.com/janearc/slayer/schemas/job-v1.json
scene: {set: set.yaml}
film: {width: 32, height: 24, rays: 1}
outputs: {picture: portrait.png}
`
	if err := os.WriteFile(jobPath, []byte(jobText), 0o600); err != nil {
		t.Fatal(err)
	}
	wanted, err := job.Read(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	read, has := wanted.ImportedSet()
	if !has {
		t.Fatal("job did not freeze its imported set")
	}
	world, has := read.World()
	if !has {
		t.Fatal("frozen set has no world")
	}
	stillPath, err := importReceiptPath(wanted, places{picture: filepath.Join(directory, "portrait.png")})
	if err != nil || stillPath != filepath.Join(directory, "portrait.png.imports.json") {
		t.Fatalf("still receipt path = %q, %v", stillPath, err)
	}
	filmText := bytes.ReplaceAll([]byte(jobText), []byte("picture: portrait.png"), []byte("frames: frames"))
	filmBytes := append(filmText, []byte("fps: 24\nduration: {value: 1, unit: s}\n")...)
	if err := os.WriteFile(jobPath, filmBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	film, err := job.Read(jobPath)
	if err != nil {
		t.Fatal(err)
	}
	persistent, err := importReceiptPath(film, places{frames: filepath.Join(directory, "frames")})
	if err != nil || persistent != filepath.Join(directory, "frames", "imports.json") {
		t.Fatalf("persistent receipt path = %q, %v", persistent, err)
	}
	working, err := importReceiptPath(film, places{work: true, movie: filepath.Join(directory, "movie.mp4")})
	if err != nil || working != filepath.Join(directory, "movie.mp4.imports.json") {
		t.Fatalf("working receipt path = %q, %v", working, err)
	}
	canonical := bytes.Clone(wanted.Canonical())
	bufferPath := filepath.Join(assetDir, "two-metre-human.bin")
	buffer, err := os.ReadFile(bufferPath)
	if err != nil {
		t.Fatal(err)
	}
	// The independent buffer digest must be retained without opening its
	// source path again while the receipt is made.
	bufferHash := sourceHash(buffer)
	for _, path := range []string{bufferPath, filepath.Join(assetDir, "two-metre-human.gltf"), setPath} {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
	picture := filepath.Join(directory, "portrait.png")
	var said bytes.Buffer
	run := runner{wanted: wanted, env: Env{build: stamp{build: "fixture-build"}},
		at: places{picture: picture}, out: &said}
	if err := run.importReceipt(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(said.Bytes(), []byte("import human: CC0-1.0; receipt "+picture+".imports.json")) {
		t.Fatalf("receipt visitor line = %q", said.String())
	}
	raw, err := os.ReadFile(picture + ".imports.json")
	if err != nil {
		t.Fatal(err)
	}
	made, err := makeImportReceipt(wanted, stamp{build: "fixture-build"}, world)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := yamlfile.Read("receipt", raw, &decoded); err != nil {
		t.Fatal(err)
	}
	var expectedCanonical map[string]any
	if err := yamlfile.Read("canonical job", canonical, &expectedCanonical); err != nil {
		t.Fatal(err)
	}
	decodedJob, ok := decoded["job"].(map[string]any)
	if !bytes.Equal(wanted.Canonical(), canonical) || !bytes.Equal(made.Job.Canonical, canonical) ||
		!ok || !reflect.DeepEqual(decodedJob["canonical"], expectedCanonical) {
		t.Fatal("receipt did not preserve the exact frozen canonical job")
	}
	if fmt.Sprint(decoded["receiptVersion"]) != "1" {
		t.Fatalf("serialized receipt version = %#v", decoded["receiptVersion"])
	}
	if len(made.Imports) != 1 {
		t.Fatalf("receipt imports = %d", len(made.Imports))
	}
	decodedImports, ok := decoded["imports"].([]any)
	if !ok || len(decodedImports) != 1 {
		t.Fatalf("serialized imports = %#v", decoded["imports"])
	}
	decodedImport, ok := decodedImports[0].(map[string]any)
	if !ok || decodedImport["license"] != "CC0-1.0" || decodedImport["source"] != "Self-authored CC0 mannequin" {
		t.Fatalf("serialized import provenance = %#v", decodedImports[0])
	}
	serializedNormalization, ok := decodedImport["normalization"].(map[string]any)
	if !ok || fmt.Sprint(serializedNormalization["scaleFactor"]) != "0.5" ||
		fmt.Sprint(serializedNormalization["verticalOffsetYMetres"]) != "-3.5" {
		t.Fatalf("serialized normalization = %#v", decodedImport["normalization"])
	}
	part := made.Imports[0]
	if part.File != filepath.Join(assetDir, "two-metre-human.gltf") ||
		part.Source != "Self-authored CC0 mannequin" || part.License != "CC0-1.0" ||
		part.RequestedHeightMetres != 1 || part.DecoderOptionsKey == "" {
		t.Fatalf("receipt provenance = %#v", part)
	}
	if part.Normalization == nil || part.Normalization.SourceBoundsYMetres != [2]float64{7, 9} ||
		part.Normalization.TargetHeight != 1 || part.Normalization.HeightUnit != "m" ||
		part.Normalization.ScaleFactor != 0.5 || part.Normalization.VerticalOffset != -3.5 {
		t.Fatalf("receipt normalization = %#v", part.Normalization)
	}
	if len(part.SourceClosure) != 2 {
		t.Fatalf("closure = %#v", part.SourceClosure)
	}
	foundBuffer := false
	for _, source := range part.SourceClosure {
		if source.Path == "assets/two-metre-human.bin" {
			foundBuffer = true
			if source.File != bufferPath || source.SHA256 != bufferHash || source.Bytes != int64(len(buffer)) {
				t.Fatalf("buffer source receipt = %#v", source)
			}
		}
	}
	if !foundBuffer {
		t.Fatal("receipt omitted nested external buffer")
	}
	if len(part.NativeBindings) != 1 || part.NativeBindings[0].Definition.Source != "eye" ||
		part.NativeBindings[0].Definition.Reflectance == nil ||
		part.NativeBindings[0].Definition.Reflectance.LinearRGB == nil {
		t.Fatalf("native material provenance = %#v", part.NativeBindings)
	}
	if made.Build.Stamp != "fixture-build " || len(made.Imports[0].SourceClosure) != 2 {
		t.Fatalf("serialized receipt lost source/build data: %#v", made)
	}
}

func sourceHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
