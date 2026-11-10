package publication

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// preparedFixture owns generated boundaries for one actual test or trial.
type preparedFixture struct {
	controller *gomock.Controller
	filesystem *Mockfilesystem
	publisher  publisher
	request    preparedPublication
}

// preparedTest starts with no filesystem effects permitted.
func preparedTest(reporter gomock.TestReporter) preparedFixture {
	controller := gomock.NewController(reporter)
	filesystem := NewMockfilesystem(controller)
	return preparedFixture{controller: controller, filesystem: filesystem,
		publisher: publisher{filesystem: filesystem},
		request: preparedPublication{parent: "/queue",
			intent: []byte("prepared operation"), state: []byte("proposed state")}}
}

// sequence permits one complete snapshot, or one exact refusal and cleanup.
func (fixture preparedFixture) sequence(name string, body []byte,
	step string, cause, cleanup error,
) []any {
	file := NewMocktemporary(fixture.controller)
	temporary := filepath.Join(fixture.request.parent, "."+name+"-private")
	create := fixture.filesystem.EXPECT().CreateTemp(tempSpec{
		parent: fixture.request.parent, pattern: "." + name + "-*"})
	if step == "create" {
		return []any{create.Return(openedTemp{err: cause})}
	}
	calls := []any{create.Return(openedTemp{file: file, name: temporary})}
	written := writeResult{written: len(body)}
	if step == "write" {
		written.err = cause
	}
	if step == "short" {
		written.written = noWrittenBytes
	}
	calls = append(calls, file.EXPECT().Write(writeSpec{body: body}).Return(written))
	var closeCause error
	if step == "close" {
		closeCause = cause
	}
	calls = append(calls, file.EXPECT().Close().Return(closeCause))
	if step != "write" && step != "short" && step != "close" {
		var renameCause error
		if step == "rename" {
			renameCause = cause
		}
		calls = append(calls, fixture.filesystem.EXPECT().Rename(renameSpec{
			from: temporary, to: filepath.Join(fixture.request.parent, name),
		}).Return(renameCause))
	}
	if step != "" {
		calls = append(calls, fixture.filesystem.EXPECT().Remove(pathSpec{
			path: temporary}).Return(cleanup))
	}
	return calls
}

// success requires prepared intent, proposed state, then intent removal.
func (fixture preparedFixture) success(removal error) {
	calls := fixture.sequence("intent", fixture.request.intent, "", nil, nil)
	calls = append(calls, fixture.sequence("state", fixture.request.state,
		"", nil, nil)...)
	calls = append(calls, fixture.filesystem.EXPECT().Remove(pathSpec{
		path: filepath.Join(fixture.request.parent, "intent")}).Return(removal))
	gomock.InOrder(calls...)
}

// refusal allows no intent deletion or later publication after the failure.
func (fixture preparedFixture) refusal(intent bool, step string,
	cause, cleanup error,
) {
	name, body := "intent", fixture.request.intent
	var calls []any
	if !intent {
		calls = fixture.sequence(name, body, "", nil, nil)
		name, body = "state", fixture.request.state
	}
	calls = append(calls, fixture.sequence(name, body, step, cause, cleanup)...)
	gomock.InOrder(calls...)
}

// TestPreparedCompletesInOrder permits no rollback after success.
func TestPreparedCompletesInOrder(t *testing.T) {
	fixture := preparedTest(t)
	fixture.success(nil)
	got := fixture.publisher.complete(fixture.request)
	if got.stage != completionComplete || got.err != nil {
		t.Fatalf("ordered completion: %+v", got)
	}
}

// TestPreparedRefusesInvalidParents pairs ordered completion with no effects.
func TestPreparedRefusesInvalidParents(t *testing.T) {
	fixture := preparedTest(t)
	for _, parent := range []string{"", "relative", "/queue/..", "/queue/.",
		"/queue//nested", "/queue/", "//"} {
		request := fixture.request
		request.parent = parent
		got := fixture.publisher.complete(request)
		if got.stage != completionIntentFailed || got.err == nil {
			t.Fatalf("invalid parent %q: %+v", parent, got)
		}
	}
}

// TestPreparedAcceptsOpaquePayloads checks independent authored byte families.
func TestPreparedAcceptsOpaquePayloads(t *testing.T) {
	for _, intent := range payloadExamples(t).Replacement.bodies() {
		for _, state := range payloadExamples(t).Replacement.bodies() {
			fixture := preparedTest(t)
			fixture.request.intent, fixture.request.state = intent, state
			fixture.success(nil)
			got := fixture.publisher.complete(fixture.request)
			if got.stage != completionComplete || got.err != nil {
				t.Fatalf("opaque bytes: %+v", got)
			}
			fixture.controller.Finish()
		}
	}
}

// TestPreparedShortWritesCannotComplete preserves io.ErrShortWrite.
func TestPreparedShortWritesCannotComplete(t *testing.T) {
	for _, intent := range []bool{true, false} {
		fixture := preparedTest(t)
		fixture.refusal(intent, "short", nil, nil)
		got := fixture.publisher.complete(fixture.request)
		want := completionStateFailed
		if intent {
			want = completionIntentFailed
		}
		if got.stage != want || !errors.Is(got.err, io.ErrShortWrite) {
			t.Fatalf("short write, intent=%v: %+v", intent, got)
		}
		fixture.controller.Finish()
	}
}

