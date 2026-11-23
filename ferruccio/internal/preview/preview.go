// Package preview renders frozen drafts using an explicitly selected transport.
package preview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
// choice. GPU is the person's own choice of the GPU renderer and is never set
// with CPURenderer; with neither, the producer's own renderer is used.
type Profile struct {
	Width, Height int
	CPURenderer   bool
	GPU           bool
	Full          bool
	Rays          int
}

// Plan owns one temporary input and a fresh output directory.
type Plan struct {
	Input, Output, Directory, Digest, Source, Kind, Specimen, Detail string
	Profile                                                          Profile
	state                                                            *planState
}

// Tools identifies existing executables and their scoped execution settings.
// Renderer and Processors are the person's own choices for a render started
// by this client: "cpu" or "gpu", and "2", "4", "8" or "all". Empty is the
// default, the CPU renderer on two processors.
type Tools struct {
	Game, Slayer, Catalogue, GameHome, Shelf, Cache, Azazel, Queue, Agent, Transport string
	Renderer, Processors                                                             string
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
	if profile.Full {
		profile, err = authoredProfile(tree, document.Kind(), profile)
		if err != nil {
			return nil, err
		}
	}
	var specimen, detail string
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
	return &Plan{Input: input.Name(), Output: output, Directory: folder, Digest: hex.EncodeToString(sum[:]), Source: request.Path, Kind: document.Kind(), Specimen: specimen, Detail: detail, Profile: profile, state: &planState{}}, nil
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
	if picture.Width != profile.Width || picture.Height != profile.Height {
		return fmt.Errorf("preview is %dx%d, expected %dx%d", picture.Width, picture.Height, profile.Width, profile.Height)
	}
	receipt := map[string]any{"source_sha256": plan.Digest, "width": picture.Width, "height": picture.Height, "preview": !profile.Full, "quality": profile.quality(), "rays_per_axis": profile.samples(), "document_kind": plan.Kind}
	receipt["lane"] = plan.lane()
	receipt["renderer_override"] = ""
	if plan.Profile.CPURenderer && plan.Kind != "catalogue" {
		receipt["renderer_override"] = "cpu"
	}
	if plan.Profile.GPU && plan.Kind != "catalogue" {
		receipt["renderer_override"] = "gpu"
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
