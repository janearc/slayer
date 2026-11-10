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

// immutableFixture owns generated boundaries with no calls allowed initially.
type immutableFixture struct {
	controller *gomock.Controller
	filesystem *MockimmutableFilesystem
	file       *Mocktemporary
	publisher  immutablePublisher
	request    installationSpec
	temporary  string
}

// immutableTest binds generated mocks to the actual test or Rapid trial.
func immutableTest(reporter gomock.TestReporter) immutableFixture {
	controller := gomock.NewController(reporter)
	filesystem := NewMockimmutableFilesystem(controller)
	return immutableFixture{
		controller: controller, filesystem: filesystem,
		file:      NewMocktemporary(controller),
		publisher: immutablePublisher{filesystem: filesystem},
		request: installationSpec{parent: "/queue", name: "result",
			body: []byte("verified bytes")},
		temporary: "/queue/.result-private",
	}
}

// creation pins the exact parent, pattern, transferred handle and private name.
func (fixture immutableFixture) creation() *gomock.Call {
	return fixture.filesystem.EXPECT().CreateTemp(tempSpec{
		parent: fixture.request.parent, pattern: "." + fixture.request.name + "-*",
	}).Return(openedTemp{file: fixture.file, name: fixture.temporary})
}

// installCall cannot authorize ordinary Rename: that method is absent here.
func (fixture immutableFixture) installCall() *gomock.Call {
	return fixture.filesystem.EXPECT().Install(installSpec{
		from: fixture.temporary,
		to:   filepath.Join(fixture.request.parent, fixture.request.name),
	})
}

// success allows full writing, one close, then exclusive installation.
func (fixture immutableFixture) success() {
	written := writeResult{written: len(fixture.request.body)}
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(writeSpec{body: fixture.request.body}).
			Return(written),
		fixture.file.EXPECT().Close().Return(nil),
		fixture.installCall().Return(nil))
}

// writeFailure permits cleanup of only the successfully transferred temporary.
func (fixture immutableFixture) writeFailure(written writeResult) {
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(writeSpec{body: fixture.request.body}).
			Return(written),
		fixture.file.EXPECT().Close().Return(nil),
		fixture.filesystem.EXPECT().Remove(pathSpec{path: fixture.temporary}).
			Return(nil))
}

// installFailure retains primary and private-name cleanup errors independently.
func (fixture immutableFixture) installFailure(cause, cleanup error) {
	written := writeResult{written: len(fixture.request.body)}
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(writeSpec{body: fixture.request.body}).
			Return(written),
		fixture.file.EXPECT().Close().Return(nil),
		fixture.installCall().Return(cause),
		fixture.filesystem.EXPECT().Remove(pathSpec{path: fixture.temporary}).
			Return(cleanup))
}

// TestImmutablePublishesCompletePayloads includes nil, empty and binary bodies.
func TestImmutablePublishesCompletePayloads(t *testing.T) {
	for _, body := range payloadExamples(t).Replacement.bodies() {
		fixture := immutableTest(t)
		fixture.request.body = body
		fixture.success()
		got := fixture.publisher.publish(fixture.request)
		if got.stage != publicationVisible || got.err != nil {
			t.Fatalf("immutable publication: %+v", got)
		}
		fixture.controller.Finish()
	}
}

// TestImmutableRefusesInvalidRequests is paired with complete-byte publication.
func TestImmutableRefusesInvalidRequests(t *testing.T) {
	fixture := immutableTest(t)
	requests := []installationSpec{
		{}, {parent: "relative", name: "result"},
		{parent: "/queue/../other", name: "result"},
		{parent: "/queue/.", name: "result"},
		{parent: "/queue//nested", name: "result"},
		{parent: "/queue/", name: "result"},
		{parent: "/queue"}, {parent: "/queue", name: "."},
		{parent: "/queue", name: ".."},
		{parent: "/queue", name: "part/result"},
		{parent: "/queue", name: "bad\x00name"},
	}
	for _, request := range requests {
		got := fixture.publisher.publish(request)
		if got.stage != publicationRefused || got.err == nil {
			t.Fatalf("invalid request accepted: %+v: %+v", request, got)
		}
	}
	got := (immutablePublisher{}).publish(fixture.request)
	if got.stage != publicationRefused || got.err == nil {
		t.Fatalf("missing filesystem accepted: %+v", got)
	}
}

