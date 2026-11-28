package workspace

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/set"
)

func TestSceneDetailRestoresThePreviousViewOnFailure(t *testing.T) {
	work := travelWorkspace(t)
	path := work.state.path
	work.play.selected = "scene.set.things/stone"
	if work.canShowSceneDetail() {
		t.Fatal("detail was available without a terrain set")
	}
	document, err := draft.Open(path, []byte(groundEditFixture))
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, []byte(document.Text()))
	if err != nil {
		t.Fatal(err)
	}
	work.state.document = document
	work.editor.SetText(document.Text())
	work.working.parsed, work.working.source = &read, document.Text()
	previous := work.working.scene
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		return nil, errors.New("test staging refusal")
	}
	defer work.closeStaged()
	work.showSceneDetail()
	if !strings.Contains(work.caption, "staging detail for stone") || !strings.Contains(work.diagnostic, "without rendering") {
		t.Fatalf("detail request did not identify its scope: %s / %s", work.caption, work.diagnostic)
	}
	waitStaged(t, work)
	if work.working.scene != previous || work.working.detail != "" ||
		work.working.failure != "" || !strings.Contains(work.diagnostic, "test staging refusal") {
		t.Fatal("failed detail lost the previous working view or its diagnostic")
	}
}