// TestPreparedRetainsEveryIntentFailure forbids state publication and clearing.
func TestPreparedRetainsEveryIntentFailure(t *testing.T) {
	for _, step := range []string{"create", "write", "close", "rename"} {
		fixture := preparedTest(t)
		cause := errors.New(step)
		fixture.refusal(true, step, cause, nil)
		got := fixture.publisher.complete(fixture.request)
		if got.stage != completionIntentFailed || !errors.Is(got.err, cause) {
			t.Fatalf("intent %s failure: %+v", step, got)
		}
		fixture.controller.Finish()
	}
}

// TestPreparedRetainsIntentAfterEveryStateFailure forbids rollback and clearing.
func TestPreparedRetainsIntentAfterEveryStateFailure(t *testing.T) {
	for _, step := range []string{"create", "write", "close", "rename"} {
		fixture := preparedTest(t)
		cause := errors.New(step)
		fixture.refusal(false, step, cause, nil)
		got := fixture.publisher.complete(fixture.request)
		if got.stage != completionStateFailed || !errors.Is(got.err, cause) {
			t.Fatalf("state %s failure: %+v", step, got)
		}
		fixture.controller.Finish()
	}
}

// TestPreparedRetainsCleanupCauses checks the phase and each joined cause.
func TestPreparedRetainsCleanupCauses(t *testing.T) {
	fixture := preparedTest(t)
	cause, cleanup := errors.New("rename"), errors.New("temporary cleanup")
	fixture.refusal(false, "rename", cause, cleanup)
	got := fixture.publisher.complete(fixture.request)
	if got.stage != completionStateFailed || !errors.Is(got.err, cause) ||
		!errors.Is(got.err, cleanup) {
		t.Fatalf("joined causes: %+v", got)
	}
}

// TestPreparedMissingFilesystemCannotAcknowledge permits no state transition.
func TestPreparedMissingFilesystemCannotAcknowledge(t *testing.T) {
	fixture := preparedTest(t)
	got := (publisher{}).complete(fixture.request)
	if got.stage != completionIntentFailed || got.err == nil {
		t.Fatalf("missing filesystem: %+v", got)
	}
}

// TestPreparedRemovalFailureCannotAcknowledge includes a possibly absent entry.
func TestPreparedRemovalFailureCannotAcknowledge(t *testing.T) {
	for _, cause := range []error{errors.New("remove uncertain"),
		errors.Join(errors.New("entry absent"), os.ErrNotExist)} {
		fixture := preparedTest(t)
		fixture.success(cause)
		got := fixture.publisher.complete(fixture.request)
		if got.stage != completionRemovalUncertain || !errors.Is(got.err, cause) {
			t.Fatalf("intent removal: %+v", got)
		}
		fixture.controller.Finish()
	}
}

// TestPreparedZeroResultGrantsNothing checks the omitted-result boundary.
func TestPreparedZeroResultGrantsNothing(t *testing.T) {
	if (completionResult{}).stage == completionComplete {
		t.Fatal("zero result acknowledges completion")
	}
}

// preparedBytesProperty keeps the existing authored arbitrary-byte bounds.
type preparedBytesProperty struct{ minimum, maximum int }

// check compares both independently generated payloads with exact mock calls.
func (bounds preparedBytesProperty) check(trial *rapid.T) {
	fixture := preparedTest(trial)
	defer fixture.controller.Finish()
	bytes := rapid.SliceOfN(rapid.Byte(), bounds.minimum, bounds.maximum)
	fixture.request.intent = bytes.Draw(trial, "prepared intent")
	fixture.request.state = bytes.Draw(trial, "proposed state")
	fixture.success(nil)
	got := fixture.publisher.complete(fixture.request)
	if got.stage != completionComplete || got.err != nil {
		trial.Fatalf("generated publication: %+v", got)
	}
}

// TestPreparedGeneratedBytesStayDistinct checks exact independent payloads.
func TestPreparedGeneratedBytesStayDistinct(t *testing.T) {
	examples := payloadExamples(t)
	property := preparedBytesProperty{minimum: examples.MinimumGeneratedBytes,
		maximum: examples.MaximumGeneratedBytes}
	rapid.Check(t, property.check)
}

// preparedFailureProperty varies the phase and its failed filesystem boundary.
type preparedFailureProperty struct{}

// check retains the failed phase and original cause without clearing evidence.
func (preparedFailureProperty) check(trial *rapid.T) {
	fixture := preparedTest(trial)
	defer fixture.controller.Finish()
	intent := rapid.Bool().Draw(trial, "intent phase")
	step := rapid.SampledFrom([]string{"create", "write", "close", "rename"}).
		Draw(trial, "failed boundary")
	cause := errors.New("injected boundary failure")
	fixture.refusal(intent, step, cause, nil)
	got := fixture.publisher.complete(fixture.request)
	want := completionStateFailed
	if intent {
		want = completionIntentFailed
	}
	if got.stage != want || !errors.Is(got.err, cause) {
		trial.Fatalf("generated refusal: %+v", got)
	}
}

// TestPreparedGeneratedFailuresRetainEvidence is the paired anti-property.
func TestPreparedGeneratedFailuresRetainEvidence(t *testing.T) {
	rapid.Check(t, preparedFailureProperty{}.check)
}
