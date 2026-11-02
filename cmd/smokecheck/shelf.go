package main

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
func (sh *shelf) close() {
	if sh.scrap != "" {
		os.RemoveAll(sh.scrap)
	}
}

// openShelf is the cache for this run. When smokecheck cannot hash its own
// binary the cache is off: every smoke then renders, and nothing is kept.
func openShelf(fresh bool) *shelf {
	sh := &shelf{fresh: fresh}
	root, err := cache.Root()
	if err != nil {
		sh.off = err.Error()
		return sh
	}
	here, err := cache.Here()
	if err != nil {
		sh.off = err.Error()
		return sh
	}
	sh.root, sh.here = root, here
	return sh
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
// this binary and what the smoke reads, and otherwise drawn from sc, the
// scene the caller built, compared and kept.
//
// The key is never the scene: it is what keyFor names. A smoke whose
// reads cannot be hashed is drawn and not kept. The caller closes the
// entry.
func (sh *shelf) get(s smoke.Smoke, sc scene, k int) (shelved, error) {
	if sh.off != "" {
		sh.drew++
		return sh.drawScrap(s, sc, k)
	}
	key, err := sh.keyFor(s, k)
	if err != nil {
		sh.drew++
		got, derr := sh.drawScrap(s, sc, k)
		got.unkept = err.Error()
		return got, derr
	}
	if !sh.fresh {
		if e, ok := cache.Open(sh.root, key); ok {
			r, err := readVerdict(e)
			if err == nil {
				sh.served++
				return shelved{report: r, entry: e}, nil
			}
			e.Close()
		}
	}
	sh.drew++
	if sh.record == "" {
		sh.record = cache.RecordText()
	}
	var r compare.Report
	e, err := cache.Store(sh.root, key, sh.record, sh.fresh,
		func(dir string) error {
			var err error
			r, err = drawInto(dir, s, sc, k)
			return err
		})
	if err != nil {
		return shelved{}, err
	}
	return shelved{report: r, entry: e, drawn: true}, nil
}

// keyFor is the key for smoke s at k: this run's binary, what the smoke
// reads, its number, and the pictures as smokecheck draws them, or why what it
// reads cannot be hashed.
func (sh *shelf) keyFor(s smoke.Smoke, k int) (cache.Key, error) {
	key := sh.here
	in, err := cache.Inputs(s.Inputs, func(in string) ([]string, error) {
		return smoke.Smoke{Inputs: []string{in}}.Reads()
	})
	if err != nil {
		return cache.Key{}, err
	}
	key.Inputs = in
	key.Smoke = s.Number
	key.Size = fmt.Sprintf("text %dx%d, real %dx%d, a text pixel %d by "+
		"%d real ones, k %d", textW, textH, textW*across*k,
		textH*down*k, across*k, down*k, k)
	key.Sampling = fmt.Sprintf("text %d rays a side a pixel, real one "+
		"ray a pixel", s.Samples)
	key.Sensor = fmt.Sprintf("pinhole %g wide, text pixel 0.75 as wide "+
		"as tall, real square, developed at exposure 1", s.Wide)
	return key, nil
}

// drawInto renders a smoke both ways from its built scene, writes both
// pictures and the verdict into dir, and is the verdict.
func drawInto(dir string, s smoke.Smoke, sc scene,
	k int) (compare.Report, error) {
	text, real := Both(s, sc, k)
	if err := file.PNG(filepath.Join(dir, cache.Text), text,
		0.75); err != nil {
		return compare.Report{}, err
	}
	if err := file.PNG(filepath.Join(dir, cache.Real), real,
		1); err != nil {
		return compare.Report{}, err
	}
	r := compare.Cells(real, text, across*k, down*k)
	return r, writeVerdict(filepath.Join(dir, cache.Verdict), r)
}

// drawScrap draws a smoke for a run whose cache is off, into a folder that
// lasts as long as the run, so a copy can still be made from it.
func (sh *shelf) drawScrap(s smoke.Smoke, sc scene, k int) (shelved, error) {
	if sh.scrap == "" {
		dir, err := os.MkdirTemp("", "smokecheck-")
		if err != nil {
			return shelved{}, err
		}
		sh.scrap = dir
	}
	dir := filepath.Join(sh.scrap,
		fmt.Sprintf("smoke-%d-k%d", s.Number, k))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return shelved{}, err
	}
	r, err := drawInto(dir, s, sc, k)
	if err != nil {
		return shelved{}, err
	}
	e, err := cache.Hold(dir, cache.Text, cache.Real, cache.Verdict)
	if err != nil {
		return shelved{}, err
	}
	return shelved{report: r, entry: e, drawn: true}, nil
}

// verdict is a comparison as it is kept: the numbers, and whether they
// make it honest.
type verdict struct {
	compare.Report
	Honest bool
}

// writeVerdict keeps a comparison's numbers beside its pictures.
func writeVerdict(path string, r compare.Report) error {
	b, err := json.MarshalIndent(verdict{r, r.Honest()}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// readVerdict is an entry's kept comparison.
func readVerdict(e *cache.Entry) (compare.Report, error) {
	b, err := e.Bytes(cache.Verdict)
	if err != nil {
		return compare.Report{}, err
	}
	var v verdict
	if err := json.Unmarshal(b, &v); err != nil {
		return compare.Report{}, err
	}
	return v.Report, nil
}
