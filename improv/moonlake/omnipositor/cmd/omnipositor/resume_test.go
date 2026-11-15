package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	compositor "github.com/janearc/slayer/improv/moonlake/omnipositor"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/encoder"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

func resumeKey(t *testing.T, frame int64, recipe string) identity.Key {
	t.Helper()
	return mustResumeKey(t, 2, 2, recipe, frame)
}

func mustResumeKey(t *testing.T, width, height int, recipe string, frame ...int64) identity.Key {
	t.Helper()
	f := int64(10)
	if len(frame) > 0 {
		f = frame[0]
	}
	k, err := identity.BuildPrehashed(identity.PrehashedSpec{BinarySHA256: strings.Repeat("a", 64), Settings: []byte(recipe), FPS: 24, Frame: f, Backend: "cpu", Device: "test", Width: width, Height: height})
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func resumeImage() *image.RGBA { return resumeImageWith(color.RGBA{R: 90, A: 255}) }

func resumeImageWith(c color.RGBA) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, 2, 2))
	im.SetRGBA(0, 0, c)
	return im
}

func resumeBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, resumeImage()); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func pngBytes(t *testing.T, im *image.RGBA) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, im); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestVerifiedPNGRefusesReusedPublishRace(t *testing.T) {
	dir := t.TempDir()
	writer, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	competitor, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer competitor.Close()
	key := resumeKey(t, 10, "race")
	want := resumeImageWith(color.RGBA{R: 12, A: 255})
	start := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := verifiedPNGWithHook(context.Background(), writer, key, false, func() error {
			close(start)
			<-release
			return nil
		}, func() (*image.RGBA, error) { return resumeImage(), nil })
		done <- err
	}()
	select {
	case <-start:
	case <-time.After(time.Second):
		t.Fatal("did not reach publish window")
	}
	if _, err := competitor.Publish(key, pngBytes(t, want), false); err != nil {
		t.Fatal(err)
	}
	close(release)
	select {
	case err := <-done:
		if !errors.Is(err, errVerifiedPNGReused) {
			t.Fatalf("reuse race: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reused publish")
	}
}

func TestVerifiedPNGRejectsNilRenderResult(t *testing.T) {
	dir := t.TempDir()
	store, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := resumeKey(t, 10, "nil-render")
	_, err = verifiedPNG(context.Background(), store, key, false, func() (*image.RGBA, error) { return nil, nil })
	if !errors.Is(err, errInvalidRenderedFrame) {
		t.Fatalf("nil render: %v", err)
	}
}

func TestVerifiedPNGRejectsMalformedRenderResult(t *testing.T) {
	dir := t.TempDir()
	store, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := resumeKey(t, 10, "bad-render")
	bad := &image.RGBA{Pix: make([]byte, 1), Stride: 1, Rect: image.Rect(0, 0, 2, 2)}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("malformed render panicked: %v", r)
		}
	}()
	_, err = verifiedPNG(context.Background(), store, key, false, func() (*image.RGBA, error) { return bad, nil })
	if !errors.Is(err, errInvalidRenderedFrame) {
		t.Fatalf("malformed render: %v", err)
	}
}

func TestValidateRenderedRGBARejectsGeometryAndOverflow(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	tooWide := &image.RGBA{Pix: make([]byte, 4), Stride: 4, Rect: image.Rect(minInt, 0, maxInt, 1)}
	if err := validateRenderedRGBA(tooWide); !errors.Is(err, errInvalidRenderedFrame) {
		t.Fatalf("huge width: %v", err)
	}
	tooTall := &image.RGBA{Pix: make([]byte, 4), Stride: 4, Rect: image.Rect(0, minInt, 1, maxInt)}
	if err := validateRenderedRGBA(tooTall); !errors.Is(err, errInvalidRenderedFrame) {
		t.Fatalf("huge height: %v", err)
	}
	tooNarrowStride := &image.RGBA{Pix: make([]byte, 7), Stride: 7, Rect: image.Rect(0, 0, 2, 1)}
	if err := validateRenderedRGBA(tooNarrowStride); !errors.Is(err, errInvalidRenderedFrame) {
		t.Fatalf("short stride: %v", err)
	}
	empty := &image.RGBA{Pix: nil, Stride: 4, Rect: image.Rect(0, 0, 0, 1)}
	if err := validateRenderedRGBA(empty); !errors.Is(err, errInvalidRenderedFrame) {
		t.Fatalf("empty geometry: %v", err)
	}
}

