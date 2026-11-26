package regioncompare

import (
	"encoding/hex"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/render/regionmanifest"
	"github.com/janearc/slayer/render/regionset"
	"github.com/janearc/slayer/session"
)

// PlanSession prepares candidate and reference identities from a frozen,
// caller-enumerated snapshot. closureAsserted is the caller's assertion that
// Entries include every input; this method cannot discover missing inputs or
// ensure that caches and environment-selected data are stable. The reference
// directory must already exist. No images or references are written here.
func (s *SessionSource) PlanSession(regions []regionset.Region, referenceRays int, referenceDir string, closureAsserted bool) error {
	if s == nil {
		return fmt.Errorf("regioncompare: source is required")
	}
	if err := s.check(s.JobName); err != nil {
		return err
	}
	if !closureAsserted || len(regions) == 0 || referenceRays <= s.Rays || referenceRays > 64 || !filepath.IsAbs(referenceDir) {
		return fmt.Errorf("regioncompare: asserted closure, regions, finer sampling and reference directory are required")
	}
	info, err := os.Stat(referenceDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("regioncompare: reference directory is not a directory")
	}
	wanted, err := job.Read(filepath.Join(s.Snapshot, s.JobFile))
	if err != nil {
		return err
	}
	setPath, ok := wanted.Scene().Set()
	if !ok || !s.includes(setPath) {
		return fmt.Errorf("regioncompare: job set is not in the snapshot")
	}
	if _, said := wanted.Print(); said {
		return fmt.Errorf("regioncompare: job print is not supported")
	}
	if _, said := wanted.Stock(); said {
		return fmt.Errorf("regioncompare: job stock is not supported")
	}
	if _, said := wanted.Titles(); said {
		return fmt.Errorf("regioncompare: job titles are not supported")
	}
	if len(wanted.Times()) != 1 {
		return fmt.Errorf("regioncompare: only stills are supported")
	}
	renderer, said := wanted.Renderer()
	camera, cameraSaid := wanted.Camera()
	cameraName, named := camera.Name()
	if !said || !cameraSaid || !named {
		return fmt.Errorf("regioncompare: explicit renderer and named camera are required")
	}
	executable, err := executableDigest()
	if err != nil {
		return err
	}
	inputs := make([]regionmanifest.Input, 0, len(s.Entries))
	for _, entry := range s.Entries {
		inputs = append(inputs, regionmanifest.Input{Name: entry.Name, Bytes: entry.Bytes, SHA256: entry.SHA256})
	}
	at := int64(wanted.Times()[0])
	detail := scrooge.Asked()
	base := regionmanifest.Manifest{ClosureAsserted: true, Inputs: inputs, Executable: executable,
		Width: s.Width, Height: s.Height, Renderer: renderer, Camera: cameraName, TimeNS: &at, Scrooge: &detail}
	if err := matchDetail(base); err != nil {
		return err
	}
	candidates := make(map[string]Identity, len(regions))
	manifests := make(map[string]regionmanifest.Manifest, len(regions))
	references := make(map[string]ReferenceFile, len(regions))
	seen := make(map[string]bool, len(regions))
	for _, region := range regions {
		if region.Name == "" || seen[region.Name] || region.Left < 0 || region.Top < 0 ||
			region.Width <= 0 || region.Height <= 0 || region.Left >= s.Width || region.Top >= s.Height ||
			region.Width > s.Width-region.Left || region.Height > s.Height-region.Top ||
			int64(region.Width) > (math.MaxInt64-80)/12/int64(region.Height) {
			return fmt.Errorf("regioncompare: invalid or duplicate region %q", region.Name)
		}
		seen[region.Name] = true
	}
	var development [32]byte
	for index, region := range regions {
		box := image.Rect(region.Left, region.Top, region.Left+region.Width, region.Top+region.Height)
		candidateJob, err := wanted.With(job.Width(s.Width), job.Height(s.Height), job.Rays(s.Rays), job.Region(box))
		if err != nil {
			return err
		}
		if index == 0 {
			opened, err := session.Open(s.Context, candidateJob, s.Stamp)
			if err != nil {
				return err
			}
			tone := opened.Tone()
			if err := opened.Close(); err != nil {
				return err
			}
			if tone.Bloom != 0 {
				return fmt.Errorf("regioncompare: bloom is not supported")
			}
			development, err = ToneDigest(tone)
			if err != nil {
				return err
			}
		}
		manifest := base
		manifest.Region = regionmanifest.Rectangle{Left: region.Left, Top: region.Top, Width: region.Width, Height: region.Height}
		manifest.Rays, manifest.Settings, manifest.Development = s.Rays, JobSettingsDigest(candidateJob), development
		candidate, err := regionmanifest.Digest(manifest)
		if err != nil {
			return err
		}
		refJob, err := candidateJob.With(job.Rays(referenceRays))
		if err != nil {
			return err
		}
		ref := manifest
		ref.Rays, ref.Settings = referenceRays, JobSettingsDigest(refJob)
		refID, err := regionmanifest.Digest(ref)
		if err != nil {
			return err
		}
		candidates[region.Name], manifests[region.Name] = Identity(candidate), manifest
		references[region.Name] = ReferenceFile{Identity: Identity(refID), Manifest: ref,
			Path: filepath.Join(referenceDir, hex.EncodeToString(refID[:])+".regionref"), Rays: referenceRays,
			MaxBytes: 80 + int64(region.Width)*int64(region.Height)*12}
	}
	s.Candidates, s.Manifests, s.References, s.tone = candidates, manifests, references, nil
	return nil
}
