package regioncompare

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/regioninput"
	"github.com/janearc/slayer/render/regionmanifest"
	"github.com/janearc/slayer/render/regionref"
	"github.com/janearc/slayer/render/regionset"
	"github.com/janearc/slayer/session"
)

// ReferenceFile identifies a raw reference and its caller-attested sampling.
type ReferenceFile struct {
	Identity Identity
	Manifest regionmanifest.Manifest
	Path     string
	Rays     int
	MaxBytes int64
}

// SessionSource draws from a caller-asserted frozen snapshot. Entries must
// enumerate all inputs used by the job; the caller, not this type, establishes
// that closure and prevents mutation during use. It is not safe for concurrent
// calls. Shelf and Cache must match the process environment. References and
// their identities, rays and development compatibility are caller-attested.
//
// Stamp is supplied by the caller; its presence does not verify the executable.
// JobFile is a job document naming the snapshotted set; a set alone needs a
// job wrapper. Width, Height and Rays stand over the job's film settings.
type SessionSource struct {
	Context             context.Context
	Stamp               session.Stamp
	Snapshot            string
	Entries             []regioninput.Entry
	JobName             string
	JobFile             string
	Shelf               string
	Cache               string
	Width, Height, Rays int
	Candidates          map[string]Identity
	Manifests           map[string]regionmanifest.Manifest
	References          map[string]ReferenceFile
	tone                map[string]colour.Tone
}

// Draw validates the asserted files and draws one region through a session.
func (s *SessionSource) Draw(name string, region regionset.Region, id Identity) (Picture, error) {
	if s != nil {
		delete(s.tone, region.Name)
	}
	if err := s.check(name); err != nil {
		return Picture{}, err
	}
	if id == (Identity{}) || s.Candidates[region.Name] != id {
		return Picture{}, fmt.Errorf("regioncompare: candidate identity is not pinned")
	}
	if region.Left < 0 || region.Top < 0 || region.Width <= 0 || region.Height <= 0 ||
		region.Left >= s.Width || region.Top >= s.Height ||
		region.Width > s.Width-region.Left || region.Height > s.Height-region.Top {
		return Picture{}, fmt.Errorf("regioncompare: region is outside the frame")
	}
	manifest, found := s.Manifests[region.Name]
	if !found {
		return Picture{}, fmt.Errorf("regioncompare: candidate manifest is required")
	}
	if err := matchManifest(manifest, region, s.Entries, s.Width, s.Height, s.Rays); err != nil {
		return Picture{}, err
	}
	if err := matchDetail(manifest); err != nil {
		return Picture{}, err
	}
	identity, err := regionmanifest.Digest(manifest)
	if err != nil || Identity(identity) != id {
		return Picture{}, fmt.Errorf("regioncompare: candidate manifest identity differs")
	}
	executable, err := executableDigest()
	if err != nil || executable != manifest.Executable {
		return Picture{}, fmt.Errorf("regioncompare: running executable differs from manifest")
	}
	wanted, err := job.Read(filepath.Join(s.Snapshot, s.JobFile))
	if err != nil {
		return Picture{}, err
	}
	setPath, ok := wanted.Scene().Set()
	if !ok || !s.includes(setPath) {
		return Picture{}, fmt.Errorf("regioncompare: job set is not in the snapshot")
	}
	if _, said := wanted.Print(); said {
		return Picture{}, fmt.Errorf("regioncompare: job print is not supported")
	}
	if _, said := wanted.Stock(); said {
		return Picture{}, fmt.Errorf("regioncompare: job stock is not supported")
	}
	if _, said := wanted.Titles(); said {
		return Picture{}, fmt.Errorf("regioncompare: job titles are not supported")
	}
	if len(wanted.Times()) != 1 {
		return Picture{}, fmt.Errorf("regioncompare: only stills are supported")
	}
	box := image.Rect(region.Left, region.Top, region.Left+region.Width, region.Top+region.Height)
	wanted, err = wanted.With(job.Width(s.Width), job.Height(s.Height), job.Rays(s.Rays), job.Region(box))
	if err != nil {
		return Picture{}, err
	}
	renderer, said := wanted.Renderer()
	if !said || renderer != manifest.Renderer {
		return Picture{}, fmt.Errorf("regioncompare: renderer must be explicit and match the manifest")
	}
	camera, said := wanted.Camera()
	cameraName, named := camera.Name()
	if !said || !named || cameraName != manifest.Camera || int64(wanted.Times()[0]) != *manifest.TimeNS ||
		JobSettingsDigest(wanted) != manifest.Settings {
		return Picture{}, fmt.Errorf("regioncompare: camera, time or effective job settings differ")
	}
	opened, err := session.Open(s.Context, wanted, s.Stamp)
	if err != nil {
		return Picture{}, err
	}
	defer opened.Close()
	tone := opened.Tone()
	if tone.Bloom != 0 {
		return Picture{}, fmt.Errorf("regioncompare: bloom is not supported")
	}
	development, err := ToneDigest(tone)
	if err != nil || development != manifest.Development {
		return Picture{}, fmt.Errorf("regioncompare: development differs from manifest")
	}
	ask, err := session.NewAsk(0, 0, 0, nil)
	if err != nil {
		return Picture{}, err
	}
	drawn, err := opened.Frame(s.Context, 0, ask)
	if err != nil {
		return Picture{}, err
	}
	if err := s.check(name); err != nil {
		return Picture{}, err
	}
	if err := matchDetail(manifest); err != nil {
		return Picture{}, err
	}
	if s.tone == nil {
		s.tone = map[string]colour.Tone{}
	}
	s.tone[region.Name] = tone
	light := drawn.Light()
	return Picture{Identity: id, Light: light, View: tone.Develop(light), Rays: s.Rays}, nil
}

