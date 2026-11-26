package manufacture

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// Distinct books must retain their own anatomy after the caller changes its
// source bytes and while both are used concurrently. Compare real animated
// geometry as well as bone construction; a changed but unused station is
// not a successful profile.
func TestWhaleBodyProfilesAreIndependentSnapshots(t *testing.T) {
	old := []byte("[0.70, 0.610, 0.560, 0.030]")
	wide := []byte("[0.70, 0.610, 0.600, 0.030]")
	changed := bytes.Replace(whaleProfilesRaw, old, wide, 1)
	if bytes.Equal(changed, whaleProfilesRaw) {
		t.Fatal("default body station was not present")
	}
	first, err := ReadWhaleProfiles("first.yaml", whaleProfilesRaw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadWhaleProfiles("second.yaml", changed)
	if err != nil {
		t.Fatal(err)
	}
	for i := range changed {
		changed[i] = 0
	}
	type result struct {
		made good
		err  error
	}
	var results [4]result
	var workers sync.WaitGroup
	for i, book := range []*WhaleProfiles{first, second, first, second} {
		workers.Add(1)
		go func() {
			defer workers.Done()
			row, err := book.NewOrca("swim", "")
			if err == nil {
				results[i].made, err = MakeAtTime(row, Sow(17), "quick", 750*time.Millisecond)
			}
			results[i].err = err
		}()
	}
	workers.Wait()
	for i, result := range results {
		if result.err != nil {
			t.Fatalf("book %d: %v", i, result.err)
		}
	}
	firstDigest := referenceDigest(t, results[0].made)
	secondDigest := referenceDigest(t, results[1].made)
	if firstDigest == secondDigest {
		t.Fatal("changed body stations produced identical animated geometry")
	}
	if referenceDigest(t, results[2].made) != firstDigest || referenceDigest(t, results[3].made) != secondDigest {
		t.Fatal("concurrent construction changed a profile's geometry")
	}
	firstPaint := results[0].made.native[0].Material.Texture
	secondPaint := results[1].made.native[0].Material.Texture
	differentPaint := false
	for yi := -30; yi <= 30 && !differentPaint; yi++ {
		for zi := 30; zi <= 70; zi++ {
			p := maths.V(.70, float64(yi)/100, float64(zi)/100)
			if firstPaint.At(p) != secondPaint.At(p) {
				differentPaint = true
				if firstPaint.At(p) != results[2].made.native[0].Material.Texture.At(p) ||
					secondPaint.At(p) != results[3].made.native[0].Material.Texture.At(p) {
					t.Fatal("concurrent construction changed a profile's paint")
				}
				break
			}
		}
	}
	if !differentPaint {
		t.Fatal("changed body stations did not reach the skin paint")
	}
	for i, book := range []*WhaleProfiles{first, second} {
		row, err := book.NewOrca("", "bones")
		if err != nil {
			t.Fatal(err)
		}
		made, err := MakeWithComplexity(row, Sow(17), "quick")
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			firstDigest = referenceDigest(t, made)
		} else if got := referenceDigest(t, made); got == firstDigest {
			t.Fatal("changed body stations did not reach the skeleton")
		}
	}
	row, _ := first.NewOrca("swim", "")
	again, err := MakeAtTime(row, Sow(17), "quick", 750*time.Millisecond)
	if err != nil || referenceDigest(t, again) != referenceDigest(t, results[0].made) {
		t.Fatalf("first book changed after the other was used: %v", err)
	}
}

func TestWhaleBodyStationsRejectInvalidProfiles(t *testing.T) {
	for _, tc := range []struct{ old, bad string }{
		{"[0.70, 0.610, 0.560, 0.030]", "[0.70, 0, 0.560, 0.030]"},
		{"[0.70, 0.610, 0.560, 0.030]", "[0.70, 0.610, .nan, 0.030]"},
		{"[0.70, 0.610, 0.560, 0.030]", "[-1.40, 0.610, 0.560, 0.030]"},
		{"[0.70, 0.610, 0.560, 0.030]", "[0.70, 0.610, 9, 0.030]"},
		{"[0.70, 0.610, 0.560, 0.030]", "[0.70, 2, 0.560, 0.030]"},
		{"[0.70, 0.610, 0.560, 0.030]", "[0.70, 0.610, 0.560, -2]"},
		{"[-2.60, 0.060, 0.030, 0.000]", "[-2.57, 0.060, 0.030, 0.000]"},
		{"[3.06, 0.010, 0.012, -0.050]", "[3.025, 0.010, 0.012, -0.050]"},
	} {
		raw := bytes.Replace(whaleProfilesRaw, []byte(tc.old), []byte(tc.bad), 1)
		if bytes.Equal(raw, whaleProfilesRaw) {
			t.Fatalf("missing fixture %s", tc.old)
		}
		if _, err := ReadWhaleProfiles("bad.yaml", raw); err == nil {
			t.Errorf("accepted invalid station %s", tc.bad)
		}
	}
}
