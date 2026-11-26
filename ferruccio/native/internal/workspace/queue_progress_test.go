package workspace

import (
	"context"
	"errors"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/native/internal/picking"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/ferruccio/internal/preview"
)

func TestQueuedPreviewWithStaleWorkerShowsWhatToCheck(t *testing.T) {
	work := playWorkspace(t)
	_, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	work.progress = make(chan renderProgress, 1)
	work.progress <- renderProgress{generation: work.generation, update: preview.Update{
		Sequence: 100, State: "queued", Held: "No recent queue worker activity · check the queue worker",
	}}
	work.Tick()
	readout := work.playReadout()
	if !strings.Contains(readout, "Queued") || !strings.Contains(readout, "check the queue worker") || !strings.Contains(readout, "waiting for completed pixels") {
		t.Fatalf("queued preview did not explain the missing worker: %q", readout)
	}
	if work.cancel == nil || work.output != "" {
		t.Fatal("stale worker evidence cancelled the request or accepted an image")
	}
}

// Queue acknowledgement and running metadata never stand in for new pixels.
func TestQueueProgressTickRetainsTheDisplayedWorld(t *testing.T) {
	work := playWorkspace(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	work.output = "retained-preview.png"
	before, pixels, selected := work.state.document.Text(), work.viewer.source, work.play.selected
	work.progress = make(chan renderProgress, 4)
	for _, update := range []preview.Update{
		{State: "queued", Held: "GPU occupied", Sequence: 1},
		{State: "running", Sequence: 1},
		{State: "succeeded", Sequence: 1},
	} {
		work.progress <- renderProgress{update: update, generation: work.generation}
		work.Tick()
		if work.state.document.Text() != before || work.viewer.source != pixels || work.play.selected != selected || work.output != "retained-preview.png" {
			t.Fatal("queue metadata changed the source, selection or displayed preview")
		}
		if output, _ := work.ProofOutput(); output != "" {
			t.Fatal("queue metadata exposed previous pixels as a completed new request")
		}
		if work.cancel == nil || ctx.Err() != nil {
			t.Fatal("queue acknowledgement finished the preview worker")
		}
		if strings.Contains(work.status, "Preview ready") {
			t.Fatal("queue state was presented as accepted pixels")
		}
		if update.State == "queued" && (!strings.Contains(work.playReadout(), "Queued") || !strings.Contains(work.playReadout(), "GPU occupied")) {
			t.Fatal("queued acknowledgement and held reason are absent from the footer")
		}
		if update.State == "running" && !strings.Contains(work.playReadout(), "Running") {
			t.Fatal("running state is absent from the footer")
		}
	}
}

// A durable job sequence binds identity; observations are not event counters.
func TestQueueProgressRejectsSupersededGenerationAndSequence(t *testing.T) {
	work := playWorkspace(t)
	_, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	work.progress = make(chan renderProgress, 4)
	work.progress <- renderProgress{update: preview.Update{State: "queued", Held: "current hold", Sequence: 8}, generation: work.generation}
	work.Tick()
	currentStatus, currentCaption := work.status, work.caption
	for _, stale := range []renderProgress{
		{update: preview.Update{State: "running", Sequence: 9}, generation: work.generation - 1},
		{update: preview.Update{State: "running", Sequence: 7}, generation: work.generation},
		{update: preview.Update{State: "running", Sequence: 9}, generation: work.generation},
		{update: preview.Update{State: "running", Sequence: 0}, generation: work.generation},
	} {
		work.progress <- stale
		work.Tick()
		if work.status != currentStatus || work.caption != currentCaption {
			t.Fatal("obsolete status snapshot replaced current queue feedback")
		}
	}
	work.progress <- renderProgress{update: preview.Update{State: "running", Sequence: 8}, generation: work.generation}
	work.Tick()
	if !strings.Contains(work.status, "Running") {
		t.Fatal("same durable job sequence did not update current feedback")
	}
}

// Stop cancels only this workspace's context and rejects late own-job metadata.
func TestQueueStopCancelsOwnRequestWithoutAcceptingLateProgress(t *testing.T) {
	work := playWorkspace(t)
	own, cancel := context.WithCancel(context.Background())
	other, otherCancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(otherCancel)
	work.cancel = cancel
	work.progress = make(chan renderProgress, 1)
	generation := work.generation
	work.StopPreview()
	status := work.status
	work.progress <- renderProgress{update: preview.Update{State: "running", Sequence: 1}, generation: generation}
	work.Tick()
	if own.Err() != context.Canceled || other.Err() != nil {
		t.Fatal("Stop did not remain scoped to its own request")
	}
	if work.status != status {
		t.Fatal("cancelled request's late progress restored running feedback")
	}
	if !strings.Contains(work.playReadout(), "cancelling") {
		t.Fatal("cancellation request was presented as submission or execution")
	}
	work.finishPreview(renderResult{generation: generation, observation: preview.Update{State: "cancelled", Sequence: 1, Terminal: true}, err: context.Canceled})
	if work.Running() || work.status != "Preview cancellation confirmed" {
		t.Fatal("completed cancelled worker remained outstanding")
	}
}

// Queue configuration is mandatory; old Game-only configuration cannot render.
func TestQueueRendererConfigurationHasNoDirectGameFallback(t *testing.T) {
	work := playWorkspace(t)
	work.options.Evidence = t.TempDir()
	for _, sample := range []struct {
		tools preview.Tools
		want  bool
	}{
		{preview.Tools{Game: "game5", Slayer: "slayer"}, false},
		{preview.Tools{Azazel: "azazel", Slayer: "slayer"}, false},
		{preview.Tools{Queue: "/queue", Slayer: "slayer"}, false},
		{preview.Tools{Azazel: "azazel", Queue: "/queue"}, false},
		{preview.Tools{Azazel: "azazel", Queue: "/queue", Slayer: "slayer"}, true},
	} {
		work.options.Tools = sample.tools
		if got := work.rendererConfigured(); got != sample.want {
			t.Fatalf("rendererConfigured=%t want%t for%+v", got, sample.want, sample.tools)
		}
	}
}

// Missing queue setup leaves the actual editable window calm on its first tick.
func TestQueueMissingConfigurationKeepsInitialDraftEditable(t *testing.T) {
	work := playWorkspace(t)
	before := work.state.document.Text()
	work.autoPreview = true
	work.Tick()
	if work.cancel != nil || work.diagnostic != "" || work.editor.Text() != before ||
		!strings.Contains(strings.ToLower(work.status), "queue") || work.autoPreview {
		t.Fatalf("missing queue disturbed editable opening: status=%q diagnostic=%q", work.status, work.diagnostic)
	}
	if work.editor.Text() != work.state.document.Text() {
		t.Fatal("missing queue changed source authority")
	}
}

// Proof receipts need a current accepted source, not retained earlier UI pixels.
func TestQueueProofOutputRequiresCurrentAcceptedSourceAndGeneration(t *testing.T) {
	work := playWorkspace(t)
	work.output = "retained-preview.png"
	work.acceptedGeneration = work.generation
	work.acceptedPath = work.state.path
	work.acceptedText = work.state.document.Text()
	if path, diagnostic := work.ProofOutput(); path != work.output || diagnostic != "" {
		t.Fatal("matching accepted view was refused")
	}
	work.generation++
	if path, _ := work.ProofOutput(); path != "" {
		t.Fatal("obsolete accepted generation was reported as current")
	}
	work.acceptedGeneration = work.generation
	work.state.document.Replace(work.state.document.Text() + "# current edit\n")
	if path, _ := work.ProofOutput(); path != "" {
		t.Fatal("older accepted source was reported as current")
	}
	if path, _ := work.Output(); path != work.output {
		t.Fatal("normal UI lost retained previous output")
	}
}

// A first queued preview stays visibly absent rather than becoming an image.
func TestQueueFirstPreviewRemainsBlankUntilCompletedPixels(t *testing.T) {
	work := playWorkspace(t)
	work.viewer.set(nil)
	_, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	work.progress = make(chan renderProgress, 1)
	work.progress <- renderProgress{generation: work.generation, update: preview.Update{State: "queued", Held: "operator pause", Sequence: 1}}
	work.Tick()
	if work.viewer.source != nil || work.output != "" || !strings.Contains(work.caption, "No preview yet") || !strings.Contains(work.status, "Queued") {
		t.Fatal("queued first request was presented as a rendered image")
	}
}

// A claimed job waiting on admission must not be reported as executing.
func TestQueueWaitingForAdmissionIsNotRunning(t *testing.T) {
	work := playWorkspace(t)
	_, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	work.progress = make(chan renderProgress, 1)
	work.progress <- renderProgress{generation: work.generation, update: preview.Update{State: "waiting", Held: "Game admission", Sequence: 1}}
	work.Tick()
	if !strings.Contains(work.status, "admission") || strings.Contains(work.playReadout(), "Running") || strings.Contains(work.playReadout(), "Drawing") {
		t.Fatalf("waiting job was presented as executing: %q / %q", work.status, work.playReadout())
	}
}

// Stale Move or Turn pixels cannot silently become a camera gesture.
func TestQueueStalePlacementPressDoesNotOrbitOrEdit(t *testing.T) {
	for _, mode := range []string{"move", "turn"} {
		t.Run(mode, func(t *testing.T) {
			work := playWorkspace(t)
			rendered := work.state.document.Text()
			work.play.renderedText = rendered
			current := rendered + "# current edit awaiting preview\n"
			work.state.document.Replace(current)
			work.editor.SetText(current)
			work.setPlayMode(mode)
			drawWholePlayRoot(t, work)
			bounds := work.viewer.ScreenBounds()
			start := geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2)
			end := geometry.Pt(start.X+90, start.Y+35)
			work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, start, start, event.ModNone))
			work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, event.ButtonStateLeft, end, end, event.ModNone))
			work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, event.ModNone))
			if work.viewer.dragging || work.state.document.Text() != current || work.editor.Text() != current || work.pending.document != nil {
				t.Fatal("stale placement drag changed source or fell back to camera orbit")
			}
			if !strings.Contains(work.caption, "Look") {
				t.Fatal("stale placement did not explain the explicit Look alternative")
			}
			work.setPlayMode("look")
			drawWholePlayRoot(t, work)
			bounds = work.viewer.ScreenBounds()
			start = geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2)
			end = geometry.Pt(start.X+90, start.Y+35)
			work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, start, start, event.ModNone))
			work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, event.ButtonStateLeft, end, end, event.ModNone))
			work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, event.ModNone))
			if work.state.document.Text() == current {
				t.Fatal("explicit Look was blocked with stale placement pixels")
			}
		})
	}
}

