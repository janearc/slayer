package publication

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// cleanupFixture owns the existing generated boundaries and cleanup subject.
type cleanupFixture struct {
	publication publicationFixture
	owner       temporaryCleanup
}

// cleanupTest transfers a trusted opened temporary without creating a file.
func cleanupTest(reporter gomock.TestReporter) cleanupFixture {
	fixture := publicationTest(reporter)
	return cleanupFixture{
		publication: fixture,
		owner: temporaryCleanup{remover: fixture.filesystem,
			opened: openedTemp{file: fixture.file, name: fixture.temporary}},
	}
}

// closeBeforeCleanup changes close ownership after registering the defer.
func (fixture *cleanupFixture) closeBeforeCleanup() (result publicationResult) {
	defer fixture.owner.finish(&result)
	fixture.owner.closeAttempted = true
	return publicationResult{stage: publicationCloseFailed,
		err: fixture.publication.file.Close()}
}

// TestTemporaryCleanupClosesBeforeRemoving checks the owned-resource order.
func TestTemporaryCleanupClosesBeforeRemoving(t *testing.T) {
	fixture := cleanupTest(t)
	primary := errors.New("write refused")
	result := publicationResult{stage: publicationWriteFailed, err: primary}
	gomock.InOrder(fixture.publication.file.EXPECT().Close().Return(nil),
		fixture.publication.filesystem.EXPECT().Remove(pathSpec{
			path: fixture.publication.temporary}).Return(nil))
	fixture.owner.finish(&result)
	if result.stage != publicationWriteFailed || !errors.Is(result.err, primary) {
		t.Fatalf("cleanup changed the primary result: %+v", result)
	}
}

// TestTemporaryCleanupRemovesAfterClose checks the remaining namespace owner.
func TestTemporaryCleanupRemovesAfterClose(t *testing.T) {
	fixture := cleanupTest(t)
	fixture.owner.closeAttempted = true
	primary := errors.New("rename refused")
	result := publicationResult{stage: publicationRenameUncertain, err: primary}
	fixture.publication.filesystem.EXPECT().Remove(pathSpec{
		path: fixture.publication.temporary}).Return(nil)
	fixture.owner.finish(&result)
	if result.stage != publicationRenameUncertain || !errors.Is(result.err, primary) {
		t.Fatalf("cleanup changed rename uncertainty: %+v", result)
	}
}

// TestTemporaryCleanupKeepsEveryCause checks primary and cleanup refusals.
func TestTemporaryCleanupKeepsEveryCause(t *testing.T) {
	fixture := cleanupTest(t)
	primary := errors.New("write refused")
	closeCause := errors.New("close refused")
	removeCause := errors.New("remove refused")
	result := publicationResult{stage: publicationWriteFailed, err: primary}
	gomock.InOrder(fixture.publication.file.EXPECT().Close().Return(closeCause),
		fixture.publication.filesystem.EXPECT().Remove(pathSpec{
			path: fixture.publication.temporary}).Return(removeCause))
	fixture.owner.finish(&result)
	if result.stage != publicationWriteFailed || !errors.Is(result.err, primary) ||
		!errors.Is(result.err, closeCause) || !errors.Is(result.err, removeCause) {
		t.Fatalf("cleanup lost a cause or changed the stage: %+v", result)
	}
}

// TestTemporaryCleanupLeavesVisibleAlone forbids close, remove and rename.
func TestTemporaryCleanupLeavesVisibleAlone(t *testing.T) {
	fixture := cleanupTest(t)
	result := publicationResult{stage: publicationVisible}
	fixture.owner.finish(&result)
	if result.stage != publicationVisible || result.err != nil {
		t.Fatalf("cleanup changed a visible result: %+v", result)
	}
}

