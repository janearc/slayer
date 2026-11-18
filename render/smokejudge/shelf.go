package smokejudge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/render/compare"
	cache "github.com/janearc/slayer/render/smokecache"
	"github.com/janearc/slayer/scene/smoke"
)

// shelf is the cache as one run uses it: where it is, this run's part of
// the key, or why nothing can be served or kept, and how many smokes were
// drawn and how many served.
type shelf struct {
	root   string
	here   cache.Key
	off    string // why nothing is served or kept, or "" when it is on
	fresh  bool   // draw everything again, and keep what is drawn
	drew   int
	served int
	record string // what drew, for people; read at the first store
	scrap  string // where a run with the cache off draws, for the run
}

// close lets go of what a run with the cache off drew.
func (held *shelf) close() {
	if held.scrap != "" {
		os.RemoveAll(held.scrap)
	}
}

// openShelf is the cache for this run. When smokejudge cannot hash its own
// binary the cache is off: every smoke then renders, and nothing is kept.
func openShelf(fresh bool) *shelf {
	held := &shelf{fresh: fresh}
	root, err := cache.Root(os.LookupEnv)
	if err != nil {
		held.off = err.Error()
		return held
	}
	here, err := cache.Here()
	if err != nil {
		held.off = err.Error()
		return held
	}
	held.root, held.here = root, here
	return held
}

// shelved is one smoke's verdict, and its entry, held open, so a copy is
// of the entry that was judged whatever happens to the cache after.
type shelved struct {
	report compare.Report
	entry  *cache.Entry
	drawn  bool
	unkept string // why a smoke drawn with the cache on was not kept
}

// get is a smoke at k: served from the cache when it holds it whole for
// this binary and what the smoke reads, and otherwise drawn from at, the
// scene the caller built, compared and kept. It is nil when the smoke
// cannot be drawn.
//
// The key is never the scene: it is what keyFor names. A smoke whose
// reads cannot be hashed is drawn and not kept. The caller closes the
// entry.
func (held *shelf) get(made smoke.Smoke, at scene, sz size, k int) (*shelved, error) {
	if held.off != "" {
		held.drew++
		return held.drawScrap(made, at, sz, k)
	}
	key, err := held.keyFor(made, sz, k)
	if err != nil {
		held.drew++
		got, derr := held.drawScrap(made, at, sz, k)
		if got != nil {
			got.unkept = err.Error()
		}
		return got, derr
	}
	if !held.fresh {
		if served := held.serve(*key); served != nil {
			return served, nil
		}
	}
	held.drew++
	return held.draw(*key, made, at, sz, k)
}

// serve is the verdict the cache holds for a key, whole, or nil when it
// holds none that can be read.
func (held *shelf) serve(key cache.Key) *shelved {
	entry, ok := cache.Open(held.root, key)
	if !ok {
		return nil
	}
	report, err := readVerdict(entry)
	if err != nil {
		entry.Close()
		return nil
	}
	held.served++
	return &shelved{report: *report, entry: entry}
}

// drawer is what is needed to draw a smoke into a folder: the smoke, its
// scene, the size and k, and, once drawn, the verdict.
type drawer struct {
	made  smoke.Smoke
	at    scene
	sz    size
	k     int
	drawn *compare.Report
}

// into draws the smoke into dir, for the cache to keep.
func (draw *drawer) into(dir string) error {
	var err error
	draw.drawn, err = drawInto(dir, draw.made, draw.at, draw.sz, draw.k)
	return err
}

// draw draws a smoke from its scene and has the cache keep it under key.
func (held *shelf) draw(key cache.Key, made smoke.Smoke, at scene, sz size,
	k int) (*shelved, error) {
	if held.record == "" {
		held.record = cache.RecordText()
	}
	drawing := &drawer{made: made, at: at, sz: sz, k: k}
	entry, err := cache.Store(held.root, key, held.record, held.fresh,
		drawing.into)
	if err != nil {
		return nil, err
	}
	return &shelved{report: *drawing.drawn, entry: entry, drawn: true}, nil
}