// Owner state and held reason may change repeatedly without changing job identity.
func TestQueueSameJobSequenceFollowsOwnerStateAndHeldChanges(t *testing.T) {
	work := playWorkspace(t)
	_, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	work.progress = make(chan renderProgress, 1)
	for _, sample := range []struct{ state, held, want string }{
		{"queued", "first hold", "first hold"},
		{"queued", "new held reason", "new held reason"},
		{"waiting", "Game admission", "admission"},
		{"running", "", "Running"},
		{"queued", "operator requeued", "operator requeued"},
		{"succeeded", "", "collecting"},
	} {
		work.progress <- renderProgress{generation: work.generation, update: preview.Update{State: sample.state, Held: sample.held, Sequence: 41}}
		work.Tick()
		if !strings.Contains(work.playReadout(), sample.want) {
			t.Fatalf("same-job owner observation %s/%s ignored: %s", sample.state, sample.held, work.playReadout())
		}
	}
}

// Current background drags retain the existing camera-look fallback.
func TestQueueCurrentBackgroundMissStillLooksAround(t *testing.T) {
	work := playWorkspace(t)
	scene, err := picking.New(work.state.document.Text(), 1.6)
	if err != nil {
		t.Fatal(err)
	}
	work.play.geometry = scene
	work.play.renderedText = work.state.document.Text()
	if scene.Pick(.01, .01) != "" {
		t.Fatal("test background point unexpectedly hits a native object")
	}
	drawWholePlayRoot(t, work)
	imageBounds := work.viewer.imageBounds()
	screen, local := work.viewer.ScreenBounds(), work.viewer.Bounds()
	start := geometry.Pt(imageBounds.Min.X+imageBounds.Width()*.01+screen.Min.X-local.Min.X, imageBounds.Min.Y+imageBounds.Height()*.01+screen.Min.Y-local.Min.Y)
	end := geometry.Pt(start.X+30, start.Y+15)
	before := work.state.document.Text()
	object, _ := work.selectedObject()
	position := object.Position()
	work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, start, start, event.ModNone))
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, event.ButtonStateLeft, end, end, event.ModNone))
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, end, end, event.ModNone))
	object, _ = work.selectedObject()
	if work.state.document.Text() == before || object.Position() != position || work.play.mode != "move" {
		t.Fatal("current background look changed placement or lost camera behavior")
	}
}