func TestValidateRenderedRGBAAcceptsValidSubimage(t *testing.T) {
	base := image.NewRGBA(image.Rect(-4, -4, 4, 4))
	sub := base.SubImage(image.Rect(-2, -1, 1, 2)).(*image.RGBA)
	if err := validateRenderedRGBA(sub); err != nil {
		t.Fatalf("valid negative subimage: %v", err)
	}
}

func TestValidateRenderedRGBAAcceptsExtremaOrigin(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	im := &image.RGBA{Pix: make([]byte, 16), Stride: 8, Rect: image.Rect(minInt+1, minInt+2, minInt+3, minInt+4)}
	if err := validateRenderedRGBA(im); err != nil {
		t.Fatalf("extrema origin: %v", err)
	}
}

func TestVerifiedPNGReturnsCancellationAfterDecode(t *testing.T) {
	dir := t.TempDir()
	store, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	im := image.NewRGBA(image.Rect(0, 0, 2048, 2048))
	for y := 0; y < 2048; y++ {
		for x := 0; x < 2048; x++ {
			im.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: uint8(x ^ y), A: 255})
		}
	}
	key := mustResumeKey(t, 2048, 2048, "cancel-after-decode")
	if _, err := store.Publish(key, pngBytes(t, im), false); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	im, err = verifiedPNG(ctx, store, key, false, func() (*image.RGBA, error) { t.Fatal("rendered reused frame"); return nil, nil })
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("decode cancellation: %v", err)
	}
	if im != nil {
		t.Fatal("returned image after cancellation")
	}
}

func TestResumeVerificationContentionCancellation(t *testing.T) {
	dir := t.TempDir()
	store, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	lock, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	rendered := make(chan struct{}, 1)
	key := resumeKey(t, 10, "contended")
	go func() {
		_, err := verifiedPNG(ctx, store, key, false, func() (*image.RGBA, error) { rendered <- struct{}{}; return resumeImage(), nil })
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("contended resume: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("resume verification ignored cancellation")
	}
	select {
	case <-rendered:
		t.Fatal("rendered canceled resume")
	default:
	}
}

func TestVerifiedPNGStorage(t *testing.T) {
	dir := t.TempDir()
	store, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := resumeKey(t, 10, "first")
	calls := 0
	render := func() (*image.RGBA, error) { calls++; return resumeImage(), nil }
	for i := 0; i < 2; i++ {
		im, err := verifiedPNG(context.Background(), store, key, false, render)
		if err != nil || im.RGBAAt(0, 0).R != 90 {
			t.Fatalf("reuse %v %v", im, err)
		}
	}
	if calls != 1 {
		t.Fatalf("rendered verified frame %d times", calls)
	}
	changed := resumeKey(t, 10, "changed")
	if _, err := verifiedPNG(context.Background(), store, changed, false, render); !errors.Is(err, identity.ErrConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if calls != 1 {
		t.Fatal("rendered conflicting frame")
	}
	if _, err := verifiedPNG(context.Background(), store, changed, true, render); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal("redraw skipped render")
	}
	if _, err := store.Verify(changed); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(dir, "00010.png"))
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("flat store link: %v %v", info, err)
	}
}

