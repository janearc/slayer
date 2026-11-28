package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestOpenSetUsesStagingValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scene.yaml")
	if err := os.WriteFile(path, []byte(addRecipeFixture), 0600); err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	defer work.closeStaged()
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		return nil, nil
	}
	work.needsCheck = true
	work.openPath(path)
	if work.openCheck == nil || work.status != "Checking YAML…" || work.needsCheck {
		t.Fatal("opening checked the set on the window thread")
	}
	work.updateWorking()
	work.working.stager.workers.Wait()
	work.drainStagedAuthoring()
	if work.openCheck != nil || work.status != "YAML checked · set" {
		t.Fatalf("staging did not report validation: %s", work.status)
	}
}

// A completed stage must not discard its still-buffered validation result.
func TestCompletedStageConsumesOpenCheck(t *testing.T) {
	work := playWorkspace(t)
	defer work.closeStaged()
	source := work.state.document.Text()
	checked := make(chan error, 1)
	checked <- nil
	ready := make(chan *set.Set, 1)
	work.working.source, work.working.path = source, work.state.path
	work.openCheck, work.status, work.renderBlocked = work.state.document, "Checking YAML…", true
	work.working.stageAuthoring = &stageAuthoring{work.state.document, ready, checked}
	work.working.stager = newStageLoader(nil)
	work.working.stager.ready = &stageResult{stageRequest: stageRequest{path: work.state.path, source: source}}
	work.drainStaged()
	if work.openCheck != nil || work.status == "Checking YAML…" {
		t.Fatal("completed stage discarded the pending validation")
	}
}

func TestOpenSetReportsInvalidSchemaFromStaging(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scene.yaml")
	source := strings.Replace(addRecipeFixture, "seed: 19", "seed: incorrect", 1)
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	defer work.closeStaged()
	called := make(chan struct{}, 1)
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		called <- struct{}{}
		return nil, nil
	}
	work.openPath(path)
	work.updateWorking()
	work.working.stager.workers.Wait()
	work.drainStaged()
	select {
	case <-called:
		t.Fatal("invalid input reached the staging helper")
	default:
	}
	if work.status != "Draft needs attention" || work.diagnostic == "" || !work.renderBlocked {
		t.Fatalf("invalid draft was not reported: %s / %s", work.status, work.diagnostic)
	}
}

func TestOpenSetCheckPreservesANewerWarning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scene.yaml")
	if err := os.WriteFile(path, []byte(addRecipeFixture), 0600); err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	defer work.closeStaged()
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		return nil, nil
	}
	work.openPath(path)
	work.updateWorking()
	work.working.stager.workers.Wait()
	work.status, work.diagnostic = "Draft saved with unchecked dependencies", "Dependency warning"
	work.drainStagedAuthoring()
	if work.status != "Draft saved with unchecked dependencies" || work.diagnostic != "Dependency warning" {
		t.Fatal("the earlier open check erased a newer warning")
	}
}

func TestOpenSetFinishesCheckIfHelperDisappears(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scene.yaml")
	if err := os.WriteFile(path, []byte(addRecipeFixture), 0600); err != nil {
		t.Fatal(err)
	}
	work := playWorkspace(t)
	defer work.closeStaged()
	work.working.stageOverride = func(context.Context, string, string, string) (*stagedView, error) {
		return nil, nil
	}
	work.openPath(path)
	work.working.stageOverride = nil
	work.updateWorking()
	if work.openCheck != nil || work.status == "Checking YAML…" {
		t.Fatal("missing helper left the check pending")
	}
}
