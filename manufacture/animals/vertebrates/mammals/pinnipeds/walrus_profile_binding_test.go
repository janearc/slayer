package pinnipeds

import (
	"bytes"
	"github.com/janearc/slayer/maths"
	"testing"
)

func TestPinnipedProfilesDriveEachSkeleton(t *testing.T) {
	book, err := Read("pinnipeds.yaml", profileRaw)
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := book.Get("walrus")
	if !ok {
		t.Fatal("missing walrus")
	}
	walrus := newWalrusBones(profile)
	if walrus.fault != nil {
		t.Fatal(walrus.fault)
	}
	if walrus.tree.Count("thoracic ") != 14 || walrus.tree.Count("rib ") != 28 || profile.AnkleForward != 2.6 {
		t.Fatal("walrus anatomy lost")
	}
	if _, ok := walrus.tree.Find("tusk left"); !ok {
		t.Fatal("walrus has no tusk")
	}
}

func TestPinnipedWalrusUsesOneExternalRig(t *testing.T) {
	raw := bytes.Replace(profileRaw, []byte("thoracic: 14"), []byte("thoracic: 15"), 1)
	raw = bytes.Replace(raw, []byte(`thoracic: {basis: read, source: "Pierce, Clack and Hutchinson 2011, Journal of Anatomy 219:502, PMC3196755", value: 14}`), []byte(`thoracic: {basis: picked}`), 1)
	book, err := Read("changed.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := book.Get("walrus")
	if !ok {
		t.Fatal("missing walrus")
	}
	row, err := NewWalrus(profile, "lying", "skin")
	if err != nil {
		t.Fatal(err)
	}
	rig := row.rig
	if rig.body.tree.Count("thoracic ") != 15 || rig.frame.condyle == walrusFrame.condyle {
		t.Fatal("external skull did not follow its skeleton")
	}
	canine := rig.body.forms["tusk left"].(walrusCanine)
	if canine.frame.condyle != rig.frame.condyle {
		t.Fatal("tusk is attached to another skeleton")
	}
	if _, _, err := rig.skin(walrusLying, 0, 0); err != nil {
		t.Fatal(err)
	}
}

func TestWalrusOutlineBooksKeepIndependentGeometry(t *testing.T) {
	changed := bytes.Replace(profileRaw, []byte("[1.30, .502, .915, -.080, 2.2]"), []byte("[1.30, .522, .915, -.080, 2.2]"), 1)
	if bytes.Equal(changed, profileRaw) {
		t.Fatal("missing walrus station fixture")
	}
	a, err := Read("original.yaml", profileRaw)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Read("changed.yaml", changed)
	if err != nil {
		t.Fatal(err)
	}
	firstProfile, _ := a.Get("walrus")
	first, err := NewWalrus(firstProfile, "lying", "skin")
	if err != nil {
		t.Fatal(err)
	}
	secondProfile, _ := b.Get("walrus")
	second, err := NewWalrus(secondProfile, "lying", "skin")
	if err != nil {
		t.Fatal(err)
	}
	field := func(row *Walrus) (func(maths.Vec) float64, maths.Vec) {
		t.Helper()
		rig := row.rig
		outline, err := rig.outline()
		if err != nil {
			t.Fatal(err)
		}
		masses, err := rig.walrusMasses(outline)
		if err != nil {
			t.Fatal(err)
		}
		point := maths.V(walrusNoseAt(rig.frame)-1.30, walrusFloor+(.915-.080)/2, .50)
		return walrusVolumeField(outline, masses), point
	}
	before, sample := field(first)
	want := before(sample)
	other, otherSample := field(second)
	if got := other(otherSample); got == want {
		t.Fatal("changed outline made the original volume field")
	}
	again, againSample := field(first)
	if got := again(againSample); got != want || before(sample) != want {
		t.Fatal("second book changed the first recipe")
	}
}

// A retained recipe must own the authored outline even when its caller
// changes a returned profile, and separate recipes must not share the cache.
func TestWalrusRecipeRetainsIndependentOutlineSnapshots(t *testing.T) {
	original := defaultPinniped("walrus")
	changed := defaultPinniped("walrus")
	changed.WalrusOutline[8][1] += .01
	a, err := NewWalrus(original, "", "bones")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewWalrus(changed, "", "bones")
	if err != nil {
		t.Fatal(err)
	}
	original.WalrusOutline[8][1] += .7
	changed.WalrusOutline[8][1] += .7
	for _, row := range []*Walrus{a, b} {
		if _, err := row.rig.outline(); err != nil {
			t.Fatal(err)
		}
	}
	measure := func(row *Walrus) float64 {
		outline, _ := row.rig.outline() // Both cached outlines were checked above.
		return outline.at(walrusNoseAt(row.rig.frame) - row.rig.body.profile.WalrusOutline[8][0]).wide
	}
	wantA, wantB := measure(a), measure(b)
	if wantA == wantB {
		t.Fatal("different profile outlines made identical recipes")
	}
	if a.rig.body.profile.WalrusOutline[8][1] == original.WalrusOutline[8][1] || b.rig.body.profile.WalrusOutline[8][1] == changed.WalrusOutline[8][1] {
		t.Fatal("a retained recipe shares caller-owned profile data")
	}
	// Exercise the same retained rigs concurrently through the skin, field,
	// texture and skeleton paths used to construct each representation.
	sample := func(row *Walrus) (maths.Vec, error) {
		skin, _, err := row.rig.skin(walrusLying, 0, 0)
		if err != nil {
			return maths.Vec{}, err
		}
		outline, err := row.rig.outline()
		if err != nil {
			return maths.Vec{}, err
		}
		masses, err := row.rig.walrusMasses(outline)
		if err != nil {
			return maths.Vec{}, err
		}
		field, err := row.rig.walrusField(outline, masses, true)
		if err != nil {
			return maths.Vec{}, err
		}
		if _, err := walrusSkeletonThings(row.rig.body); err != nil {
			return maths.Vec{}, err
		}
		point := maths.V(0, .3, .1)
		_ = (walrusHideTexture{rig: row.rig}).At(point)
		posed := skin(point)
		posed.X += field(point)
		return posed, nil
	}
	wantSkinA, err := sample(a)
	if err != nil {
		t.Fatal(err)
	}
	wantSkinB, err := sample(b)
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		outline [2]float64
		skin    [2]maths.Vec
		err     error
	}
	const workers = 8
	results := make(chan result, workers)
	for i := 0; i < workers; i++ {
		go func() {
			first, err := sample(a)
			if err != nil {
				results <- result{err: err}
				return
			}
			second, err := sample(b)
			results <- result{outline: [2]float64{measure(a), measure(b)}, skin: [2]maths.Vec{first, second}, err: err}
		}()
	}
	for i := 0; i < workers; i++ {
		got := <-results
		if got.err != nil {
			t.Fatal(got.err)
		}
		if got.outline != [2]float64{wantA, wantB} || got.skin != [2]maths.Vec{wantSkinA, wantSkinB} {
			t.Fatalf("recipe cache changed: %v", got)
		}
	}
}
