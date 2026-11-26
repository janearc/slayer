// Package regioncompare compares named picture regions with pinned references.
package regioncompare

import (
	"fmt"
	"image"
	"io"
	"reflect"
	"time"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/interfaces/sheet"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/regionjudge"
	"github.com/janearc/slayer/render/regionset"
)

// Identity identifies a build, inputs and effective rendering settings.
type Identity [32]byte

// Picture is linear light and its developed view, with verified identity.
type Picture struct {
	Identity Identity
	Light    *frame.Frame
	View     image.Image
	Rays     int
}

// Source draws a candidate and reads a verified reference. Each method must
// return an error if it cannot establish the requested complete identity.
type Source interface {
	Draw(string, regionset.Region, Identity) (Picture, error)
	Reference(string, regionset.Region, Identity) (Picture, error)
}

// RegionResult is the comparison of one named region with its reference.
type RegionResult struct {
	Region            regionset.Region
	Candidate         Identity
	Reference         Identity
	CandidateRays     int
	ReferenceRays     int
	DrawDuration      time.Duration
	ReferenceDuration time.Duration
	Report            regionjudge.Report
	Repeats           []RepeatResult
}

// RepeatResult compares a repeated candidate's light with the first draw.
type RepeatResult struct {
	Duration time.Duration
	Report   regionjudge.Report
}

// Result holds every region's numbers and their labelled comparisons.
type Result struct {
	Regions   []RegionResult
	Sheet     image.Image
	Job       string
	Threshold float64
}

// Run reads a set and compares all its regions at the supplied frame size.
func Run(input io.Reader, width, height int, candidates map[string]Identity,
	references map[string]Identity, source Source) (Result, error) {
	return RunRepeated(input, width, height, candidates, references, source, 1)
}

