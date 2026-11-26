package job

import (
	"fmt"
	"testing"
)

func TestExactSamplesAreAJobSettingAndPartOfItsKey(t *testing.T) {
	var previous string
	for _, count := range []int{1, 4, 7, 16, 25, 64} {
		made := read(t, still+fmt.Sprintf("film: {samples: %d}\n", count))
		if got, said := made.Film().Samples(); !said || got != count {
			t.Fatalf("samples = %d, %t; want %d", got, said, count)
		}
		if key := made.Key(); key == previous {
			t.Fatalf("%d samples reused the preceding job's key", count)
		} else {
			previous = key
		}
	}
}

func TestExactSamplesRefuseAmbiguityAndUnsupportedModes(t *testing.T) {
	for _, suffix := range []string{
		"film: {samples: 0}\n", "film: {samples: 65}\n",
		"film: {samples: 7, rays: 2}\n", "film: {samples: 7, mode: wire}\n",
		"film: {samples: 7, aperture: 1, focus: 2}\n",
		"film: {samples: 7, region: {left: 0, top: 0, width: 1, height: 1}}\n",
	} {
		refused(t, still+suffix)
	}
	refused(t, film+"film: {samples: 7}\n", "stills only")
	made := read(t, still+"film: {samples: 7}\n")
	over, err := made.With(Rays(2))
	if err != nil {
		t.Fatal(err)
	}
	if _, said := over.Film().Samples(); said {
		t.Fatal("rays override left exact samples in the job")
	}
	over, err = over.With(Samples(7))
	if err != nil {
		t.Fatal(err)
	}
	if _, said := over.Film().Rays(); said {
		t.Fatal("samples override left rays in the job")
	}
}