func TestVerifiedPNGCorruptionAndUnownedRedraw(t *testing.T) {
	for _, owned := range []bool{false, true} {
		dir := t.TempDir()
		store, err := identity.Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		key := resumeKey(t, 10, "first")
		path := filepath.Join(dir, "00010.png")
		if owned {
			if _, err := store.Publish(key, resumeBytes(t), false); err != nil {
				t.Fatal(err)
			}
			target, err := os.Readlink(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, target), []byte("broken"), 0600); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(path, []byte("arbitrary input"), 0600); err != nil {
			t.Fatal(err)
		}
		render := func() (*image.RGBA, error) { return resumeImage(), nil }
		if _, err := verifiedPNG(context.Background(), store, key, false, render); !errors.Is(err, identity.ErrCorrupt) {
			t.Fatalf("corruption: %v", err)
		}
		_, err = verifiedPNG(context.Background(), store, key, true, render)
		if owned {
			if err != nil {
				t.Fatal(err)
			}
		} else {
			if !errors.Is(err, identity.ErrCorrupt) {
				t.Fatalf("unowned redraw: %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "arbitrary input" {
				t.Fatalf("clobbered arbitrary file: %q %v", data, err)
			}
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVerifiedPNGCancelAndNoClobber(t *testing.T) {
	dir := t.TempDir()
	store, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	key := resumeKey(t, 10, "first")
	ctx, cancel := context.WithCancel(context.Background())
	_, err = verifiedPNG(ctx, store, key, false, func() (*image.RGBA, error) { cancel(); return resumeImage(), nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := store.Verify(key); !errors.Is(err, identity.ErrMissing) {
		t.Fatalf("published canceled frame: %v", err)
	}
	path := filepath.Join(dir, "00010.png")
	_, err = verifiedPNG(context.Background(), store, key, false, func() (*image.RGBA, error) {
		if err := os.WriteFile(path, []byte("racing file"), 0600); err != nil {
			t.Fatal(err)
		}
		return resumeImage(), nil
	})
	if !errors.Is(err, identity.ErrCorrupt) {
		t.Fatalf("racing file: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "racing file" {
		t.Fatalf("clobbered racing file: %q %v", data, err)
	}
}

func TestMixedVerifiedAndRenderedMovie(t *testing.T) {
	dir := t.TempDir()
	store, err := identity.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(resumeKey(t, 10, "mixed"), resumeBytes(t), false); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	var rendered []int
	e := &mockEncoding{}
	deps := stretchDeps{prepare: func(ctx context.Context, index int) (preparedFrame, error) {
		return mockPreparedFrame{key: resumeKey(t, int64(index), "mixed"), render: func(ctx context.Context, consume func(compositor.Result) error) error {
			rendered = append(rendered, index)
			return consume(compositor.Result{Index: index, Image: resumeImage()})
		}}, nil
	}, encode: func(context.Context, encoder.Config) (frameEncoder, error) { return e, nil }}
	o := stretchOptions{from: 10, to: 13, out: dir, png: true, encode: filepath.Join(dir, "clip.mp4")}
	c := compositor.Config{From: 10, To: 13, Width: 2, Height: 2, FPS: 24}
	if err := renderStretch(context.Background(), o, c, deps); err != nil {
		t.Fatal(err)
	}
	if len(rendered) != 2 || rendered[0] != 11 || rendered[1] != 12 || len(e.indices) != 3 || e.indices[0] != 0 || e.indices[1] != 1 || e.indices[2] != 2 || !e.closed || e.aborted {
		t.Fatalf("mixed movie: %v %+v", rendered, e)
	}
}

func TestMovieCannotReplaceStoreGeneration(t *testing.T) {
	dir := t.TempDir()
	o := stretchOptions{from: 10, to: 11, out: dir, png: true, redraw: true, encode: filepath.Join(dir, ".generation-example", "frame.png")}
	if err := preflightOutputs(o); err == nil {
		t.Fatal("accepted movie inside store generation")
	}
}

func TestOutputDirectoryProtectsInputs(t *testing.T) {
	dir := t.TempDir()
	s := settings.Settings{Score: filepath.Join(dir, "score.yaml")}
	o := stretchOptions{from: 10, to: 11, out: dir, png: true}
	if err := protectInputs(o, s); err == nil {
		t.Fatal("accepted input inside store directory")
	}
}