// RunRepeated also compares each repeated candidate with its first draw.
func RunRepeated(input io.Reader, width, height int, candidates map[string]Identity,
	references map[string]Identity, source Source, repeats int) (Result, error) {
	const maxRepeats = 8
	if repeats < 1 || repeats > maxRepeats {
		return Result{}, fmt.Errorf("regioncompare: repeats must be between 1 and %d", maxRepeats)
	}
	set, err := regionset.Read(input, width, height)
	if err != nil {
		return Result{}, err
	}
	if isNil(source) {
		return Result{}, fmt.Errorf("regioncompare: verified source is required")
	}
	seenCandidates := make(map[Identity]regionset.Region)
	seenReferences := make(map[Identity]regionset.Region)
	for _, region := range set.Regions {
		if candidates[region.Name] == (Identity{}) {
			return Result{}, fmt.Errorf("regioncompare: candidate identity required for %q", region.Name)
		}
		if references[region.Name] == (Identity{}) {
			return Result{}, fmt.Errorf("regioncompare: reference identity required for %q", region.Name)
		}
		if !sameBoxOrNew(seenCandidates, candidates[region.Name], region) ||
			!sameBoxOrNew(seenReferences, references[region.Name], region) {
			return Result{}, fmt.Errorf("regioncompare: an identity names different rectangles")
		}
	}
	page := sheet.New(2)
	result := Result{Job: set.Job, Threshold: set.Threshold}
	for _, region := range set.Regions {
		candidate := candidates[region.Name]
		reference := references[region.Name]
		started := time.Now()
		drawn, err := source.Draw(set.Job, region, candidate)
		if err != nil {
			return Result{}, fmt.Errorf("regioncompare: draw %q: %w", region.Name, err)
		}
		if err := check(drawn, candidate, region); err != nil {
			return Result{}, err
		}
		if drawn.Rays <= 0 {
			return Result{}, fmt.Errorf("regioncompare: candidate for %q needs positive sampling", region.Name)
		}
		drawDuration := time.Since(started)
		page.Add(fmt.Sprintf("%s %d/side", region.Name, drawn.Rays), sheetView(drawn.View), 1)
		var repeatResults []RepeatResult
		first := *drawn.Light
		first.Pix = append([]frame.Pixel(nil), drawn.Light.Pix...)
		drawn.Light = &first
		for i := 1; i < repeats; i++ {
			started = time.Now()
			again, err := source.Draw(set.Job, region, candidate)
			if err != nil {
				return Result{}, fmt.Errorf("regioncompare: repeat %q: %w", region.Name, err)
			}
			if err := check(again, candidate, region); err != nil {
				return Result{}, err
			}
			if again.Rays != drawn.Rays {
				return Result{}, fmt.Errorf("regioncompare: repeat sampling differs for %q", region.Name)
			}
			duration := time.Since(started)
			difference, err := regionjudge.Compare(again.Light, drawn.Light, set.Threshold)
			if err != nil {
				return Result{}, fmt.Errorf("regioncompare: repeat %q: %w", region.Name, err)
			}
			repeatResults = append(repeatResults, RepeatResult{Duration: duration, Report: difference})
		}
		started = time.Now()
		if maker, ok := source.(interface {
			EnsureReference(string, regionset.Region, Identity) error
		}); ok {
			if err := maker.EnsureReference(set.Job, region, reference); err != nil {
				return Result{}, fmt.Errorf("regioncompare: prepare reference %q: %w", region.Name, err)
			}
		}
		kept, err := source.Reference(set.Job, region, reference)
		if err != nil {
			return Result{}, fmt.Errorf("regioncompare: reference %q: %w", region.Name, err)
		}
		if err := check(kept, reference, region); err != nil {
			return Result{}, err
		}
		referenceDuration := time.Since(started)
		if drawn.Rays <= 0 || kept.Rays <= drawn.Rays {
			return Result{}, fmt.Errorf("regioncompare: reference for %q needs more rays per side", region.Name)
		}
		report, err := regionjudge.Compare(drawn.Light, kept.Light, set.Threshold)
		if err != nil {
			return Result{}, fmt.Errorf("regioncompare: %q: %w", region.Name, err)
		}
		page.Add(fmt.Sprintf("%s ref %d/side", region.Name, kept.Rays), sheetView(kept.View), 1)
		result.Regions = append(result.Regions, RegionResult{Region: region,
			Candidate: candidate, Reference: reference, CandidateRays: drawn.Rays,
			ReferenceRays: kept.Rays, DrawDuration: drawDuration, ReferenceDuration: referenceDuration, Report: report, Repeats: repeatResults})
	}
	result.Sheet = page.Image()
	return result, nil
}

// sheetView enlarges small crops in whole-pixel blocks without blurring.
func sheetView(view image.Image) image.Image {
	const target, limit = 192, 8
	largest := max(view.Bounds().Dx(), view.Bounds().Dy())
	if largest <= 0 {
		return view
	}
	scale := min(max(target/largest, 1), limit)
	if scale == 1 {
		return view
	}
	return file.Grow(view, scale, scale)
}

// sameBoxOrNew records an identity and rejects its reuse for another box.
func sameBoxOrNew(seen map[Identity]regionset.Region, id Identity, region regionset.Region) bool {
	previous, found := seen[id]
	if !found {
		seen[id] = region
		return true
	}
	return previous.Left == region.Left && previous.Top == region.Top &&
		previous.Width == region.Width && previous.Height == region.Height
}

// isNil includes nil values held in interfaces.
func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// check verifies a picture's identity and dimensions.
func check(picture Picture, expected Identity, region regionset.Region) error {
	if picture.Identity != expected {
		return fmt.Errorf("regioncompare: identity differs for %q", region.Name)
	}
	if picture.Light == nil || picture.Light.W != region.Width || picture.Light.H != region.Height ||
		isNil(picture.View) || picture.View.Bounds().Dx() != region.Width || picture.View.Bounds().Dy() != region.Height {
		return fmt.Errorf("regioncompare: picture size differs for %q", region.Name)
	}
	return nil
}