func TestQueueCatalogueRequiresItsAppropriateBackend(t *testing.T) {
	work := cameraWorkspace(t)
	work.options.Evidence = t.TempDir()
	work.options.Tools = preview.Tools{Azazel: "azazel", Queue: "/queue", Slayer: "slayer"}
	if work.rendererConfigured() {
		t.Fatal("catalogue accepted the unrelated scene backend")
	}
	work.options.Tools.Catalogue = "catalogue"
	if !work.rendererConfigured() {
		t.Fatal("configured queue catalogue backend was refused")
	}
}

// Ending observation is distinct from a confirmed own-process outcome.
func TestQueueStopCompletionRequiresOwnerTerminalConfirmation(t *testing.T) {
	for _, sample := range []struct {
		name      string
		update    preview.Update
		failure   error
		want      string
		uncertain bool
	}{
		{"confirmed cancellation", preview.Update{State: "cancelled", Sequence: 41, Terminal: true}, context.Canceled, "Preview cancellation confirmed", false},
		{"completed before stop", preview.Update{State: "succeeded", Sequence: 41, Terminal: true}, nil, "Preview completed before cancellation", false},
		{"confirmed failure", preview.Update{State: "failed", Sequence: 41, Terminal: true}, errors.New("own renderer failed"), "Preview job failed", false},
		{"unconfirmed wait", preview.Update{State: "waiting", Sequence: 41}, errors.New("cancellation unconfirmed; frozen input retained: retained.yaml"), "render outcome unconfirmed", true},
		{"interrupted uncertain", preview.Update{State: "interrupted", Sequence: 41}, errors.New("interrupted; frozen input retained: retained.yaml"), "render outcome unconfirmed", true},
	} {
		t.Run(sample.name, func(t *testing.T) {
			work := playWorkspace(t)
			before, pixels, selected := work.state.document.Text(), work.viewer.source, work.play.selected
			work.output = "previous-preview.png"
			_, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			work.cancel = cancel
			old := work.generation
			work.log.Write([]byte("own cancellation audit\n"))
			work.StopPreview()
			work.finishPreview(renderResult{generation: old, observation: sample.update, err: sample.failure})
			if !strings.Contains(work.status, sample.want) {
				t.Fatalf("cancel outcome=%q want%q", work.status, sample.want)
			}
			if work.Running() || work.viewer.source != pixels || work.state.document.Text() != before || work.play.selected != selected || work.output != "previous-preview.png" {
				t.Fatal("cancel outcome changed retained source, image or selection")
			}
			if sample.uncertain && (!strings.Contains(work.diagnostic, "retained.yaml") || !strings.Contains(work.diagnostic, "own cancellation audit")) {
				t.Fatal("unconfirmed cancellation lost its own retained-input/error audit")
			}
		})
	}
}

