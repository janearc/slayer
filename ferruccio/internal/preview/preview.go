// Package preview renders frozen drafts using an explicitly selected transport.
package preview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"gopkg.in/yaml.v3"
)

// Request is the current draft and its original path for relative inputs.
type Request struct {
	Path, Text, Specimen string
	Profile              Profile
	OutputRoot           string
	DirectoryPrefix      string
}

// Profile freezes preview dimensions and an optional CPU or GPU renderer
// choice. With no renderer selected, the producer's own renderer is used.
type Profile struct {
	Width, Height int
	CPURenderer   bool
	GPU           bool
	NativeGPU     bool
	Full          bool
	Rays          int
	// Samples is an explicit total per pixel; zero keeps authored sampling.
	Samples int
	// SizePreset names a producer output-size preset selected for this render.
	// Width and Height are its dimensions, frozen when the choice is made.
	SizePreset string
	// Camera overrides the authored camera by its producer-defined name.
	Camera string
	// CameraWide is the checked horizontal angle inherited from a still job.
	CameraWide float64
	// CameraPathRecipe is the checked job camera and still time, frozen at open.
	CameraPathRecipe string
	// StillAt preserves an inherited clock when an explicit camera replaces a path.
	StillAt   time.Duration
	rig       *physicalRig
	stock     *developmentStock
	reference filmReference
	region    *renderRegion
}

// Plan owns one temporary input and a fresh output directory.
type Plan struct {
	Input, Output, Directory, Digest, Source, Kind, Specimen, Detail string
	Profile                                                          Profile
	state                                                            *planState
	// relocated says its frozen input lives away from the source directory.
	relocated bool
}

// Tools identifies existing executables and their scoped execution settings.
// Renderer and Processors are the person's own choices for a render started
// by this client: "cpu" or "gpu", and "2", "4", "8" or "all". Empty is the
// default, the CPU renderer on two processors.
type Tools struct {
	Game, Slayer, Catalogue, GameHome, Shelf, Cache, Data, Azazel, Queue, Agent, Transport string
	Renderer, Processors                                                                   string
	// ManageWorker permits a drain process on demand for this Azazel queue.
	ManageWorker bool
}

