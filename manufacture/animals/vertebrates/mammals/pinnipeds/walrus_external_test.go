package pinnipeds

import (
	"bytes"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Exercise the complete external construction, including the field mesh,
// skin, face and tusks, and then independently build the default profile.
func TestPinnipedWalrusExternalSpecimen(t *testing.T) {
	raw := bytes.Replace(profileRaw, []byte("thoracic: 14"), []byte("thoracic: 15"), 1)
	raw = bytes.Replace(raw, []byte(`thoracic: {basis: read, source: "Pierce, Clack and Hutchinson 2011, Journal of Anatomy 219:502, PMC3196755", value: 14}`), []byte("thoracic: {basis: picked}"), 1)
	custom, err := Read("custom.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	standard, err := Read("default.yaml", profileRaw)
	if err != nil {
		t.Fatal(err)
	}
	globalSkull := walrusFrame.condyle
	var eyes [2]maths.Vec
	for index, book := range []*Book{custom, standard} {
		profile, ok := book.Get("walrus")
		if !ok {
			t.Fatal("missing walrus")
		}
		kind, err := NewWalrus(profile, "lying", "skin")
		if err != nil {
			t.Fatal(err)
		}
		rig := kind.rig
		parts, err := kind.Build()
		if err != nil {
			t.Fatal(err)
		}
		if len(parts) != walrusPartIndex(1, walrusHeadPartCount) {
			t.Fatalf("specimen %d has %d parts", index, len(parts))
		}
		hide, faces, closed := testWalrusMesh(t, parts[walrusHidePart])
		if !closed || len(hide) == 0 || len(faces) == 0 {
			t.Fatalf("specimen %d has no closed hide", index)
		}
		for sideIndex, side := range []float64{-1, 1} {
			for _, part := range []int{walrusEyePart, walrusNostrilPart} {
				at := parts[walrusPartIndex(sideIndex, part)].At(0).Origin
				nearest := math.Inf(1)
				for _, vertex := range hide {
					nearest = math.Min(nearest, vertex.Sub(at).Len())
				}
				if nearest > .05 {
					t.Fatalf("specimen %d side %d part %d is %.3f m from hide", index, sideIndex, part, nearest)
				}
			}
			tusk, _, closed := testWalrusMesh(t, parts[walrusPartIndex(sideIndex, walrusTuskPart)])
			if !closed || len(tusk) < 2 {
				t.Fatal("missing closed tusk")
			}
			canine := rig.body.forms["tusk "+walrusSideName(side)].(walrusCanine)
			if delta := tusk[len(tusk)-2].Sub(canine.at(0)).Len(); delta > 1e-9 {
				t.Fatalf("tusk root is %.6g m from its own skeleton", delta)
			}
		}
		eyes[index] = parts[walrusPartIndex(0, walrusEyePart)].At(0).Origin
	}
	if eyes[0].Sub(eyes[1]).Len() < .05 {
		t.Fatal("changed profile did not move the face")
	}
	if walrusFrame.condyle != globalSkull {
		t.Fatal("external specimen changed the default skull")
	}
}