// An obsolete cancellation diagnostic must not replace feedback for a newer edit.
func TestQueueOldUnconfirmedCancellationPreservesNewPendingFeedback(t *testing.T) {
	work := playWorkspace(t)
	_, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	old := work.generation
	work.StopPreview()
	work.pending.remember(work.state, work.specimen, time.Now())
	work.status, work.caption = "New source pending", "New view pending"
	work.finishPreview(renderResult{generation: old, observation: preview.Update{State: "interrupted", Sequence: 41}, err: errors.New("old cancellation unconfirmed; input retained")})
	if work.status != "New source pending" || work.caption != "New view pending" || work.pending.document == nil || work.diagnostic != "" {
		t.Fatal("old cancellation report replaced the new pending generation")
	}
}

// The actual Stop button must not turn a cancellation request into a stopped claim.
func TestQueueStopButtonReleaseWaitsForTerminalConfirmation(t *testing.T) {
	work := playWorkspace(t)
	own, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	work.cancel = cancel
	work.output = "previous-preview.png"
	before, pixels := work.state.document.Text(), work.viewer.source
	generation := work.generation
	work.rebuild()
	drawWholePlayRoot(t, work)
	var stop widget.Widget
	var visit func(widget.Widget)
	visit = func(current widget.Widget) {
		if identified, ok := current.(interface{ ID() string }); ok && identified.ID() == "Stop" {
			stop = current
		}
		for _, child := range current.Children() {
			visit(child)
		}
	}
	visit(work.root)
	if stop == nil {
		t.Fatal("actual play widget tree has no Stop control")
	}
	positioned, ok := stop.(interface{ ScreenBounds() geometry.Rect })
	if !ok {
		t.Fatal("Stop control has no framework screen bounds")
	}
	bounds := positioned.ScreenBounds()
	center := geometry.Pt((bounds.Min.X+bounds.Max.X)/2, (bounds.Min.Y+bounds.Max.Y)/2)
	work.ui.HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, center, center, event.ModNone))
	if own.Err() != nil {
		t.Fatal("Stop acted before the button release")
	}
	work.ui.HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, center, center, event.ModNone))
	if own.Err() != context.Canceled || !work.Running() {
		t.Fatal("Stop release did not request and await own cancellation")
	}
	if strings.Contains(strings.ToLower(work.caption), "stopped") || strings.Contains(strings.ToLower(work.status), "stopped") {
		t.Fatalf("Stop button claimed quiescence before confirmation: %q / %q", work.status, work.caption)
	}
	if !strings.Contains(strings.ToLower(work.caption), "cancell") {
		t.Fatal("Stop button did not show cancellation requested")
	}
	if work.state.document.Text() != before || work.viewer.source != pixels {
		t.Fatal("Stop button changed the retained world")
	}
	work.finishPreview(renderResult{generation: generation, observation: preview.Update{State: "cancelled", Sequence: 41, Terminal: true}, err: context.Canceled})
	if work.status != "Preview cancellation confirmed" || !strings.Contains(work.caption, "confirmed") {
		t.Fatal("confirmed owner outcome did not replace requested cancellation")
	}
}

