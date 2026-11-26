package job

import (
	"image"
	"testing"
)

func TestARegionNamesPixelsOfTheFullPicture(t *testing.T) {
	made := read(t, still+"film: {width: 100, height: 80, region: {left: 12, top: 7, width: 20, height: 15}}\n")
	got, said := made.Film().Region()
	if want := image.Rect(12, 7, 32, 22); !said || got != want {
		t.Fatalf("region is %v, %v; want %v, true", got, said, want)
	}
	if _, said := read(t, still).Film().Region(); said {
		t.Fatal("an absent region was said")
	}
}

// Anti: a malformed region or one outside an explicitly sized picture is refused.
func TestAnInvalidRegionIsRefused(t *testing.T) {
	for _, region := range []string{
		"{left: -1, top: 0, width: 2, height: 2}",
		"{left: 0, top: 0, width: 0, height: 2}",
		"{left: 0, top: 0, width: 2}",
		"{left: 99, top: 0, width: 2, height: 2}",
		"{left: 0, top: 79, width: 2, height: 2}",
	} {
		refused(t, still+"film: {width: 100, height: 80, region: "+region+"}\n", "region")
	}
}

func TestARegionOverrideStandsOverTheJobAndChangesLightIdentity(t *testing.T) {
	made := read(t, still+"film: {width: 100, height: 80, region: {left: 12, top: 7, width: 20, height: 15}}\n")
	want := image.Rect(2, 3, 22, 18)
	over, err := made.With(Region(want))
	if err != nil {
		t.Fatal(err)
	}
	if got, said := over.Film().Region(); !said || got != want {
		t.Fatalf("got %v, %v", got, said)
	}
	if got, _ := made.Film().Region(); got == want {
		t.Fatal("override changed the original")
	}
	if over.LightKey() == made.LightKey() {
		t.Fatal("different regions share a light key")
	}
	whole := read(t, still+"film: {width: 100, height: 80}\n")
	part, err := whole.With(Region(want))
	if err != nil {
		t.Fatal(err)
	}
	if whole.LightKey() == part.LightKey() {
		t.Fatal("whole and region share a light key")
	}
}

// Anti: the override cannot make an empty, reversed or out-of-frame region.
func TestAnInvalidRegionOverrideIsRefused(t *testing.T) {
	made := read(t, still+"film: {width: 100, height: 80}\n")
	for _, box := range []image.Rectangle{
		{Min: image.Pt(-1, 0), Max: image.Pt(1, 2)},
		{Min: image.Pt(5, 5), Max: image.Pt(5, 6)},
		{Min: image.Pt(5, 5), Max: image.Pt(4, 6)},
		{Min: image.Pt(99, 0), Max: image.Pt(101, 2)},
	} {
		if _, err := made.With(Region(box)); err == nil {
			t.Errorf("%v was taken", box)
		}
	}
}

func TestARegionCanWaitForTheSetsPictureSize(t *testing.T) {
	made := read(t, still+"film: {region: {left: 12, top: 7, width: 20, height: 15}}\n")
	if _, said := made.Film().Region(); !said {
		t.Fatal("region was lost")
	}
}

func TestARegionCanAskForAHigherReference(t *testing.T) {
	made := read(t, still+"film: {width: 100, height: 80, rays: 64, region: {left: 12, top: 7, width: 20, height: 15}}\n")
	if rays, said := made.Film().Rays(); !said || rays != 64 {
		t.Fatalf("rays are %d, %v; want 64, true", rays, said)
	}
}

// Anti: a region cannot exceed the same 64-ray limit as a whole picture.
func TestARegionCannotExceedTheRayLimit(t *testing.T) {
	refused(t, still+"film: {width: 100, height: 80, rays: 65, region: {left: 0, top: 0, width: 1, height: 1}}\n", "rays")
}

func TestAWholeStillCanAskForSixtyFourRays(t *testing.T) {
	made := read(t, still+"film: {width: 100, height: 80, rays: 64}\n")
	if rays, said := made.Film().Rays(); !said || rays != 64 {
		t.Fatalf("rays are %d, %v; want 64, true", rays, said)
	}
}

// Anti: a region is for a still, not a film.
func TestARegionOnAFilmIsRefused(t *testing.T) {
	refused(t, film+"film: {region: {left: 0, top: 0, width: 2, height: 2}}\n", "region")
}
