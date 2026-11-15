package live

import (
	"os"
	"path/filepath"
	"testing"
)

// each kind of change to a job costs what the design note's table says.
func TestCompareCostsEachKindOfChange(t *testing.T) {
	for _, each := range []struct {
		name, was, is string
		want          Change
	}{
		{"film size", stillJob,
			jobLine + "film: {width: 80, height: 60, rays: 1}\n", Redraw},
		{"rays", stillJob,
			jobLine + "film: {width: 40, height: 30, rays: 2}\n", Redraw},
		{"print", stillJob, stillJob + "print: {knee: 0.5, top: 4}\n",
			Redevelop},
		{"stock", stillJob, stillJob + "stock: {name: lantern}\n",
			Redevelop},
		{"titles", stillJob, stillJob + "titles: {elements: {a: " +
			"{kind: text, words: hi, ink: ink, across: 0.5, " +
			"down: 0.5}}}\n", Redevelop},
		{"film exposure", stillJob, jobLine + "film: {width: 40, " +
			"height: 30, rays: 1, exposure: 2}\n", Restage},
		{"camera", stillJob, stillJob + "camera: {name: near}\n",
			Restage},
		{"moment", stillJob, stillJob + "at: {value: 3, unit: s}\n",
			Restage},
		{"scene", stillJob, "$schema: https://github.com/janearc/" +
			"slayer/schemas/job-v1.json\nscene: {smoke: 2}\n" +
			"film: {width: 40, height: 30, rays: 1}\n", Restage},
		{"renderer", stillJob, stillJob + "renderer: cpu\n", Restage},
		{"still to film", stillJob, filmJob, Restage},
	} {
		got := Compare(readJob(t, each.was), readJob(t, each.is))
		if got != each.want {
			t.Errorf("%s: %v, want %v", each.name, got, each.want)
		}
	}
}

// anti-test of TestCompareCostsEachKindOfChange
// a job compared with itself, or with one that differs only
// in its outputs or its sound, which the preview does not draw, costs
// nothing; and two changes cost the dearer, not the first.
func TestCompareLeavesAloneWhatIsNotDrawn(t *testing.T) {
	same := readJob(t, filmJob)
	if got := Compare(same, readJob(t, filmJob)); got != Unchanged {
		t.Errorf("the same job costs %v", got)
	}
	outputs := readJob(t, stillJob+"outputs: {picture: elsewhere.png}\n")
	if got := Compare(readJob(t, stillJob), outputs); got != Unchanged {
		t.Errorf("another output costs %v", got)
	}
	sound := readJob(t, filmJob+"sound: {tune: a.json, voicing: b}\n")
	if got := Compare(same, sound); got != Unchanged {
		t.Errorf("sound costs %v", got)
	}
	both := readJob(t, jobLine+"film: {width: 80, height: 30, rays: 1}\n"+
		"print: {knee: 0.5, top: 4}\n")
	if got := Compare(readJob(t, stillJob), both); got != Redraw {
		t.Errorf("a size and a print cost %v, want redraw", got)
	}
}

// a change has a name, and the dearer of two is what both need.
func TestChangeNamesAndFolds(t *testing.T) {
	if Redraw.String() != "redraw" || Restage.With(Redevelop) != Restage ||
		Unchanged.With(Redraw) != Redraw {
		t.Error("changes do not name or fold as they should")
	}
	if Change(-one).String() != "unknown" ||
		(Restage+one).String() != "unknown" {
		t.Error("a change that is none of them has a name")
	}
}

// a file the job reads, changed on disk, costs what its key would: the
// set restages, a theme develops again, a tune costs nothing.
func TestCompareCostsTheFilesTheJobReads(t *testing.T) {
	directory := t.TempDir()
	tune := filepath.Join(directory, "a.json")
	write := func(text string) {
		if err := os.WriteFile(tune, []byte(text), fileMode); err != nil {
			t.Fatal(err)
		}
	}
	text := filmJob + "sound: {tune: " + tune + ", voicing: b}\n"
	write("one")
	was := readJob(t, text)
	was.Key() // the digests are taken when first asked for
	write("two")
	if got := Compare(was, readJob(t, text)); got != Unchanged {
		t.Errorf("a tune changed on disk costs %v", got)
	}
}

// anti-test of TestCompareCostsTheFilesTheJobReads
// the set's own file changed on disk is not left alone: the
// scene changed, so it restages.
func TestCompareRestagesAChangedSet(t *testing.T) {
	directory := t.TempDir()
	set := filepath.Join(directory, "set.yaml")
	text := "$schema: https://github.com/janearc/slayer/schemas/" +
		"job-v1.json\nscene: {set: " + set + "}\n"
	if err := os.WriteFile(set, []byte("one"), fileMode); err != nil {
		t.Fatal(err)
	}
	was := readJob(t, text)
	was.Key() // the digests are taken when first asked for
	if err := os.WriteFile(set, []byte("two"), fileMode); err != nil {
		t.Fatal(err)
	}
	if got := Compare(was, readJob(t, text)); got != Restage {
		t.Errorf("a set changed on disk costs %v", got)
	}
}