// The native frontend never creates unlisted temporary preview destinations.
func TestQueueMissingEvidenceDestinationIsCalmEditOnly(t *testing.T) {
	work := playWorkspace(t)
	work.options.Tools = preview.Tools{Azazel: "azazel", Queue: "/queue", Slayer: "slayer", Agent: "studio"}
	if work.rendererConfigured() {
		t.Fatal("preview allowed without its shared evidence destination")
	}
	before := work.editor.Text()
	work.autoPreview = true
	work.Tick()
	if work.Running() || work.editor.Text() != before || work.diagnostic != "" || !strings.Contains(work.status, "destination") {
		t.Fatalf("missing destination disturbed editable opening: %s / %s", work.status, work.diagnostic)
	}
	work.options.Evidence = t.TempDir()
	if !work.rendererConfigured() {
		t.Fatal("explicit preview evidence destination was refused")
	}
}

// Every native request explicitly carries its configured shared root and safe prefix.
func TestQueuePreviewRequestCarriesSharedRootAndAgentPrefix(t *testing.T) {
	work := playWorkspace(t)
	work.options.Evidence = "/shared/renders/frames"
	work.options.Tools.Agent = "../Studio / Solis"
	now := time.Date(2026, 11, 16, 0, 20, 0, 0, time.UTC)
	request := work.previewRequest(now)
	if request.OutputRoot != work.options.Evidence || request.Path != work.state.path || request.Text != work.editor.Text() {
		t.Fatal("native request lost shared destination or source identity")
	}
	if request.DirectoryPrefix != "2026-11-16-studio---solis-ferruccio-preview-" {
		t.Fatalf("unsafe or incorrect agent prefix %q", request.DirectoryPrefix)
	}
	if request.Profile.Width != 640 || request.Profile.Height != 400 || !request.Profile.CPURenderer {
		t.Fatal("shared destination changed the native preview budget")
	}
	for _, agent := range []string{"", "../", strings.Repeat("a", 400), "Studio/../other"} {
		prefix := previewDirectoryPrefix(now, agent)
		if strings.ContainsAny(prefix, "/.\\") || len(prefix) > 100 || !strings.HasSuffix(prefix, "-ferruccio-preview-") {
			t.Fatalf("agent produced unsafe basename %q", prefix)
		}
	}
}
