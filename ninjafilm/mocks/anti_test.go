package mocks_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/mocks"
	"github.com/janearc/slayer/ninjafilm/record"
	"go.uber.org/mock/gomock"
)

// stopped is what a recording reporter panics with when a mock fails the
// test, so that the call does not go on.
const stopped = "the mock failed the test"

// recorder is a test reporter that remembers what a mock said instead of
// failing. Mocks that fail the test stop with a panic.
type recorder struct{ said []string }

// Errorf remembers what was said.
func (heard *recorder) Errorf(format string, args ...any) {
	heard.said = append(heard.said, fmt.Sprintf(format, args...))
}

// Fatalf remembers what was said and stops the call.
func (heard *recorder) Fatalf(format string, args ...any) {
	heard.Errorf(format, args...)
	panic(stopped)
}

// attempt makes a call on a mock that nobody told what to expect. It
// gives what the mock said, and whether it stopped the call.
func attempt(call func(*gomock.Controller)) (said []string, stoppedIt bool) {
	heard := listening()
	defer reached(heard, &said, &stoppedIt)
	call(gomock.NewController(heard))
	return heard.said, false
}

// listening is a recorder that has heard nothing.
func listening() *recorder { return new(recorder) }

// reached notes where a call that panicked had got to.
func reached(heard *recorder, said *[]string, stoppedIt *bool) {
	if recover() == nil {
		return
	}
	*said, *stoppedIt = heard.said, true
}

// layAcross is a call on a lay nobody told to expect it.
func layAcross(ctrl *gomock.Controller) { mocks.NewMockLay(ctrl).Across(once) }

// laySite is a call on a lay nobody told to expect it.
func laySite(ctrl *gomock.Controller) {
	mocks.NewMockLay(ctrl).Site([]record.Record{}, once, once, once, once)
}

// shelfFind is a call on a shelf nobody told to expect it.
func shelfFind(ctrl *gomock.Controller) { mocks.NewMockShelf(ctrl).Find("a") }

// anti-tests: a lay fails the test for a call nobody expected. It names
// the method that was called.
func TestAMockLayFailsTheTestForACallNobodyExpected(t *testing.T) {
	for name, call := range map[string]func(*gomock.Controller){
		"Across": layAcross, "Site": laySite} {
		said, stoppedIt := attempt(call)
		if !stoppedIt || !strings.Contains(strings.Join(said, ""), name) {
			t.Errorf("an unexpected %s said %v, stopped %v", name, said,
				stoppedIt)
		}
	}
}

// anti-tests: a shelf fails the test for a call nobody expected.
func TestAMockShelfFailsTheTestForACallNobodyExpected(t *testing.T) {
	said, stoppedIt := attempt(shelfFind)
	if !stoppedIt || !strings.Contains(strings.Join(said, ""), "Find") {
		t.Errorf("an unexpected Find said %v, stopped %v", said, stoppedIt)
	}
}

// projectionRay is a call on a projection nobody told to expect it.
func projectionRay(ctrl *gomock.Controller) {
	mocks.NewMockProjection(ctrl).Ray(once, once, once, once, maths.Band{})
}

// emitterLight is a call on an emitter nobody told to expect it.
func emitterLight(ctrl *gomock.Controller) {
	mocks.NewMockEmitter(ctrl).Light(maths.Vec{}, maths.Vec{})
}

// beamerBeams is a call on a beamer nobody told to expect it.
func beamerBeams(ctrl *gomock.Controller) {
	mocks.NewMockBeamer(ctrl).Beams(maths.Band{})
}

// anti-tests: a projection fails the test for a call nobody expected.
func TestAMockProjectionFailsTheTestForACallNobodyExpected(t *testing.T) {
	said, stoppedIt := attempt(projectionRay)
	if !stoppedIt || !strings.Contains(strings.Join(said, ""), "Ray") {
		t.Errorf("an unexpected Ray said %v, stopped %v", said, stoppedIt)
	}
}

// anti-tests: an emitter fails the test for a call nobody expected.
func TestAMockEmitterFailsTheTestForACallNobodyExpected(t *testing.T) {
	said, stoppedIt := attempt(emitterLight)
	if !stoppedIt || !strings.Contains(strings.Join(said, ""), "Light") {
		t.Errorf("an unexpected Light said %v, stopped %v", said, stoppedIt)
	}
}

// anti-tests: a beamer fails the test for a call nobody expected.
func TestAMockBeamerFailsTheTestForACallNobodyExpected(t *testing.T) {
	said, stoppedIt := attempt(beamerBeams)
	if !stoppedIt || !strings.Contains(strings.Join(said, ""), "Beams") {
		t.Errorf("an unexpected Beams said %v, stopped %v", said, stoppedIt)
	}
}