// TestImmutableCreatesBesideItsDestination checks the selected nested parent.
func TestImmutableCreatesBesideItsDestination(t *testing.T) {
	fixture := immutableTest(t)
	fixture.request.parent = "/queue/nested"
	fixture.request.name = "result.v2"
	fixture.temporary = "/queue/nested/.result.v2-private"
	fixture.success()
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationVisible || got.err != nil {
		t.Fatalf("selected parent: %+v", got)
	}
}

// TestImmutableFailedCreateTransfersNothing permits no write, close or removal.
func TestImmutableFailedCreateTransfersNothing(t *testing.T) {
	fixture := immutableTest(t)
	cause := errors.New("create refused")
	fixture.filesystem.EXPECT().CreateTemp(tempSpec{
		parent: fixture.request.parent, pattern: ".result-*",
	}).Return(openedTemp{err: cause})
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationCreateFailed || !errors.Is(got.err, cause) {
		t.Fatalf("create refusal: %+v", got)
	}
}

// TestImmutableClosesBeforeInstalling forbids cleanup after success.
func TestImmutableClosesBeforeInstalling(t *testing.T) {
	fixture := immutableTest(t)
	fixture.success()
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationVisible || got.err != nil {
		t.Fatalf("ordered publication: %+v", got)
	}
}

// TestImmutableFailedCloseNeverInstalls permits neither retry nor publication.
func TestImmutableFailedCloseNeverInstalls(t *testing.T) {
	fixture := immutableTest(t)
	cause := errors.New("close refused")
	written := writeResult{written: len(fixture.request.body)}
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(writeSpec{body: fixture.request.body}).
			Return(written),
		fixture.file.EXPECT().Close().Return(cause),
		fixture.filesystem.EXPECT().Remove(pathSpec{path: fixture.temporary}).
			Return(nil))
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationCloseFailed || !errors.Is(got.err, cause) {
		t.Fatalf("close refusal: %+v", got)
	}
}

// TestImmutableWriteFailureCleansItsOwnedTemporary keeps the write cause.
func TestImmutableWriteFailureCleansItsOwnedTemporary(t *testing.T) {
	fixture := immutableTest(t)
	examples := payloadExamples(t)
	examples.requirePartial(t, fixture.request.body)
	cause := errors.New("partial write refused")
	fixture.writeFailure(writeResult{
		written: examples.FailedWriteCount, err: cause,
	})
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationWriteFailed || !errors.Is(got.err, cause) {
		t.Fatalf("write refusal: %+v", got)
	}
}

// TestImmutableShortWriteCannotInstall requires every authored byte.
func TestImmutableShortWriteCannotInstall(t *testing.T) {
	fixture := immutableTest(t)
	examples := payloadExamples(t)
	examples.requirePartial(t, fixture.request.body)
	fixture.writeFailure(writeResult{
		written: len(fixture.request.body) - examples.ShortWriteGap,
	})
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationWriteFailed ||
		!errors.Is(got.err, io.ErrShortWrite) {
		t.Fatalf("short-write refusal: %+v", got)
	}
}

// TestImmutableKeepsEveryCleanupCause tests independent error membership.
func TestImmutableKeepsEveryCleanupCause(t *testing.T) {
	fixture := immutableTest(t)
	writeCause, closeCause := errors.New("write"), errors.New("close")
	removeCause := errors.New("remove")
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(writeSpec{body: fixture.request.body}).
			Return(writeResult{err: writeCause}),
		fixture.file.EXPECT().Close().Return(closeCause),
		fixture.filesystem.EXPECT().Remove(pathSpec{path: fixture.temporary}).
			Return(removeCause))
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationWriteFailed {
		t.Fatalf("cleanup changed primary stage: %+v", got)
	}
	for _, cause := range []error{writeCause, closeCause, removeCause} {
		if !errors.Is(got.err, cause) {
			t.Fatalf("lost cause %v: %+v", cause, got)
		}
	}
}

// TestImmutableUncertainInstallNeverDeletesTheDestination is the anti-test.
func TestImmutableUncertainInstallNeverDeletesTheDestination(t *testing.T) {
	fixture := immutableTest(t)
	cause, cleanup := errors.New("install uncertain"), errors.New("cleanup")
	fixture.installFailure(cause, cleanup)
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationInstallUncertain || !errors.Is(got.err, cause) ||
		!errors.Is(got.err, cleanup) {
		t.Fatalf("uncertain install: %+v", got)
	}
}