// Reference verifies the retained artifact and develops it with the same tone.
func (s *SessionSource) Reference(name string, region regionset.Region, id Identity) (Picture, error) {
	ref, tone, err := s.referenceSpec(name, region, id)
	if err != nil {
		return Picture{}, err
	}
	light, err := regionref.Read(ref.Path, regionref.Identity(id), region.Width, region.Height, ref.MaxBytes)
	if err != nil {
		return Picture{}, err
	}
	return Picture{Identity: id, Light: light, View: tone.Develop(light), Rays: ref.Rays}, nil
}

// referenceSpec checks the reference against the preceding candidate draw.
func (s *SessionSource) referenceSpec(name string, region regionset.Region, id Identity) (ReferenceFile, colour.Tone, error) {
	if err := s.check(name); err != nil {
		return ReferenceFile{}, colour.Tone{}, err
	}
	ref, found := s.References[region.Name]
	tone, drawn := s.tone[region.Name]
	if !found || !drawn || id == (Identity{}) || ref.Identity != id || ref.Rays <= s.Rays {
		return ReferenceFile{}, colour.Tone{}, fmt.Errorf("regioncompare: reference or its finer sampling is not pinned")
	}
	if err := s.matchReference(region, ref); err != nil {
		return ReferenceFile{}, colour.Tone{}, err
	}
	return ref, tone, nil
}

// matchReference checks compatibility before a reference is used or drawn.
func (s *SessionSource) matchReference(region regionset.Region, ref ReferenceFile) error {
	refIdentity, err := regionmanifest.Digest(ref.Manifest)
	if err != nil || Identity(refIdentity) != ref.Identity || ref.Rays <= s.Rays || ref.Manifest.Rays != ref.Rays ||
		ref.Manifest.Width != s.Width || ref.Manifest.Height != s.Height ||
		ref.Manifest.Region != (regionmanifest.Rectangle{Left: region.Left, Top: region.Top, Width: region.Width, Height: region.Height}) {
		return fmt.Errorf("regioncompare: reference manifest differs")
	}
	candidate, found := s.Manifests[region.Name]
	if !found {
		return fmt.Errorf("regioncompare: candidate manifest is required")
	}
	if err := matchManifest(ref.Manifest, region, s.Entries, s.Width, s.Height, ref.Rays); err != nil {
		return err
	}
	wanted, err := job.Read(filepath.Join(s.Snapshot, s.JobFile))
	if err != nil {
		return err
	}
	box := image.Rect(region.Left, region.Top, region.Left+region.Width, region.Top+region.Height)
	wanted, err = wanted.With(job.Width(s.Width), job.Height(s.Height), job.Rays(ref.Rays), job.Region(box))
	if err != nil {
		return err
	}
	if ref.Manifest.Renderer != candidate.Renderer ||
		ref.Manifest.Settings != JobSettingsDigest(wanted) || ref.Manifest.Camera != candidate.Camera || ref.Manifest.TimeNS == nil || candidate.TimeNS == nil ||
		*ref.Manifest.TimeNS != *candidate.TimeNS || ref.Manifest.Development != candidate.Development ||
		ref.Manifest.Scrooge == nil || candidate.Scrooge == nil || *ref.Manifest.Scrooge != *candidate.Scrooge {
		return fmt.Errorf("regioncompare: reference provenance or settings differ")
	}
	return nil
}

