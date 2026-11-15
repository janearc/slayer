package hillside

import (
	"context"
	"fmt"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/frame"
)

// CompositeSession exposes a set's linear surface plate and foreground
// transport. It stages through the ordinary session so geometry, lamps,
// weather, time and camera use the same recipe as the full renderer.
// It holds only one pass drawer at a time; changing passes releases it.
type CompositeSession struct {
	*Session
	drawer *gpu.Frames
	pass   gpu.CompositePass
}

// OpenComposite opens GPU set jobs with a pinhole and a closed shutter.
// Named scenes, lens rigs and film stocks need their own decomposition.
func OpenComposite(ctx context.Context, wanted job.Job, stamp Stamp) (*CompositeSession, error) {
	if wanted.Scene().Form() != "set" {
		return nil, fmt.Errorf("composite: a set job is required")
	}
	if _, yes := wanted.Shutter(); yes {
		return nil, fmt.Errorf("composite: shutter sampling is not supported")
	}
	if _, yes := wanted.Film().Aperture(); yes {
		return nil, fmt.Errorf("composite: an aperture is not supported")
	}
	if _, yes := wanted.Stock(); yes {
		return nil, fmt.Errorf("composite: a film stock is not supported")
	}
	session, err := Open(ctx, wanted, stamp)
	if err != nil {
		return nil, err
	}
	if !session.given.onGPU || session.given.glass != nil || session.given.frame != nil || session.given.overlay != nil {
		session.Close()
		return nil, fmt.Errorf("composite: use a GPU set with a pinhole, without a lens rig or HUD")
	}
	return &CompositeSession{Session: session}, nil
}

// Pass draws one primary-path contribution. A VolumeTransport pass returns
// scattered light as Drawn.Light and RGB transmission in the second frame.
// CompletePicture draws normally and returns no transmission. This does not
// decide when a plate is safe to reuse; that is the compositing plan's job.
func (session *CompositeSession) Pass(ctx context.Context, index int, ask Ask, pass gpu.CompositePass) (Drawn, *frame.Frame, error) {
	if session.closed {
		return Drawn{}, nil, errClosed
	}
	if pass == gpu.CompletePicture {
		session.releasePass()
		drawn, err := session.Session.Frame(ctx, index, ask)
		return drawn, nil, err
	}
	if session.drawer == nil || session.pass != pass {
		session.releasePass()
		drawer, err := gpu.NewCompositeFrames(framesMode(session.given), pass)
		if err != nil {
			return Drawn{}, nil, err
		}
		session.drawer, session.pass = drawer, pass
	}
	session.given.composite = session.drawer
	defer func() { session.given.composite = nil }()
	drawn, err := session.Session.Frame(ctx, index, ask)
	return drawn, session.drawer.Transmission(), err
}

// releasePass drops a pass's device allocations before another is opened.
func (session *CompositeSession) releasePass() {
	if session.drawer != nil {
		session.drawer.Close()
		session.drawer = nil
	}
}

// Close releases the compositor drawer and the underlying scene session.
func (session *CompositeSession) Close() error {
	session.releasePass()
	return session.Session.Close()
}