// Prepare checks a still request without changing the source or its outputs.
func Prepare(request Request) (*Plan, error) {
	if request.Path == "" {
		return nil, fmt.Errorf("open or save a YAML file before rendering")
	}
	prefix, err := request.directoryPrefix()
	if err != nil {
		return nil, err
	}
	profile, err := request.Profile.resolved()
	if err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(request.Path)
	if err != nil {
		return nil, err
	}
	request.Path = absolute
	document, err := draft.Open(filepath.Base(request.Path), []byte(request.Text))
	if err != nil {
		return nil, err
	}
	tree, err := document.Tree()
	if err != nil {
		return nil, err
	}
	if profile.Full || profile.CameraPathRecipe != "" {
		full, err := readFullFilm(request.Path, document, tree)
		if err != nil {
			return nil, err
		}
		if profile.Full {
			profile, err = full.Resolve(profile)
		} else {
			var authored Profile
			authored, err = full.Resolve(Profile{Full: true, CameraPathRecipe: profile.CameraPathRecipe})
			if err == nil {
				profile.Width, profile.Height, err = fitCameraAspect(authored.Width, authored.Height, profile.Width, profile.Height)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	var specimen, detail string
	if (profile.CameraWide != 0 || profile.CameraPathRecipe != "" || profile.StillAt != 0) && document.Kind() != "set" && document.Kind() != "legacy-set" {
		return nil, fmt.Errorf("an inherited camera override requires a scene set")
	}
	if profile.CameraWide != 0 && field(field(field(tree.Content[0], "scene"), "film"), "frame") != nil {
		return nil, fmt.Errorf("a physical camera rig cannot use an inherited job zoom")
	}
	if document.Kind() == "catalogue" {
		specimen, err = ResolveSpecimen(request)
		if err != nil {
			return nil, err
		}
		profile.CPURenderer, profile.GPU = false, false
		detail = catalogueSnapshot(tree, profile)
	}
	if document.Kind() == "job" {
		root := tree.Content[0]
		for index := 0; index+1 < len(root.Content); index += 2 {
			if root.Content[index].Value == "fps" || root.Content[index].Value == "duration" {
				return nil, fmt.Errorf("this preview draws still jobs and set files; movie rendering is not connected yet")
			}
		}
		for index := len(root.Content) - 2; index >= 0; index -= 2 {
			if root.Content[index].Value == "outputs" {
				root.Content = append(root.Content[:index], root.Content[index+2:]...)
			}
		}
	}
	var source bytes.Buffer
	encoder := yaml.NewEncoder(&source)
	encoder.SetIndent(2)
	if err := encoder.Encode(tree); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	outputRoot := request.OutputRoot
	if outputRoot != "" {
		outputRoot, err = filepath.Abs(outputRoot)
		if err != nil {
			return nil, err
		}
	}
	folder, err := os.MkdirTemp(outputRoot, prefix)
	if err != nil {
		return nil, err
	}
	var inputName string
	prepared := false
	defer func() {
		if prepared {
			return
		}
		// Every error path removes what was made, including a folder that
		// already holds a copied manifest.
		os.RemoveAll(folder)
		if inputName != "" {
			os.Remove(inputName)
		}
	}()
	if err := CopySourceManifest(request.Path, folder); err != nil {
		return nil, err
	}
	input, err := os.CreateTemp(filepath.Dir(request.Path), ".ferruccio-preview-*.yaml")
	relocated := false
	if err != nil && (os.IsPermission(err) || errors.Is(err, syscall.EROFS)) && document.Kind() != "catalogue" {
		input, err = os.CreateTemp(folder, ".ferruccio-preview-*.yaml")
		relocated = err == nil
	}
	if err != nil {
		return nil, err
	}
	inputName = input.Name()
	if _, err := input.Write(source.Bytes()); err != nil {
		input.Close()
		return nil, err
	}
	if err := input.Close(); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(folder, "scene.yaml"), source.Bytes(), 0600); err != nil {
		return nil, err
	}
	prepared = true
	sum := sha256.Sum256([]byte(request.Text))
	output := filepath.Join(folder, "preview.png")
	if document.Kind() == "catalogue" {
		output = filepath.Join(folder, "book", specimen+"-"+detail, "specimen.png")
	}
	return &Plan{Input: input.Name(), Output: output, Directory: folder, Digest: hex.EncodeToString(sum[:]), Source: request.Path, Kind: document.Kind(), Specimen: specimen, Detail: detail, Profile: profile, state: &planState{}, relocated: relocated}, nil
}

// Close removes a temporary input only when no unconfirmed queue job needs it.
// Queued, running, uncertain and interrupted inputs remain; image evidence stays.
func (plan *Plan) Close() error {
	if plan == nil || plan.Input == "" {
		return nil
	}
	if plan.state != nil {
		plan.state.mu.Lock()
		active := plan.state.active || plan.state.uncertain
		plan.state.mu.Unlock()
		if active {
			return nil
		}
	}
	if plan.queuedInputNeeded() {
		return nil
	}
	err := os.Remove(plan.Input)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Run submits and observes one external queue job without progress callbacks.
func Run(ctx context.Context, tools Tools, plan *Plan, output io.Writer) error {
	return RunObserved(ctx, tools, plan, output, nil)
}

// decodeWholePNG reads every pixel of the file before it is called good, so a
// truncated file is a failure here and not later in the viewer. The header is
// read first, so a wrong-sized image is named by its size.
func decodeWholePNG(path string) (image.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return image.Config{}, fmt.Errorf("producer finished without the preview image: %w", err)
	}
	defer file.Close()
	configuration, err := png.DecodeConfig(file)
	if err != nil {
		return image.Config{}, fmt.Errorf("preview PNG: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return image.Config{}, err
	}
	if _, err := png.Decode(file); err != nil {
		return image.Config{}, fmt.Errorf("preview PNG is truncated or damaged (%dx%d header): %w", configuration.Width, configuration.Height, err)
	}
	return configuration, nil
}

// record accepts only the frozen profile output and records its actual dimensions.
func (plan *Plan) record(_ Tools) error {
	profile, err := plan.Profile.resolved()
	if err != nil {
		return err
	}
	picture, err := decodeWholePNG(plan.Output)
	if err != nil {
		return err
	}
	width, height := profile.OutputSize()
	if picture.Width != width || picture.Height != height {
		return fmt.Errorf("preview is %dx%d, expected %dx%d", picture.Width, picture.Height, width, height)
	}
	receipt := map[string]any{"source_sha256": plan.Digest, "width": picture.Width, "height": picture.Height, "preview": !profile.Full, "quality": profile.quality(), "document_kind": plan.Kind}
	receipt["samples_per_pixel"] = profile.SamplesPerPixel()
	if profile.Full {
		receipt["primary_samples_requested"] = profile.PrimarySamples()
		if reference := profile.reference; reference.path != "" {
			receipt["referenced_set"] = map[string]string{"path": reference.path, "sha256": reference.digest}
		}
	}
	if region := profile.region; region != nil {
		receipt["region"] = map[string]int{
			"left": region.Left, "top": region.Top, "width": width, "height": height,
			"frame_width": profile.Width, "frame_height": profile.Height,
		}
	}
	if profile.rig != nil {
		receipt["physical_rig"] = profile.rig
	}
	if profile.stock != nil {
		receipt["development_stock"] = profile.stock
	}
	if !profile.Full || profile.Samples == 0 {
		receipt["rays_per_axis"] = profile.samples()
	}
	if profile.SizePreset != "" {
		receipt["size_preset"] = profile.SizePreset
	}
	if profile.Camera != "" {
		receipt["camera_override"] = profile.Camera
	}
	if profile.CameraWide != 0 {
		receipt["camera_wide_degrees"] = profile.CameraWide
	}
	if profile.CameraPathRecipe != "" {
		receipt["camera_path_recipe"] = json.RawMessage(profile.CameraPathRecipe)
	}
	if profile.StillAt != 0 {
		receipt["still_at"] = profile.StillAt.String()
	}
	receipt["lane"] = plan.lane()
	receipt["renderer_override"] = ""
	if plan.Kind != "catalogue" {
		receipt["renderer_override"] = profile.rendererOverride()
	}
	if plan.Kind == "catalogue" {
		receipt["specimen"], receipt["detail"] = plan.Specimen, plan.Detail
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(plan.Directory, "receipt.json"), raw, 0o644)
}

// ChildEnvironment keeps the user's environment except this process's
// diagnostic listener addresses, which cannot be rebound by a child.
func ChildEnvironment() []string {
	var result []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key != "SLAYER_METRICS_ADDR" && key != "SLAYER_PPROF_ADDR" {
			result = append(result, entry)
		}
	}
	return result
}

// defaultProcessors is how many processors a render may use when the person
// has not chosen: two, which keeps the rest of the machine usable.
const defaultProcessors = "2"

// ProcessorsAll is the choice of every processor the machine has.
const ProcessorsAll = "all"

// environment retains the user's machine identity while bounding renderer
// work to the processors the person chose, two by default. With "all" the
// child is given no limit, and any limit this process inherited is dropped.
func environment(tools Tools) []string {
	replacements := map[string]string{}
	processors := tools.Processors
	if count, err := strconv.Atoi(processors); processors != ProcessorsAll && (err != nil || count < 1) {
		processors = defaultProcessors
	}
	if processors != ProcessorsAll {
		replacements["GOMAXPROCS"] = processors
	}
	if tools.GameHome != "" {
		replacements["GAME_HOME"] = tools.GameHome
	}
	if tools.Cache != "" {
		replacements["SLAYER_CACHE"] = tools.Cache
	}
	if tools.Shelf != "" {
		replacements["SLAYER_SHELF"] = tools.Shelf
	}
	if tools.Data != "" {
		replacements["SLAYER_DATA"] = tools.Data
	}
	var result []string
	for _, value := range ChildEnvironment() {
		key, _, _ := strings.Cut(value, "=")
		_, replaced := replacements[key]
		unlimited := key == "GOMAXPROCS" && processors == ProcessorsAll
		if !replaced && !unlimited {
			result = append(result, value)
		}
	}
	for key, value := range replacements {
		result = append(result, key+"="+value)
	}
	return result
}
