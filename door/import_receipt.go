package door

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oakland"
	"github.com/janearc/slayer/set"
)

type receipt struct {
	Version int             `json:"receiptVersion"`
	Job     jobReceipt      `json:"job"`
	Build   buildReceipt    `json:"build"`
	Imports []importReceipt `json:"imports"`
}

type jobReceipt struct {
	File        string            `json:"file,omitempty"`
	Key         string            `json:"key"`
	PictureKey  string            `json:"pictureKey"`
	Canonical   json.RawMessage   `json:"canonical"`
	InputHashes map[string]string `json:"inputHashes"`
	InputFiles  map[string]string `json:"inputFiles"`
}

type buildReceipt struct {
	Stamp    string `json:"stamp"`
	Build    string `json:"build,omitempty"`
	Built    string `json:"built,omitempty"`
	Who      string `json:"who,omitempty"`
	Sessions string `json:"sessions,omitempty"`
}

type importReceipt struct {
	Name                  string                `json:"name"`
	File                  string                `json:"file"`
	Source                string                `json:"source"`
	License               string                `json:"license"`
	SelectedScene         *int                  `json:"selectedScene,omitempty"`
	RequestedHeightMetres float64               `json:"requestedHeightMetres"`
	Normalization         *normalizationReceipt `json:"normalization,omitempty"`
	DecoderOptionsKey     string                `json:"decoderOptionsKey"`
	SourceClosure         []sourceReceipt       `json:"sourceClosure"`
	NativeBindings        []bindingReceipt      `json:"nativeBindings"`
	AssetInstances        []instanceReceipt     `json:"assetInstances"`
	NativePlacement       transformReceipt      `json:"nativePlacement"`
}

type normalizationReceipt struct {
	SourceBoundsYMetres [2]float64 `json:"sourceBoundsYMetres"`
	TargetHeight        float64    `json:"targetHeight"`
	HeightUnit          string     `json:"heightUnit"`
	ScaleFactor         float64    `json:"scaleFactor"`
	VerticalOffset      float64    `json:"verticalOffsetYMetres"`
}