// readsOf is the files a smoke that reads only the one input reads.
func readsOf(input string) ([]string, error) {
	var reader smoke.Smoke
	reader.Inputs = []string{input}
	return reader.Reads()
}

// keyFor is the key for smoke made at k: this run's binary, what the smoke
// reads, its number, and the pictures as smokejudge draws them, or why what
// it reads cannot be hashed. It is nil when it cannot be hashed.
func (held *shelf) keyFor(made smoke.Smoke, sz size, k int) (*cache.Key, error) {
	key := held.here
	inputs, err := cache.Inputs(made.Inputs, readsOf)
	if err != nil {
		return nil, err
	}
	key.Inputs = inputs
	key.Smoke = made.Number
	key.Size = fmt.Sprintf("text %dx%d, real %dx%d, a text pixel %d by "+
		"%d real ones, k %d", sz.textW, sz.textH, sz.textW*across*k,
		sz.textH*down*k, across*k, down*k, k)
	key.Sampling = fmt.Sprintf("text %d rays a side a pixel, real one "+
		"ray a pixel", raysFor(made, sz))
	key.Sensor = fmt.Sprintf("pinhole %g wide, text pixel 0.75 as wide "+
		"as tall, real square, developed at exposure 1", made.Wide)
	return &key, nil
}

// drawInto renders a smoke both ways from its built scene, writes both
// pictures and the verdict into dir, and is the verdict.
func drawInto(dir string, made smoke.Smoke, at scene, sz size,
	k int) (*compare.Report, error) {
	terminalPicture, realPicture, err := both(made, at, sz, k)
	if err != nil {
		return nil, err
	}
	if err := file.PNG(filepath.Join(dir, cache.Text), terminalPicture,
		sextantWide); err != nil {
		return nil, err
	}
	if err := file.PNG(filepath.Join(dir, cache.Real), realPicture,
		exposure); err != nil {
		return nil, err
	}
	report := compare.Cells(realPicture, terminalPicture, across*k, down*k)
	return &report, writeVerdict(filepath.Join(dir, cache.Verdict), report)
}

// drawScrap draws a smoke for a run whose cache is off, into a folder that
// lasts as long as the run, so a copy can still be made from it.
func (held *shelf) drawScrap(made smoke.Smoke, at scene, sz size,
	k int) (*shelved, error) {
	if held.scrap == "" {
		dir, err := os.MkdirTemp("", "smokecheck-")
		if err != nil {
			return nil, err
		}
		held.scrap = dir
	}
	dir := filepath.Join(held.scrap,
		fmt.Sprintf("smoke-%d-%s-k%d", made.Number, sz.name, k))
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return nil, err
	}
	report, err := drawInto(dir, made, at, sz, k)
	if err != nil {
		return nil, err
	}
	entry, err := cache.Hold(dir, cache.Text, cache.Real, cache.Verdict)
	if err != nil {
		return nil, err
	}
	return &shelved{report: *report, entry: entry, drawn: true}, nil
}

// verdict is a comparison as it is kept: the numbers, and whether they
// make it honest.
type verdict struct {
	report compare.Report
}

// MarshalJSON writes the comparison's numbers and, beside them, whether
// they make it honest.
func (kept verdict) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		compare.Report
		Honest bool
	}{kept.report, kept.report.Honest()})
}

// writeVerdict keeps a comparison's numbers beside its pictures.
func writeVerdict(path string, report compare.Report) error {
	raw, err := json.MarshalIndent(verdict{report}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), fileMode)
}

// readVerdict is an entry's kept comparison. It is nil when it cannot be
// read.
func readVerdict(entry *cache.Entry) (*compare.Report, error) {
	raw, err := entry.Bytes(cache.Verdict)
	if err != nil {
		return nil, err
	}
	var kept compare.Report
	if err := json.Unmarshal(raw, &kept); err != nil {
		return nil, err
	}
	return &kept, nil
}
