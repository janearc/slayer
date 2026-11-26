package regioncompare

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/janearc/slayer/render/regionmanifest"
	"github.com/janearc/slayer/render/regionref"
	"github.com/janearc/slayer/render/regionset"
)

type referencePlan struct {
	Version string               `json:"version"`
	Regions []referencePlanEntry `json:"regions"`
}

type referencePlanEntry struct {
	Name     string                  `json:"name"`
	Manifest regionmanifest.Manifest `json:"manifest"`
}

// WriteReferencePlan records the manifests needed to pin references on a
// later run. Their origin and the declared input closure remain attested.
func WriteReferencePlan(out io.Writer, source *SessionSource) error {
	if out == nil || source == nil || len(source.References) == 0 {
		return fmt.Errorf("regioncompare: references and writer are required")
	}
	names := make([]string, 0, len(source.References))
	for name := range source.References {
		names = append(names, name)
	}
	sort.Strings(names)
	plan := referencePlan{Version: "region-reference-plan-v1"}
	for _, name := range names {
		ref := source.References[name]
		id, err := regionmanifest.Digest(ref.Manifest)
		if err != nil || Identity(id) != ref.Identity {
			return fmt.Errorf("regioncompare: invalid reference manifest for %q", name)
		}
		plan.Regions = append(plan.Regions, referencePlanEntry{Name: name, Manifest: ref.Manifest})
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(plan)
}

// ApplyReferencePlan selects previously recorded references. The directory
// supplies artifacts named by manifest identity, never paths from the plan.
func ApplyReferencePlan(input io.Reader, source *SessionSource, directory string) error {
	if input == nil || source == nil || len(source.References) == 0 || !filepath.IsAbs(directory) {
		return fmt.Errorf("regioncompare: planned references and directory are required")
	}
	if err := source.check(source.JobName); err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(input, (4<<20)+1))
	if err != nil {
		return err
	}
	if len(data) > 4<<20 {
		return fmt.Errorf("regioncompare: reference plan is too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var plan referencePlan
	if err := decoder.Decode(&plan); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("regioncompare: expected one reference plan")
	}
	if plan.Version != "region-reference-plan-v1" || len(plan.Regions) != len(source.References) {
		return fmt.Errorf("regioncompare: reference plan differs from regions")
	}
	executable, err := executableDigest()
	if err != nil {
		return err
	}
	selected := make(map[string]ReferenceFile, len(plan.Regions))
	for _, entry := range plan.Regions {
		current, found := source.References[entry.Name]
		if !found || entry.Name == "" {
			return fmt.Errorf("regioncompare: unexpected reference region %q", entry.Name)
		}
		if _, duplicate := selected[entry.Name]; duplicate {
			return fmt.Errorf("regioncompare: duplicate reference region %q", entry.Name)
		}
		id, err := regionmanifest.Digest(entry.Manifest)
		if err != nil {
			return err
		}
		ref := ReferenceFile{Identity: Identity(id), Manifest: entry.Manifest,
			Path: filepath.Join(directory, hex.EncodeToString(id[:])+".regionref"), Rays: entry.Manifest.Rays, MaxBytes: current.MaxBytes}
		candidate, found := source.Manifests[entry.Name]
		if !found {
			return fmt.Errorf("regioncompare: candidate manifest is required for %q", entry.Name)
		}
		box := candidate.Region
		region := regionset.Region{Name: entry.Name, Left: box.Left, Top: box.Top, Width: box.Width, Height: box.Height}
		if err := source.matchReference(region, ref); err != nil {
			return fmt.Errorf("regioncompare: reference %q: %w", entry.Name, err)
		}
		if _, err := regionref.Read(ref.Path, regionref.Identity(id), region.Width, region.Height, ref.MaxBytes); err != nil {
			if !errors.Is(err, os.ErrNotExist) || ref.Manifest.Executable != executable {
				return fmt.Errorf("regioncompare: reference %q is unavailable: %w", entry.Name, err)
			}
		}
		selected[entry.Name] = ref
	}
	source.References = selected
	return nil
}