// TestImmutableIncumbentIsNotThisAttemptsSuccess needs a refusal stage.
func TestImmutableIncumbentIsNotThisAttemptsSuccess(t *testing.T) {
	fixture := immutableTest(t)
	cause := errors.Join(errors.New("destination exists"), os.ErrExist)
	fixture.installFailure(cause, nil)
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationExists || !errors.Is(got.err, cause) ||
		!errors.Is(got.err, os.ErrExist) {
		t.Fatalf("incumbent refusal: %+v", got)
	}
}

// TestImmutableZeroResultNeverClaimsVisibility is the zero-value anti-test.
func TestImmutableZeroResultNeverClaimsVisibility(t *testing.T) {
	if (publicationResult{}).stage == publicationVisible ||
		(publicationResult{}).stage == publicationExists {
		t.Fatal("zero result claims a known namespace outcome")
	}
}

// TestImmutableAcceptsTheCleanRootParent uses mocks, never the actual root.
func TestImmutableAcceptsTheCleanRootParent(t *testing.T) {
	fixture := immutableTest(t)
	fixture.request.parent = "/"
	fixture.temporary = "/.result-private"
	fixture.success()
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationVisible || got.err != nil {
		t.Fatalf("clean root refused: %+v", got)
	}
}

// TestImmutableRefusesUncleanRootSpellings permits no filesystem calls.
func TestImmutableRefusesUncleanRootSpellings(t *testing.T) {
	fixture := immutableTest(t)
	for _, parent := range []string{"/.", "//", "/../"} {
		request := fixture.request
		request.parent = parent
		got := fixture.publisher.publish(request)
		if got.stage != publicationRefused || got.err == nil {
			t.Fatalf("unclean root accepted: %q: %+v", parent, got)
		}
	}
}

// immutableBytesProperty names the existing authored arbitrary-byte bounds.
type immutableBytesProperty struct {
	minimum, maximum int
}

// check owns fresh generated boundaries for every actual Rapid trial.
func (bounds immutableBytesProperty) check(property *rapid.T) {
	fixture := immutableTest(property)
	defer fixture.controller.Finish()
	fixture.request.body = rapid.SliceOfN(rapid.Byte(), bounds.minimum,
		bounds.maximum).Draw(property, "payload")
	fixture.success()
	got := fixture.publisher.publish(fixture.request)
	if got.stage != publicationVisible || got.err != nil {
		property.Fatalf("generated bytes: %+v", got)
	}
}

// TestImmutableGeneratedBytesArriveWhole checks exact arbitrary payload calls.
func TestImmutableGeneratedBytesArriveWhole(t *testing.T) {
	examples := payloadExamples(t)
	property := immutableBytesProperty{minimum: examples.MinimumGeneratedBytes,
		maximum: examples.MaximumGeneratedBytes}
	rapid.Check(t, property.check)
}

// immutableRefusalProperty draws primary and cleanup failures independently.
type immutableRefusalProperty struct{}

// check rejects success, changed stages, missing causes and invented causes.
func (immutableRefusalProperty) check(property *rapid.T) {
	fixture := immutableTest(property)
	defer fixture.controller.Finish()
	exists := rapid.Bool().Draw(property, "incumbent exists")
	cleanupFails := rapid.Bool().Draw(property, "cleanup fails")
	cause, cleanupCause := errors.New("install"), errors.New("cleanup")
	var cleanup error
	stage := publicationInstallUncertain
	if exists {
		cause = errors.Join(cause, os.ErrExist)
		stage = publicationExists
	}
	if cleanupFails {
		cleanup = cleanupCause
	}
	fixture.installFailure(cause, cleanup)
	got := fixture.publisher.publish(fixture.request)
	if got.stage != stage || !errors.Is(got.err, cause) ||
		errors.Is(got.err, os.ErrExist) != exists ||
		errors.Is(got.err, cleanupCause) != cleanupFails {
		property.Fatalf("generated refusal: %+v", got)
	}
}

// TestImmutableGeneratedRefusalsNeverClaimSuccess is the paired anti-property.
func TestImmutableGeneratedRefusalsNeverClaimSuccess(t *testing.T) {
	rapid.Check(t, immutableRefusalProperty{}.check)
}
