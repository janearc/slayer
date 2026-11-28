package workspace

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

func BenchmarkWorkingSetSource(b *testing.B) {
	path := os.Getenv("FERRUCCIO_WORKING_SET_BENCH")
	if path == "" {
		b.Skip("set FERRUCCIO_WORKING_SET_BENCH to the local scene to measure")
	}
	document, absolute, err := ReadDocument(path)
	if err != nil {
		b.Fatal(err)
	}
	if err := document.Check(); err != nil {
		b.Fatal(err)
	}
	source := document.Text()
	b.Run("draft validation", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if err := document.Check(); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("typed set", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := set.Read(absolute, []byte(source)); err != nil {
				b.Fatal(err)
			}
		}
	})
	for _, scenario := range []struct {
		name        string
		provisional bool
	}{
		{name: "initial kickoff"},
		{name: "restage kickoff", provisional: true},
	} {
		b.Run(scenario.name, func(b *testing.B) {
			work := &Workspace{composition: "play", state: documentState{document, absolute}, host: gogpu.NewApp(gogpu.DefaultConfig()), ui: app.New(), viewer: newImageView()}
			startedHelper := make(chan struct{}, 1)
			work.working.stageOverride = func(ctx context.Context, _, _, _ string) (*stagedView, error) {
				startedHelper <- struct{}{}
				<-ctx.Done()
				return nil, ctx.Err()
			}
			view := &placement.DisplayScene{Width: 640, Height: 400}
			b.ReportAllocs()
			var kickoff, helper time.Duration
			for b.Loop() {
				work.working.source = ""
				if scenario.provisional {
					work.working.preview = &workingPreview{document: document, scene: view, source: source, path: absolute}
				}
				started := time.Now()
				work.updateWorking()
				kickoff += time.Since(started)
				if !work.working.staging {
					b.Fatal("benchmark did not take the staging path")
				}
				select {
				case <-startedHelper:
					helper += time.Since(started)
				case <-time.After(60 * time.Second):
					work.closeStaged()
					b.Fatal("benchmark staging did not reach the helper")
				}
				work.closeStaged()
			}
			b.ReportMetric(float64(kickoff.Nanoseconds())/float64(b.N), "kickoff-ns/op")
			b.ReportMetric(float64(helper.Nanoseconds())/float64(b.N), "helper-launch-ns/op")
		})
	}
}