type sourceReceipt struct {
	Path   string `json:"path"`
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type bindingReceipt struct {
	Slot       string          `json:"slot"`
	Material   string          `json:"material"`
	Definition materialReceipt `json:"definition"`
}

type materialReceipt struct {
	Name          string            `json:"name"`
	Source        string            `json:"source"`
	Reflectance   *colourReceipt    `json:"reflectance,omitempty"`
	Emission      *colourReceipt    `json:"emission,omitempty"`
	BlackBodyGlow *blackBodyReceipt `json:"blackBodyGlow,omitempty"`
}

type colourReceipt struct {
	LinearRGB *[3]float64 `json:"linearRGB,omitempty"`
	Swatch    string      `json:"swatch,omitempty"`
}

type blackBodyReceipt struct {
	TemperatureKelvin float64 `json:"temperatureKelvin"`
	Luminance         float64 `json:"luminance"`
}

type instanceReceipt struct {
	MaterialSlot string           `json:"materialSlot"`
	FaceCount    int              `json:"faceCount"`
	Placement    transformReceipt `json:"placement"`
}

type transformReceipt struct {
	Origin [3]float64 `json:"origin"`
	XAxis  [3]float64 `json:"xAxis"`
	YAxis  [3]float64 `json:"yAxis"`
	ZAxis  [3]float64 `json:"zAxis"`
	Scale  float64    `json:"scale"`
}

// importReceipt writes provenance after the draw has succeeded.
func (run runner) importReceipt() error {
	read, has := run.wanted.ImportedSet()
	if !has {
		return nil
	}
	world, has := read.World()
	if !has || len(world.Imports()) == 0 {
		return nil
	}
	path, err := importReceiptPath(run.wanted, run.at)
	if err != nil {
		return err
	}
	made, err := makeImportReceipt(run.wanted, run.env.build, world)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(made, "", "  ")
	if err != nil {
		return fmt.Errorf("import receipt: encode: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), folderMode); err != nil {
		return fmt.Errorf("import receipt: %w", err)
	}
	if err := replaced(path, raw); err != nil {
		return fmt.Errorf("import receipt: %w", err)
	}
	for _, part := range world.Imports() {
		if _, err := fmt.Fprintf(run.out, "import %s: %s; receipt %s\n",
			part.Name(), part.License(), path); err != nil {
			return err
		}
	}
	return nil
}

// importReceiptPath puts the record beside durable outputs, never temp frames.
func importReceiptPath(wanted job.Job, at places) (string, error) {
	if wanted.Clock().Still() {
		if at.picture == "" {
			return "", fmt.Errorf("import receipt: still has no picture path")
		}
		return at.picture + ".imports.json", nil
	}
	if at.work {
		if at.movie == "" {
			return "", fmt.Errorf("import receipt: working frames have no movie path")
		}
		return at.movie + ".imports.json", nil
	}
	if at.frames == "" {
		return "", fmt.Errorf("import receipt: film has no persistent frames path")
	}
	return filepath.Join(at.frames, "imports.json"), nil
}

// makeImportReceipt records the immutable job and already-checked import data.
func makeImportReceipt(wanted job.Job, build stamp, world set.World) (receipt, error) {
	canonical := wanted.Canonical()
	var parsed map[string]any
	if err := yamlfile.Read("job canonical form", canonical, &parsed); err != nil {
		return receipt{}, err
	}
	hashes, err := inputHashes(parsed)
	if err != nil {
		return receipt{}, err
	}
	importCount := len(world.Imports())
	imports := make([]importReceipt, 0, importCount)
	for _, part := range world.Imports() {
		asset := part.Asset()
		if asset == nil {
			return receipt{}, fmt.Errorf("import receipt: %s has no checked asset", part.Name())
		}
		optionsKey := asset.OptionsKey()
		worldUnit := world.Unit()
		made := importReceipt{Name: part.Name(), File: part.File(),
			Source: part.Source(), License: part.License(),
			RequestedHeightMetres: part.RequestedHeightMetres(),
			DecoderOptionsKey:     hex.EncodeToString(optionsKey[:]),
			SourceClosure:         sourceRecords(part),
			NativePlacement:       transformRecord(part.Placement(worldUnit)),
			AssetInstances:        instanceRecords(asset.Instances())}
		made.Normalization = normalizationRecord(asset)
		if selected, has := part.SelectedScene(); has {
			made.SelectedScene = &selected
		}
		bindings, err := bindingRecords(part, world.Materials())
		if err != nil {
			return receipt{}, err
		}
		made.NativeBindings = bindings
		imports = append(imports, made)
	}
	return receipt{Version: 1, Job: jobReceipt{File: wanted.File(), Key: wanted.Key(),
		PictureKey: wanted.PictureKey(), Canonical: json.RawMessage(canonical),
		InputHashes: hashes, InputFiles: wanted.Inputs()},
		Build: buildReceipt{Stamp: build.drawnBy(), Build: build.build,
			Built: build.built, Who: build.who, Sessions: build.sessions},
		Imports: imports}, nil
}

// normalizationRecord reports original bounds and the applied local transform.
func normalizationRecord(asset *oakland.Asset) *normalizationReceipt {
	value, has := asset.HeightNormalization()
	if !has {
		return nil
	}
	minimum, maximum := value.SourceBoundsY()
	return &normalizationReceipt{SourceBoundsYMetres: [2]float64{minimum, maximum},
		TargetHeight: value.TargetHeight(), HeightUnit: value.HeightUnit(),
		ScaleFactor: value.ScaleFactor(), VerticalOffset: value.VerticalOffset()}
}

// inputHashes extracts the path digests from the canonical checked job.
func inputHashes(canonical map[string]any) (map[string]string, error) {
	values, ok := canonical["inputs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("import receipt: canonical job has no input hashes")
	}
	hashes := make(map[string]string, len(values))
	for name, value := range values {
		digest, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("import receipt: input hash %q is not text", name)
		}
		hashes[name] = digest
	}
	return hashes, nil
}

// sourceRecords resolves closure-relative names beneath the original asset root.
func sourceRecords(part set.Import) []sourceReceipt {
	root := part.AssetRoot()
	sources := part.Asset().SourceClosure()
	made := make([]sourceReceipt, 0, len(sources))
	for _, source := range sources {
		digest := source.SHA256()
		sourceRelative := filepath.FromSlash(source.Path())
		made = append(made, sourceReceipt{Path: source.Path(),
			File:  filepath.Join(root, sourceRelative),
			Bytes: source.Size(), SHA256: hex.EncodeToString(digest[:])})
	}
	return made
}

// bindingRecords attaches every selected glTF slot to its native definition.
func bindingRecords(part set.Import, materials []set.Material) ([]bindingReceipt, error) {
	byName := make(map[string]set.Material, len(materials))
	for _, material := range materials {
		byName[material.Name()] = material
	}
	bindings := part.Bindings()
	slots := make([]string, 0, len(bindings))
	for slot := range bindings {
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	made := make([]bindingReceipt, 0, len(slots))
	for _, slot := range slots {
		name := bindings[slot]
		material, has := byName[name]
		if !has {
			return nil, fmt.Errorf("import receipt: native material %q is absent", name)
		}
		made = append(made, bindingReceipt{Slot: slot, Material: name,
			Definition: materialRecord(material)})
	}
	return made, nil
}

// materialRecord preserves a native definition and its authored provenance.
func materialRecord(material set.Material) materialReceipt {
	made := materialReceipt{Name: material.Name(), Source: material.Source()}
	if colour, has := material.Colour(); has {
		record := colourRecord(colour)
		made.Reflectance = &record
	}
	if colour, has := material.Glowing(); has {
		record := colourRecord(colour)
		made.Emission = &record
	}
	if temperature, luminance, has := material.Glow(); has {
		made.BlackBodyGlow = &blackBodyReceipt{TemperatureKelvin: temperature,
			Luminance: luminance}
	}
	return made
}

// colourRecord keeps either the exact linear colour or the named swatch.
func colourRecord(colour set.Colour) colourReceipt {
	if swatch, has := colour.Swatch(); has {
		return colourReceipt{Swatch: swatch}
	}
	red, green, blue := colour.Linear()
	linear := [3]float64{red, green, blue}
	return colourReceipt{LinearRGB: &linear}
}

// instanceRecords documents every mesh placement in the checked source scene.
func instanceRecords(instances []oakland.Instance) []instanceReceipt {
	made := make([]instanceReceipt, 0, len(instances))
	for _, instance := range instances {
		instancePlace := instance.Placement()
		made = append(made, instanceReceipt{MaterialSlot: instance.MaterialSlot(),
			FaceCount: instance.Geometry().FaceCount(),
			Placement: transformRecord(instancePlace)})
	}
	return made
}

// transformRecord writes a native transform as basis, origin and scale.
func transformRecord(transform maths.Subspace) transformReceipt {
	return transformReceipt{
		Origin: [3]float64{transform.Origin.X, transform.Origin.Y, transform.Origin.Z},
		XAxis:  [3]float64{transform.Orientation.X.X, transform.Orientation.X.Y, transform.Orientation.X.Z},
		YAxis:  [3]float64{transform.Orientation.Y.X, transform.Orientation.Y.Y, transform.Orientation.Y.Z},
		ZAxis:  [3]float64{transform.Orientation.Z.X, transform.Orientation.Z.Y, transform.Orientation.Z.Z},
		Scale:  transform.Scale,
	}
}
