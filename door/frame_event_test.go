//go:build darwin || linux

package door

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/observability"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/frame"
	"go.uber.org/mock/gomock"
	"golang.org/x/sys/unix"
)

func TestStillFrameEventsCoverDrawnAndCached(t *testing.T) {
	fd := frameEvents(t)
	directory := t.TempDir()
	file := written(t, directory, "still.yaml", stillJob(t, ""))
	wanted, err := job.Read(file)
	if err != nil {
		t.Fatal(err)
	}
	control := gomock.NewController(t)
	source := NewMocksource(control)
	developer := NewMockdeveloper(control)
	drawn := NewMockdrawn(control)
	light := &frame.Frame{}
	picture := image.NewRGBA(image.Rect(0, 0, 1, 1))
	source.EXPECT().Frame(gomock.Any(), 0).Return(drawn, nil)
	drawn.EXPECT().Light().Return(light).AnyTimes()
	drawn.EXPECT().Overlay(gomock.Any()).Return(nil)
	developer.EXPECT().Develop(0, []*frame.Frame{light}).Return(picture, nil)
	run := runner{ctx: context.Background(), out: io.Discard, now: time.Now, wanted: wanted, at: places{picture: filepath.Join(directory, "drawn.png")}}
	if err = run.still(kit{source: source, developer: developer}); err != nil {
		t.Fatal(err)
	}
	event := nextFrameEvent(t, fd)
	if !event.Still || event.Kept || !event.DrawKnown || event.Frame != 0 {
		t.Fatal(event)
	}
	home := t.TempDir()
	env, err := NewEnv(Given{Context: context.Background(), Directory: directory, Now: time.Now, Environment: func(name string) (string, bool) {
		switch name {
		case "SLAYER_CACHE":
			return home, true
		case keptStillsVariable:
			return keptStillsOn, true
		}
		return "", false
	}})
	if err != nil {
		t.Fatal(err)
	}
	store, err := pictureStore(env)
	if err != nil {
		t.Fatal(err)
	}
	var payload bytes.Buffer
	if err = png.Encode(&payload, picture); err != nil {
		t.Fatal(err)
	}
	if err = store.Put(pictureKey(wanted), payload.Bytes()); err != nil {
		t.Fatal(err)
	}
	run = runner{ctx: context.Background(), out: io.Discard, now: time.Now, wanted: wanted, at: places{picture: filepath.Join(directory, "kept.png")}, env: env, keeps: true}
	if err = run.drawStill(); err != nil {
		t.Fatal(err)
	}
	event = nextFrameEvent(t, fd)
	if !event.Still || !event.Kept || event.DrawKnown || event.Frame != 0 {
		t.Fatal(event)
	}
}

func frameEvents(t *testing.T) int {
	t.Helper()
	pair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = unix.SetNonblock(pair[0], true); err != nil {
		t.Fatal(err)
	}
	if err = unix.SetNonblock(pair[1], true); err != nil {
		t.Fatal(err)
	}
	sender, err := observability.NewFrameSender(pair[0], "0123456789abcdef0123456789abcdef", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	stop, err := observability.InstallFrameSender(sender)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stop(); unix.Close(pair[1]) })
	return pair[1]
}

func nextFrameEvent(t *testing.T, fd int) observability.FramePacket {
	t.Helper()
	var buffer [observability.MaxFramePacket]byte
	n, _, err := unix.Recvfrom(fd, buffer[:], unix.MSG_DONTWAIT)
	if err != nil {
		t.Fatal(err)
	}
	var event observability.FramePacket
	if err = json.Unmarshal(buffer[:n], &event); err != nil {
		t.Fatal(err)
	}
	return event
}

func TestFrameEventsDistinguishDrawnAndKeptFrames(t *testing.T) {
	fd := frameEvents(t)
	folder := filepath.Join(t.TempDir(), "work")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := benchedIn(t, ctx, folder)
	first.shop.EXPECT().Reel(ctx, gomock.Any()).Return(first.reel, nil)
	picture := first.frameAt(t, 0, cancel)
	first.reel.EXPECT().Frame(0, picture).Return(nil)
	first.reel.EXPECT().Abort().Return(nil)
	if err := first.run.draw(); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	event := nextFrameEvent(t, fd)
	if event.Kind != "frame" || event.Frame != 0 || event.Kept || !event.DrawKnown || event.Draw.Seconds() != drawSeconds {
		t.Fatal(event)
	}
	second := benchedIn(t, context.Background(), folder)
	second.shop.EXPECT().Reel(second.run.ctx, gomock.Any()).Return(second.reel, nil)
	second.reel.EXPECT().Frame(0, gomock.Any()).Return(nil)
	for index := 1; index < filmFrames(t); index++ {
		drawn := second.frameAt(t, index, nil)
		second.reel.EXPECT().Frame(index, drawn).Return(nil)
	}
	second.reel.EXPECT().Close().Return(nil)
	if err := second.run.draw(); err != nil {
		t.Fatal(err)
	}
	event = nextFrameEvent(t, fd)
	if event.Frame != 0 || !event.Kept || event.DrawKnown || !event.ReadKnown {
		t.Fatal(event)
	}
}

func TestFailedFrameDoesNotProduceSuccessObservation(t *testing.T) {
	fd := frameEvents(t)
	made := benched(t, context.Background(), "")
	made.shop.EXPECT().Titler(made.run.wanted).Return(made.titles, true, nil)
	made.shop.EXPECT().Reel(made.run.ctx, gomock.Any()).Return(made.reel, nil)
	picture := made.frameAt(t, 0, nil)
	made.reel.EXPECT().Frame(0, picture).Return(nil)
	broken := errors.New("frame failed")
	made.source.EXPECT().Frame(made.run.ctx, 1).Return(nil, broken)
	made.reel.EXPECT().Abort().Return(nil)
	if err := made.run.draw(); !errors.Is(err, broken) {
		t.Fatal(err)
	}
	if event := nextFrameEvent(t, fd); event.Frame != 0 {
		t.Fatal(event)
	}
	var buffer [observability.MaxFramePacket]byte
	if _, _, err := unix.Recvfrom(fd, buffer[:], unix.MSG_DONTWAIT); err != unix.EAGAIN && err != unix.EWOULDBLOCK {
		t.Fatal("unexpected event", err)
	}
}
