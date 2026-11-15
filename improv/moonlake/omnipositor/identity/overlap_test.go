package identity

import (
	"bytes"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestOverlappingJobRanges(t *testing.T) {
	dir := t.TempDir()
	a, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	spec := fixture(t)
	data := tiny(t, 1)
	ranges := [][2]int64{{10, 15}, {13, 18}}
	stores := []*Store{a, b}
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	reuses := make(chan bool, 12)
	for job := range ranges {
		wg.Add(1)
		go func(job int) {
			defer wg.Done()
			for frame := ranges[job][0]; frame <= ranges[job][1]; frame++ {
				local := spec
				local.Frame = frame
				k, err := Build(local)
				if err != nil {
					errs <- err
					continue
				}
				reused, err := stores[job].Publish(k, data, false)
				errs <- err
				reuses <- reused
			}
		}(job)
	}
	wg.Wait()
	close(errs)
	close(reuses)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	for reused := range reuses {
		if reused {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("overlap reuse count=%d, want 3", count)
	}
	for frame := int64(10); frame <= 18; frame++ {
		local := spec
		local.Frame = frame
		k := build(t, local)
		if _, err := a.Verify(k); err != nil {
			t.Fatal(err)
		}
		readFlat(t, dir, k, data)
	}
}

func TestRedrawNeverMixesManifestAndPixels(t *testing.T) {
	dir := t.TempDir()
	writer, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	reader, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	spec := fixture(t)
	a := build(t, spec)
	spec.Backend = "other"
	b := build(t, spec)
	pixelsA := tiny(t, 1)
	pixelsB := tiny(t, 2)
	if _, err := writer.Publish(a, pixelsA, false); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 30; i++ {
			k := a
			data := pixelsA
			if i%2 == 0 {
				k = b
				data = pixelsB
			}
			if _, err := writer.Publish(k, data, true); err != nil {
				errs <- err
				return
			}
		}
	}()
	for i := 0; i < 30; i++ {
		flat, err := os.ReadFile(filepath.Join(dir, slot(a)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := png.Decode(bytes.NewReader(flat)); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(flat, pixelsA) && !bytes.Equal(flat, pixelsB) {
			t.Fatal("flat reader saw partial redraw")
		}
		for _, pair := range []struct {
			k    Key
			data []byte
		}{{a, pixelsA}, {b, pixelsB}} {
			got, err := reader.Verify(pair.k)
			if err != nil && !errors.Is(err, ErrConflict) {
				t.Fatal(err)
			}
			if err == nil && !bytes.Equal(got, pair.data) {
				t.Fatal("manifest and pixels mixed across redraw")
			}
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}
