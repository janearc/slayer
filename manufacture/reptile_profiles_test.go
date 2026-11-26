package manufacture

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles/hatzegopteryx"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

func TestReptileProfilesSelectDistinctStudies(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	book, err := ReadReptileProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		species, study string
		valid          bool
	}{
		{"pliosaur", "", true}, {"pliosaur", "baseline", true}, {"pliosaur", "continuous", true},
		{"purussaurus", "", true}, {"purussaurus", "study", true},
		{"hatzegopteryx", "", true}, {"hatzegopteryx", "cage", true}, {"hatzegopteryx", "density", true},
		{"pliosaur", "cage", false}, {"purussaurus", "baseline", false}, {"hatzegopteryx", "continuous", false}, {"unknown", "", false},
	} {
		row, err := book.NewReptile(tc.species, tc.study)
		if (err == nil) != tc.valid {
			t.Errorf("%s %s: %v", tc.species, tc.study, err)
		}
		if err == nil && Deforms(row) {
			t.Errorf("%s %s is a fixed profile pose", tc.species, tc.study)
		}
	}
	var missing *ReptileProfiles
	if _, err := missing.NewPliosaur(""); err == nil {
		t.Fatal("nil profile book accepted")
	}
	if _, err := book.NewPliosaur("continuous"); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewPurussaurus(); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewHatzegopteryx("cage"); err != nil {
		t.Fatal(err)
	}
}

