package regioncompare

import (
	"errors"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/regionset"
)

func TestSmallSheetViewsEnlargeWholePixels(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 32, 32))
	source.Set(3, 4, color.RGBA{R: 200, G: 20, B: 10, A: 255})
	view := sheetView(source)
	if view.Bounds().Dx() != 192 || view.Bounds().Dy() != 192 {
		t.Fatalf("small view is %v", view.Bounds())
	}
	for y := 24; y < 30; y++ {
		for x := 18; x < 24; x++ {
			if got := color.RGBAModel.Convert(view.At(x, y)).(color.RGBA); got != (color.RGBA{R: 200, G: 20, B: 10, A: 255}) {
				t.Fatalf("pixel at %d,%d changed to %v", x, y, got)
			}
		}
	}
}

// Anti: large views are not enlarged and tiny ones have a bounded scale.
func TestSheetViewBoundsItsEnlargement(t *testing.T) {
	large := image.NewRGBA(image.Rect(0, 0, 320, 300))
	if got := sheetView(large); got.Bounds() != large.Bounds() {
		t.Fatalf("large view changed: %v", got.Bounds())
	}
	tiny := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if got := sheetView(tiny); got.Bounds().Dx() != 8 || got.Bounds().Dy() != 8 {
		t.Fatalf("tiny view: %v", got.Bounds())
	}
}

const exampleSet = `job: shot.yaml
threshold: 1
regions:
  - name: water
    observation: bright points
    left: 1
    top: 0
    width: 3
    height: 3
`

type sourceStub struct {
	wrong, fail, badView, badLight, sameRays bool
	draws, reads                             int
	swapped                                  map[string]Identity
}

type changingSource struct {
	sourceStub
	shared       *frame.Frame
	sharedView   *image.RGBA
	badRepeat    bool
	failedRepeat bool
}

func (s *changingSource) Draw(_ string, region regionset.Region, id Identity) (Picture, error) {
	s.draws++
	if s.draws > 1 && s.failedRepeat {
		return Picture{}, errors.New("repeat failed")
	}
	if s.shared == nil {
		s.shared = frame.New(region.Width, region.Height)
		s.sharedView = image.NewRGBA(image.Rect(0, 0, region.Width, region.Height))
	}
	for i := range s.shared.Pix {
		s.shared.Pix[i].R = float32(2 * (s.draws - 1))
	}
	for y := 0; y < region.Height; y++ {
		for x := 0; x < region.Width; x++ {
			s.sharedView.SetRGBA(x, y, color.RGBA{R: uint8(100 * (s.draws - 1)), A: 255})
		}
	}
	rays := 4
	if s.draws > 1 && s.badRepeat {
		rays = 2
	}
	return Picture{Identity: id, Light: s.shared, View: s.sharedView, Rays: rays}, nil
}

// Anti: a failed or differently sampled repeat produces no partial result.
func TestRepeatedDrawFailureReturnsNoPartialResult(t *testing.T) {
	for _, source := range []*changingSource{{badRepeat: true}, {failedRepeat: true}} {
		got, err := RunRepeated(strings.NewReader(exampleSet), 4, 3, map[string]Identity{"water": {1}},
			map[string]Identity{"water": {2}}, source, 2)
		if err == nil || got.Sheet != nil || got.Regions != nil || source.reads != 0 {
			t.Fatalf("result %+v, error %v, source %+v", got, err, source)
		}
	}
}

func TestRepeatedDrawsMeasureRawVariation(t *testing.T) {
	source := &changingSource{}
	got, err := RunRepeated(strings.NewReader(exampleSet), 4, 3, map[string]Identity{"water": {1}},
		map[string]Identity{"water": {2}}, source, 2)
	if err != nil {
		t.Fatal(err)
	}
	if source.draws != 2 || len(got.Regions) != 1 || len(got.Regions[0].Repeats) != 1 {
		t.Fatalf("draws %d, result %+v", source.draws, got)
	}
	variation := got.Regions[0].Repeats[0].Report
	if variation.Signed.R != 2 || variation.MeanAbsolute != 2.0/3 || got.Regions[0].Report.MeanAbsolute != 0 {
		t.Fatalf("variation %+v, first comparison %+v", variation, got.Regions[0].Report)
	}
	if pixel := color.RGBAModel.Convert(got.Sheet.At(12, 12)).(color.RGBA); pixel.R != 0 || pixel.A != 255 {
		t.Fatalf("sheet did not retain the first view: %v", pixel)
	}
}

// Anti: invalid repetition counts are refused before any draw.
func TestRepeatCountIsBoundedBeforeDrawing(t *testing.T) {
	for _, repeats := range []int{0, 9} {
		source := &sourceStub{}
		_, err := RunRepeated(strings.NewReader(exampleSet), 4, 3, map[string]Identity{"water": {1}},
			map[string]Identity{"water": {2}}, source, repeats)
		if err == nil || source.draws != 0 {
			t.Fatalf("repeats %d: %v, %d draws", repeats, err, source.draws)
		}
	}
}

type makingSource struct {
	sourceStub
	made     int
	failMake bool
}

func (s *makingSource) EnsureReference(_ string, _ regionset.Region, _ Identity) error {
	s.made++
	if s.failMake {
		return errors.New("reference unavailable")
	}
	return nil
}