// EnsureReference draws and installs a missing reference without replacing one.
// The candidate must have been successfully drawn first. The same declared
// closure and cache-stability obligations apply to both draws.
func (s *SessionSource) EnsureReference(name string, region regionset.Region, id Identity) error {
	ref, _, err := s.referenceSpec(name, region, id)
	if err != nil {
		return err
	}
	if _, err := s.Reference(name, region, id); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	other := *s
	other.Rays = ref.Rays
	other.Candidates = map[string]Identity{region.Name: id}
	other.Manifests = map[string]regionmanifest.Manifest{region.Name: ref.Manifest}
	other.tone = nil
	drawn, err := other.Draw(name, region, id)
	if err != nil {
		return err
	}
	light := make([]byte, len(drawn.Light.Pix)*12)
	for i, pixel := range drawn.Light.Pix {
		binary.LittleEndian.PutUint32(light[i*12:], math.Float32bits(pixel.R))
		binary.LittleEndian.PutUint32(light[i*12+4:], math.Float32bits(pixel.G))
		binary.LittleEndian.PutUint32(light[i*12+8:], math.Float32bits(pixel.B))
	}
	return regionref.Publish(ref.Path, regionref.Identity(id), region.Width, region.Height, light)
}

// check verifies the declared snapshot and active environment.
func (s *SessionSource) check(name string) error {
	if s == nil || s.Context == nil || s.Stamp == (session.Stamp{}) || !filepath.IsAbs(s.Snapshot) || s.JobName == "" || name != s.JobName ||
		!filepath.IsLocal(s.JobFile) || s.Width <= 0 || s.Height <= 0 || s.Rays <= 0 || len(s.Entries) == 0 ||
		!filepath.IsAbs(s.Shelf) || !filepath.IsAbs(s.Cache) || os.Getenv("SLAYER_SHELF") != s.Shelf || os.Getenv("SLAYER_CACHE") != s.Cache {
		return fmt.Errorf("regioncompare: frozen snapshot and matching shelf and cache are required")
	}
	if !s.includes(filepath.Join(s.Snapshot, s.JobFile)) {
		return fmt.Errorf("regioncompare: job is not in the snapshot")
	}
	shelfRelative, err := filepath.Rel(s.Snapshot, s.Shelf)
	if err != nil || !filepath.IsLocal(shelfRelative) {
		return fmt.Errorf("regioncompare: shelf must be in the snapshot")
	}
	for _, entry := range s.Entries {
		if !filepath.IsLocal(entry.Name) || entry.Bytes < 0 {
			return fmt.Errorf("regioncompare: invalid snapshot entry")
		}
		f, err := os.Open(filepath.Join(s.Snapshot, entry.Name))
		if err != nil {
			return err
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		if !info.Mode().IsRegular() || info.Size() != entry.Bytes {
			f.Close()
			return fmt.Errorf("regioncompare: snapshot entry changed")
		}
		hash := sha256.New()
		_, err = io.Copy(hash, f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if string(hash.Sum(nil)) != string(entry.SHA256[:]) {
			return fmt.Errorf("regioncompare: snapshot digest differs")
		}
	}
	return nil
}

// includes reports whether a path is among the declared snapshot entries.
func (s *SessionSource) includes(path string) bool {
	for _, entry := range s.Entries {
		if filepath.Join(s.Snapshot, entry.Name) == path {
			return true
		}
	}
	return false
}