func TestReptileProfileControlsAreFrozenAndChangeGeometry(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	changed := bytes.Replace(raw, []byte("yaw: 0.15"), []byte("yaw: 0.4"), 1)
	if bytes.Equal(changed, raw) {
		t.Fatal("missing yaw fixture")
	}
	first, err := ReadReptileProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadReptileProfiles(file, changed)
	if err != nil {
		t.Fatal(err)
	}
	for i := range changed {
		changed[i] = 0
	}
	points := func(book *ReptileProfiles) []maths.Vec {
		row, err := book.NewPurussaurus()
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(row, Sow(1))
		if err != nil {
			t.Fatal(err)
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		return parts[0].Shape.(*shape.Mesh).Vertices
	}
	before, after := points(first), points(second)
	if reflect.DeepEqual(before, after) {
		t.Fatal("changing the profile did not change the posed geometry")
	}
	if !reflect.DeepEqual(after, points(second)) {
		t.Fatal("a later source mutation changed the frozen profile")
	}
}

func TestReptileProfilePlacesNativeParts(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	book, err := ReadReptileProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	row, err := book.NewPliosaur("baseline")
	if err != nil {
		t.Fatal(err)
	}
	made, err := Make(row, Sow(1))
	if err != nil {
		t.Fatal(err)
	}
	parent := maths.NewSubspace(maths.V(4, 5, 6), maths.About(maths.YAxis, .4), 1.3)
	parts, err := made.Things(parent)
	if err != nil || len(parts) != len(made.native) {
		t.Fatalf("parts: %d, %v", len(parts), err)
	}
	for i := range parts {
		if parts[i].Shape != made.native[i].Shape || parts[i].Material != made.native[i].Material {
			t.Fatalf("part %d was rebuilt", i)
		}
		for _, p := range []maths.Vec{maths.Vec{}, maths.XAxis, maths.YAxis, maths.ZAxis} {
			want := parent.Out(made.native[i].At(0).Out(p))
			if got := parts[i].At(0).Out(p); got.Sub(want).Len() > 1e-12 {
				t.Fatalf("part %d lost placement: %v != %v", i, got, want)
			}
		}
	}
}

func TestCrocodylianAnatomyAndScaleComeFromProfile(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	changed := bytes.Replace(raw, []byte("0.215, 0.24, 0.18, 0.34"), []byte("0.215, 0.28, 0.18, 0.34"), 1)
	changed = bytes.Replace(changed, []byte("scale: 2.8"), []byte("scale: 3.1"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("missing anatomy fixture")
	}
	first, err := ReadReptileProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadReptileProfiles(file, changed)
	if err != nil {
		t.Fatal(err)
	}
	makeBody := func(book *ReptileProfiles) (maths.Box, []maths.Vec) {
		row, err := book.NewPurussaurus()
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(row, Sow(1))
		if err != nil {
			t.Fatal(err)
		}
		parts, err := made.Things(maths.Local())
		if err != nil {
			t.Fatal(err)
		}
		return made.Bounds(), parts[0].Shape.(*shape.Mesh).Vertices
	}
	beforeBox, before := makeBody(first)
	afterBox, after := makeBody(second)
	if beforeBox == afterBox || reflect.DeepEqual(before, after) {
		t.Fatal("profile anatomy or scale was ignored")
	}
}

func TestPliosaurAnatomyComesFromProfile(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	for _, tc := range []struct{ study, old, replacement string }{
		{"baseline", "[0.25, 0, 1.02, 0.77]", "[0.25, 0, 1.15, 0.77]"},
		{"continuous", "[-0.24, 0, 0, 0.79, 0.26, 0.34]", "[-0.24, 0, 0, 0.79, 0.30, 0.34]"},
	} {
		t.Run(tc.study, func(t *testing.T) {
			changed := bytes.Replace(raw, []byte(tc.old), []byte(tc.replacement), 1)
			if bytes.Equal(raw, changed) {
				t.Fatal("missing anatomy fixture")
			}
			first, err := ReadReptileProfiles(file, raw)
			if err != nil {
				t.Fatal(err)
			}
			second, err := ReadReptileProfiles(file, changed)
			if err != nil {
				t.Fatal(err)
			}
			points := func(book *ReptileProfiles) []maths.Vec {
				row, err := book.NewPliosaur(tc.study)
				if err != nil {
					t.Fatal(err)
				}
				made, err := Make(row, Sow(1))
				if err != nil {
					t.Fatal(err)
				}
				switch body := made.native[0].Shape.(type) {
				case *mesh.Mesh:
					return body.Points()
				case *shape.Mesh:
					return body.Vertices
				default:
					t.Fatalf("unexpected body %T", body)
					return nil
				}
			}
			if reflect.DeepEqual(points(first), points(second)) {
				t.Fatal("profile anatomy was ignored")
			}
		})
	}
}

func TestPliosaurFinAndEyeComeFromProfile(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	changed := bytes.Replace(raw, []byte("[0.95, 0.59, 0.12, -0.06]"), []byte("[0.95, 0.70, 0.12, -0.06]"), 1)
	changed = bytes.Replace(changed, []byte("eye: [2.67, 0.14, 0.37, 0.045]"), []byte("eye: [2.60, 0.14, 0.37, 0.045]"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("missing fin or eye fixture")
	}
	first, err := ReadReptileProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadReptileProfiles(file, changed)
	if err != nil {
		t.Fatal(err)
	}
	makeParts := func(book *ReptileProfiles) good {
		row, err := book.NewPliosaur("baseline")
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(row, Sow(1))
		if err != nil {
			t.Fatal(err)
		}
		return made
	}
	a, b := makeParts(first), makeParts(second)
	if reflect.DeepEqual(a.native[1].Shape.(*mesh.Mesh).Points(), b.native[1].Shape.(*mesh.Mesh).Points()) {
		t.Fatal("fin section did not alter fin geometry")
	}
	if a.native[5].At(0) == b.native[5].At(0) {
		t.Fatal("eye placement did not change")
	}
}

func TestCrocodylianLimbsComeFromProfile(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	changed := bytes.Replace(raw, []byte("[-0.04, -0.16, 0.40]"), []byte("[-0.04, -0.16, 0.45]"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("missing limb fixture")
	}
	first, err := ReadReptileProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadReptileProfiles(file, changed)
	if err != nil {
		t.Fatal(err)
	}
	points := func(book *ReptileProfiles) []maths.Vec {
		row, err := book.NewPurussaurus()
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(row, Sow(1))
		if err != nil {
			t.Fatal(err)
		}
		return made.native[0].Shape.(*shape.Mesh).Vertices
	}
	if reflect.DeepEqual(points(first), points(second)) {
		t.Fatal("profile limb geometry was ignored")
	}
}

func TestPterosaurStudiesReadTheirOwnBodyAnatomy(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	for _, tc := range []struct{ study, old, replacement string }{
		{"composite", "[-0.35, 2.68, 0, 0.45, 0.36]", "[-0.35, 2.68, 0, 0.55, 0.36]"},
		{"cage", "[-0.28, 2.68, 0.43, 0.34]", "[-0.28, 2.68, 0.50, 0.34]"},
		{"density", "[-0.80, 0, 0, 0.35, 0.20, 0.16, 0.95]", "[-0.80, 0, 0, 0.40, 0.20, 0.16, 0.95]"},
	} {
		t.Run(tc.study, func(t *testing.T) {
			changed := bytes.Replace(raw, []byte(tc.old), []byte(tc.replacement), 1)
			if bytes.Equal(raw, changed) {
				t.Fatal("missing anatomy fixture")
			}
			first, err := ReadReptileProfiles(file, raw)
			if err != nil {
				t.Fatal(err)
			}
			second, err := ReadReptileProfiles(file, changed)
			if err != nil {
				t.Fatal(err)
			}
			points := func(book *ReptileProfiles) []maths.Vec {
				row, err := book.NewHatzegopteryx(tc.study)
				if err != nil {
					t.Fatal(err)
				}
				made, err := Make(row, Sow(1))
				if err != nil {
					t.Fatal(err)
				}
				return made.native[0].Shape.(*mesh.Mesh).Points()
			}
			if reflect.DeepEqual(points(first), points(second)) {
				t.Fatal("study ignored its profile anatomy")
			}
		})
	}
}

func TestPterosaurWingRigComesFromProfile(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	changed := bytes.Replace(raw, []byte("wing-lengths: [1.35, 1.45, 2.85]"), []byte("wing-lengths: [1.40, 1.45, 2.85]"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("missing wing fixture")
	}
	first, err := ReadReptileProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadReptileProfiles(file, changed)
	if err != nil {
		t.Fatal(err)
	}
	points := func(book *ReptileProfiles) []maths.Vec {
		row, err := book.NewHatzegopteryx("composite")
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(row, Sow(1))
		if err != nil {
			t.Fatal(err)
		}
		return made.native[3].Shape.(*mesh.Mesh).Points()
	}
	if reflect.DeepEqual(points(first), points(second)) {
		t.Fatal("wing rig ignored its profile")
	}
}

func TestPterosaurCageRigComesFromProfile(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	changed := bytes.Replace(raw, []byte("[-1.00, 1.55, 0.60]"), []byte("[-0.90, 1.55, 0.60]"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("missing cage leg fixture")
	}
	first, err := ReadReptileProfiles(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadReptileProfiles(file, changed)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := first.book.Get("hatzegopteryx")
	b, _ := second.book.Get("hatzegopteryx")
	if reflect.DeepEqual(hatzegopteryx.CageLegRig(a, .5, 1), hatzegopteryx.CageLegRig(b, .5, 1)) {
		t.Fatal("cage leg ignored its profile")
	}
}

func TestContinuousPliosaurHeadAndFinsComeFromProfile(t *testing.T) {
	file, raw := EmbeddedReptileBook()
	changes := [][2]string{
		{"continuous-head: [0.66, 1.33, 0.035, 0.085, 0.025, 0.105, 0.036]", "continuous-head: [0.66, 1.33, 0.045, 0.085, 0.025, 0.105, 0.036]"},
		{"continuous-fins: [0.18, -0.72, 0.20, 0.90, 0.17, 0.23]", "continuous-fins: [0.18, -0.72, 0.20, 0.90, 0.20, 0.23]"},
	}
	points := func(data []byte) []maths.Vec {
		t.Helper()
		book, err := ReadReptileProfiles(file, data)
		if err != nil {
			t.Fatal(err)
		}
		row, err := book.NewPliosaur("continuous")
		if err != nil {
			t.Fatal(err)
		}
		made, err := Make(row, Sow(1))
		if err != nil {
			t.Fatal(err)
		}
		return made.native[0].Shape.(*shape.Mesh).Vertices
	}
	original := points(raw)
	for _, change := range changes {
		altered := bytes.Replace(raw, []byte(change[0]), []byte(change[1]), 1)
		if bytes.Equal(altered, raw) {
			t.Fatalf("missing fixture %q", change[0])
		}
		if reflect.DeepEqual(original, points(altered)) {
			t.Errorf("profile change did not affect geometry: %s", change[1])
		}
	}
}