// TestTemporaryCleanupNeverRetriesFailedClose catches a copied defer receiver.
func TestTemporaryCleanupNeverRetriesFailedClose(t *testing.T) {
	fixture := cleanupTest(t)
	primary := errors.New("close refused")
	gomock.InOrder(fixture.publication.file.EXPECT().Close().Return(primary),
		fixture.publication.filesystem.EXPECT().Remove(pathSpec{
			path: fixture.publication.temporary}).Return(nil))
	result := fixture.closeBeforeCleanup()
	if result.stage != publicationCloseFailed || !errors.Is(result.err, primary) {
		t.Fatalf("cleanup changed the failed close: %+v", result)
	}
}

// TestTemporaryCleanupNeverResolvesRename keeps uncertainty on cleanup refusal.
func TestTemporaryCleanupNeverResolvesRename(t *testing.T) {
	fixture := cleanupTest(t)
	fixture.owner.closeAttempted = true
	primary := errors.New("rename refused")
	removeCause := errors.New("remove refused")
	result := publicationResult{stage: publicationRenameUncertain, err: primary}
	fixture.publication.filesystem.EXPECT().Remove(pathSpec{
		path: fixture.publication.temporary}).Return(removeCause)
	fixture.owner.finish(&result)
	if result.stage != publicationRenameUncertain || !errors.Is(result.err, primary) ||
		!errors.Is(result.err, removeCause) {
		t.Fatalf("cleanup resolved uncertainty or lost a cause: %+v", result)
	}
}

// TestTemporaryCleanupNeverPublishesOnRemovalFailure forbids invented success.
func TestTemporaryCleanupNeverPublishesOnRemovalFailure(t *testing.T) {
	fixture := cleanupTest(t)
	removeCause := errors.New("remove refused")
	result := publicationResult{stage: publicationUnknown}
	gomock.InOrder(fixture.publication.file.EXPECT().Close().Return(nil),
		fixture.publication.filesystem.EXPECT().Remove(pathSpec{
			path: fixture.publication.temporary}).Return(removeCause))
	fixture.owner.finish(&result)
	if result.stage != publicationUnknown || !errors.Is(result.err, removeCause) {
		t.Fatalf("cleanup invented visibility or lost refusal: %+v", result)
	}
}

// cleanupProperty gives the generated witness a named trial owner.
type cleanupProperty struct{}

// check varies cleanup refusals while independently checking cause membership.
func (cleanupProperty) check(property *rapid.T) {
	fixture := cleanupTest(property)
	defer fixture.publication.controller.Finish()
	closeAttempted := rapid.Bool().Draw(property, "close-attempted")
	fixture.owner.closeAttempted = closeAttempted
	closeFails := rapid.Bool().Draw(property, "close-fails")
	removeFails := rapid.Bool().Draw(property, "remove-fails")
	renameFailed := rapid.Bool().Draw(property, "rename-failed")
	primary := errors.New("primary refused")
	closeCause := errors.New("close refused")
	removeCause := errors.New("remove refused")
	stage := publicationWriteFailed
	if renameFailed {
		stage = publicationRenameUncertain
	}
	result := publicationResult{stage: stage, err: primary}
	var closeError, removeError error
	if closeFails {
		closeError = closeCause
	}
	if removeFails {
		removeError = removeCause
	}
	var closeCall *gomock.Call
	if !closeAttempted {
		closeCall = fixture.publication.file.EXPECT().Close().Return(closeError)
	}
	removeCall := fixture.publication.filesystem.EXPECT().Remove(pathSpec{
		path: fixture.publication.temporary}).Return(removeError)
	if closeCall != nil {
		removeCall.After(closeCall)
	}
	fixture.owner.finish(&result)
	if result.stage != stage || !errors.Is(result.err, primary) {
		property.Fatalf("cleanup changed the primary result: %+v", result)
	}
	wantClose := !closeAttempted && closeFails
	if errors.Is(result.err, closeCause) != wantClose ||
		errors.Is(result.err, removeCause) != removeFails {
		property.Fatalf("cleanup causes differ from actual refusals: %+v", result)
	}
}

// TestTemporaryCleanupGeneratedCauses checks the generated ownership cases.
func TestTemporaryCleanupGeneratedCauses(t *testing.T) {
	rapid.Check(t, cleanupProperty{}.check)
}
