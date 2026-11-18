package door

import "github.com/janearc/slayer/internal/observability"

// frameEvent is a best-effort observation after existing frame work succeeds.
func (run runner) frameEvent(index int, cost spent, kept, still, wrote, read, encoded bool) {
	if !observability.FrameEventsEnabled() {
		return
	}
	at, known := run.wanted.Clock().FrameTime(index)
	event := observability.FrameEvent{Frame: index, ObservedAt: run.now(), SceneTime: at, SceneTimeKnown: known, Kept: kept, Still: still, DrawKnown: !kept, WriteKnown: wrote, ReadKnown: read, EncodeKnown: encoded, Draw: cost.draw, Develop: cost.develop, Titles: cost.titles, Encode: cost.encode}
	if wrote {
		event.Write = cost.write
	}
	if read {
		event.Read = cost.write
	}
	observability.EmitFrame(event)
}
