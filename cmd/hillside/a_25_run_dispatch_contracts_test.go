package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// Stop at the actual build report, before the default large render. This
// writer observes the real command route without replacing its parser,
// constructors or renderer, and bounds a wrongly routed sequence call too.
type dispatchProgressStop struct {
	data bytes.Buffer
	stop error
}

func (writer *dispatchProgressStop) Write(data []byte) (int, error) {
	writer.data.Write(data)
	panic(writer.stop)
}

func (writer *dispatchProgressStop) String() string { return writer.data.String() }
func (writer *dispatchProgressStop) Len() int       { return writer.data.Len() }

func TestNoArgumentsReachTheNormalBuildReportWithoutIndexingAnAbsentVerb(t *testing.T) {
	stop := errors.New("stop after actual command build report")
	said := &dispatchProgressStop{stop: stop}
	var wrong bytes.Buffer
	var stopped interface{}
	func() {
		defer func() { stopped = recover() }()
		run(nil, said, &wrong)
	}()
	if stopped != stop {
		t.Fatalf("empty command did not reach normal build progress: panic %v, diagnostics %s", stopped, wrong.String())
	}
	if wrong.Len() != 0 || !strings.Contains(said.String(), "things of") || !strings.Contains(said.String(), "faces") {
		t.Fatalf("default route did not report a real built hillside: %q; %q", said.String(), wrong.String())
	}
}

func TestBareSequenceVerbReachesItsOwnPlanRefusal(t *testing.T) {
	stop := errors.New("sequence incorrectly fell into single-frame construction")
	said := &dispatchProgressStop{stop: stop}
	var wrong bytes.Buffer
	var stopped interface{}
	code := -1
	func() {
		defer func() { stopped = recover() }()
		code = run([]string{"sequence"}, said, &wrong)
	}()
	if stopped != nil || code != 1 || !strings.Contains(wrong.String(), "sequence wants --plan") {
		t.Fatalf("bare sequence lost its actual command refusal: code%d, panic %v, diagnostics %q", code, stopped, wrong.String())
	}
	if said.Len() != 0 {
		t.Fatal("sequence started single-frame construction before checking its own plan")
	}
}

func TestRunReportsAnActualSceneConstructionRefusalBeforeRendering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-scene.yaml")
	stop := errors.New("construction error incorrectly reached drawing")
	said := &dispatchProgressStop{stop: stop}
	var wrong bytes.Buffer
	var stopped interface{}
	code := -1
	func() {
		defer func() { stopped = recover() }()
		code = run([]string{"-set", path}, said, &wrong)
	}()
	if stopped != nil || code != 1 || said.Len() != 0 ||
		!strings.Contains(wrong.String(), "hillside:") || !strings.Contains(wrong.String(), path) {
		t.Fatalf("lost the real scene-open refusal: code%d, panic %v, progress %q, diagnostics %q",
			code, stopped, said.String(), wrong.String())
	}
}
