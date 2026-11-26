package snakes

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

// Even positive gaps can round to coincident subdivided rings. Refuse them
// while checking the book, before a renderer tries to create a degenerate face.
func TestSnakeProfileRefusesSubresolutionStations(t *testing.T) {
	p := testProfile(t, "eunectes-murinus")
	p.Stations[1].At = math.Nextafter(p.Stations[0].At, 1)
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "stations[1].at") {
		t.Fatalf("subresolution station gap accepted: %v", err)
	}
	if _, err := Extent(p); err == nil {
		t.Fatal("extent accepted subresolution station gap")
	}
	if _, err := Build(p); err == nil || !strings.Contains(err.Error(), "stations[1].at") {
		t.Fatalf("build did not refuse station gap before meshing: %v", err)
	}
	_, raw := Embedded()
	old := []byte("at: 0.040")
	if !bytes.Contains(raw, old) {
		t.Fatal("missing station fixture")
	}
	bad := bytes.Replace(raw, old, []byte("at: 0.00000000000000001"), 1)
	if _, err := Read("too-close.yaml", bad); err == nil || !strings.Contains(err.Error(), "stations[1].at") {
		t.Fatalf("book accepted station gap: %v", err)
	}
}

// The renderer has to be able to place each eye within the head and behind
// the snout. These bounds apply to the representation, not to snake biology.
func TestSnakeProfileRefusesEyesTooLargeForTheHead(t *testing.T) {
	p := testProfile(t, "eunectes-murinus")
	p.Head.EyeSize = .03
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "head.eyesize") {
		t.Fatalf("oversized eye accepted: %v", err)
	}
	p = testProfile(t, "eunectes-murinus")
	p.Head.EyeAt = 1
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "head.eyesize") {
		t.Fatalf("eye beyond snout accepted: %v", err)
	}
	p = testProfile(t, "eunectes-murinus")
	p.Head.EyeAt = p.Stations[len(p.Stations)-2].At - p.Head.EyeSize
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "head.eyesize") {
		t.Fatalf("eye crossing terminal taper accepted: %v", err)
	}
}

func TestSnakeMeshHandlesSmallestPermittedStationSpacing(t *testing.T) {
	for _, length := range []float64{.001, 50} {
		for _, radius := range []float64{.0001, .25} {
			p := testProfile(t, "eunectes-murinus")
			p.Length = length
			p.Head.EyeSize = .00001
			p.Stations[1].At = stationGap
			for i := range p.Stations {
				p.Stations[i].Width, p.Stations[i].Height = radius, radius
			}
			if err := p.Validate(); err != nil {
				t.Fatalf("length %g radius %g: %v", length, radius, err)
			}
			if _, err := Build(p); err != nil {
				t.Fatalf("length %g radius %g: %v", length, radius, err)
			}
		}
	}
}
