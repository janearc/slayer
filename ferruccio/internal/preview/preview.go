// Package preview renders frozen drafts using an explicitly selected transport.
package preview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
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

// Profile freezes preview dimensions and an optional CPU renderer choice.
type Profile struct {
	Width, Height int
	CPURenderer   bool
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
type Tools struct{ Game, Slayer, Catalogue, GameHome, Shelf, Cache, Azazel, Queue, Agent, Transport string }

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
		profile.CPURenderer = false
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
	if err := CopySourceManifest(request.Path, folder); err != nil {
		os.RemoveAll(folder)
		return nil, err
	}
	input, err := os.CreateTemp(filepath.Dir(request.Path), ".ferruccio-preview-*.yaml")
	if err != nil {
		os.Remove(folder)
		return nil, err
	}
	if _, err := input.Write(source.Bytes()); err != nil {
		input.Close()
		os.Remove(input.Name())
		os.Remove(folder)
		return nil, err
	}
	if err := input.Close(); err != nil {
		os.Remove(input.Name())
		os.Remove(folder)
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(folder, "scene.yaml"), source.Bytes(), 0600); err != nil {
		os.Remove(input.Name())
		os.RemoveAll(folder)
		return nil, err
	}
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

// record accepts only the frozen profile output and records its actual dimensions.
func (plan *Plan) record(_ Tools) error {
	profile, err := plan.Profile.resolved()
	if err != nil {
		return err
	}
	file, err := os.Open(plan.Output)
	if err != nil {
		return fmt.Errorf("producer finished without the preview image: %w", err)
	}
	picture, err := png.DecodeConfig(file)
	closeError := file.Close()
	if err != nil {
		return fmt.Errorf("preview PNG: %w", err)
	}
	if closeError != nil {
		return closeError
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

// environment retains the user's machine identity while bounding renderer work.
func environment(tools Tools) []string {
	replacements := map[string]string{"GOMAXPROCS": "2"}
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
		if _, exists := replacements[key]; !exists {
			result = append(result, value)
		}
	}
	for key, value := range replacements {
		result = append(result, key+"="+value)
	}
	return result
}