func TestRunPreparesReferenceAfterValidCandidate(t *testing.T) {
	for _, tc := range []struct {
		name               string
		wrong, failMake    bool
		wantMade, wantRead int
		wantErr            bool
	}{
		{"success", false, false, 1, 1, false},
		{"invalid candidate", true, false, 0, 0, true},
		{"prepare fails", false, true, 1, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &makingSource{sourceStub: sourceStub{wrong: tc.wrong}, failMake: tc.failMake}
			result, err := Run(strings.NewReader(exampleSet), 4, 3, map[string]Identity{"water": {1}}, map[string]Identity{"water": {2}}, source)
			if (err != nil) != tc.wantErr || source.made != tc.wantMade || source.reads != tc.wantRead || (tc.wantErr && result.Sheet != nil) {
				t.Fatalf("result %+v, error %v, source %+v", result, err, source)
			}
		})
	}
}

func (s *sourceStub) picture(region regionset.Region, id Identity) (Picture, error) {
	if s.fail {
		return Picture{}, errors.New("identity cannot be verified")
	}
	if s.wrong {
		id = Identity{99}
	}
	light := frame.New(region.Width, region.Height)
	view := image.NewRGBA(image.Rect(0, 0, region.Width, region.Height))
	if s.badLight {
		light = frame.New(1, 1)
	}
	if s.badView {
		view = image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	return Picture{Identity: id, Light: light, View: view, Rays: 4}, nil
}

func (s *sourceStub) Draw(_ string, region regionset.Region, id Identity) (Picture, error) {
	s.draws++
	if other, ok := s.swapped[region.Name]; ok {
		id = other
	}
	return s.picture(region, id)
}

func (s *sourceStub) Reference(_ string, region regionset.Region, id Identity) (Picture, error) {
	s.reads++
	picture, err := s.picture(region, id)
	if !s.sameRays {
		picture.Rays = 16
	}
	return picture, err
}

func TestRunComparesAndLaysOutThePinnedPair(t *testing.T) {
	source := &sourceStub{}
	candidate, reference := Identity{1}, Identity{2}
	got, err := Run(strings.NewReader(exampleSet), 4, 3, map[string]Identity{"water": candidate},
		map[string]Identity{"water": reference}, source)
	if err != nil || len(got.Regions) != 1 || got.Sheet == nil ||
		got.Regions[0].Candidate != candidate || got.Regions[0].Reference != reference ||
		got.Regions[0].Region.Name != "water" || got.Threshold != 1 || got.Job != "shot.yaml" ||
		got.Regions[0].CandidateRays != 4 || got.Regions[0].ReferenceRays != 16 ||
		source.draws != 1 || source.reads != 1 {
		t.Fatalf("result %+v, %v, source %+v", got, err, source)
	}
}

func TestRunRefusesUnverifiedOrMismatchedPictures(t *testing.T) {
	for name, source := range map[string]*sourceStub{
		"unverified": {fail: true}, "wrong identity": {wrong: true},
		"wrong light size": {badLight: true}, "wrong view size": {badView: true},
		"reference not finer": {sameRays: true},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := Run(strings.NewReader(exampleSet), 4, 3, map[string]Identity{"water": {1}},
				map[string]Identity{"water": {2}}, source)
			if err == nil || got.Sheet != nil || got.Regions != nil {
				t.Fatalf("partial or unverified result %+v, %v", got, err)
			}
		})
	}
}

func TestRunRequiresEveryIdentityBeforeDrawing(t *testing.T) {
	for name, candidate := range map[string]Identity{"missing candidate": {}, "candidate": {1}} {
		source := &sourceStub{}
		got, err := Run(strings.NewReader(exampleSet), 4, 3, map[string]Identity{"water": candidate}, nil, source)
		if err == nil || got.Sheet != nil || source.draws != 0 || source.reads != 0 {
			t.Fatalf("%s: %+v, %v, %+v", name, got, err, source)
		}
	}
}

func TestRunRejectsMalformedSetBeforeDrawing(t *testing.T) {
	source := &sourceStub{}
	got, err := Run(strings.NewReader("job: shot.yaml"), 4, 3, map[string]Identity{"water": {1}},
		map[string]Identity{"water": {2}}, source)
	if err == nil || got.Sheet != nil || source.draws != 0 {
		t.Fatalf("invalid set: %+v, %v", got, err)
	}
}

func TestRunRefusesSwappedSameSizeRegions(t *testing.T) {
	two := exampleSet + `  - name: wall
    observation: red points
    left: 4
    top: 0
    width: 3
    height: 3
`
	source := &sourceStub{swapped: map[string]Identity{"water": {2}, "wall": {1}}}
	got, err := Run(strings.NewReader(two), 7, 3,
		map[string]Identity{"water": {1}, "wall": {2}},
		map[string]Identity{"water": {3}, "wall": {4}}, source)
	if err == nil || got.Sheet != nil || got.Regions != nil {
		t.Fatalf("accepted swapped identities: %+v, %v", got, err)
	}
}

type sharedReferenceSource struct{ changingSource }

func (s *sharedReferenceSource) Reference(_ string, region regionset.Region, id Identity) (Picture, error) {
	for i := range s.shared.Pix {
		s.shared.Pix[i].R = 3
	}
	return Picture{Identity: id, Light: s.shared, View: s.sharedView, Rays: 8}, nil
}

func TestSingleDrawSurvivesReferenceBufferReuse(t *testing.T) {
	source := &sharedReferenceSource{}
	got, err := Run(strings.NewReader(exampleSet), 4, 3, map[string]Identity{"water": {1}},
		map[string]Identity{"water": {2}}, source)
	if err != nil {
		t.Fatal(err)
	}
	if score := got.Regions[0].Report; score.Signed.R != -3 || score.MeanAbsolute != 1 {
		t.Fatalf("candidate was changed by reference: %+v", score)
	}
}
