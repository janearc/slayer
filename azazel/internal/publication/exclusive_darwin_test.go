package publication

import (
	"bytes"
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"golang.org/x/sys/unix"
	"pgregory.net/rapid"
)

//go:embed testdata/exclusive.yaml
var exclusiveData []byte

// noWaitNanos is defined as the duration boundary between waiting and no wait.
const noWaitNanos int64 = 0

// noAttemptResults is defined as the cardinality of an empty result slice.
const noAttemptResults = 0

// noInstallWinners is defined as the count before any successful install.
const noInstallWinners = 0

// exclusiveWinnerCount is defined by the single-winner installation contract.
const exclusiveWinnerCount = 1

// exclusiveInputs supplies the bounded wait for the two native contenders.
type exclusiveInputs struct {
	WaitNanos int64 `yaml:"wait_ns"`
}

// exclusiveWait refuses missing or nonpositive fixture timing.
func exclusiveWait(t *testing.T) time.Duration {
	t.Helper()
	var inputs exclusiveInputs
	if err := yamlfile.Read("testdata/exclusive.yaml", exclusiveData,
		&inputs); err != nil {
		t.Fatal(err)
	}
	if inputs.WaitNanos <= noWaitNanos {
		t.Fatal("exclusive install needs a positive bounded receive")
	}
	return time.Duration(inputs.WaitNanos)
}

// exclusiveSource retains a closed source's authored bytes and identity.
type exclusiveSource struct {
	request installSpec
	body    []byte
	entry   os.FileInfo
}

// newExclusiveSource prepares only an owned tiny same-directory file.
func newExclusiveSource(t *testing.T, spec nativeFixtureSpec) exclusiveSource {
	t.Helper()
	file := namedNativeFixture(t, spec)
	entry := nativeFileIdentity(t, file)
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return exclusiveSource{
		request: installSpec{from: file.Name(),
			to: filepath.Join(spec.parent, "result")},
		body: spec.body, entry: entry,
	}
}

// assertStillOwned catches source deletion or replacement after refusal.
func (source exclusiveSource) assertStillOwned(t *testing.T) {
	t.Helper()
	assertNativeSameEntry(t, source.request.from, source.entry)
	assertNativeBytes(t, source.request.from, source.body)
}

// assertInstalled checks both identity and complete bytes, not only existence.
func (source exclusiveSource) assertInstalled(t *testing.T) {
	t.Helper()
	assertNativeSameEntry(t, source.request.to, source.entry)
	assertNativeBytes(t, source.request.to, source.body)
	if _, err := os.Lstat(source.request.from); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("installed source name remains: %v", err)
	}
}

// TestExclusiveInstallMovesTheFirstEntry needs real namespace identity.
func TestExclusiveInstallMovesTheFirstEntry(t *testing.T) {
	parent := t.TempDir()
	source := newExclusiveSource(t, nativeFixtureSpec{
		parent: parent, name: "source", body: payloadExamples(t).RenameSource,
	})
	reader, err := os.Open(source.request.from)
	if err != nil {
		t.Fatal(err)
	}
	owner := nativeHandleCleanup{reporter: t, close: reader.Close}
	t.Cleanup(owner.finishIfOpen)
	sibling := nativeSibling(t, parent)
	if err := (localFilesystem{}).Install(source.request); err != nil {
		t.Fatal(err)
	}
	source.assertInstalled(t)
	assertNativeHandleBytes(t, reader, source.body)
	assertNativeSibling(t, sibling)
	assertNativeEntryNames(t, parent, []string{"keep", "result"})
}

// TestExclusiveInstallKeepsAnIncumbent is the paired overwrite anti-test.
func TestExclusiveInstallKeepsAnIncumbent(t *testing.T) {
	parent := t.TempDir()
	source := newExclusiveSource(t, nativeFixtureSpec{
		parent: parent, name: "source", body: payloadExamples(t).RenameSource,
	})
	incumbent := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "result", body: []byte("first authored result"),
	})
	identity := nativeFileIdentity(t, incumbent)
	err := (localFilesystem{}).Install(source.request)
	if !errors.Is(err, unix.EEXIST) {
		t.Fatalf("incumbent refusal: %v, want native existing-entry cause", err)
	}
	source.assertStillOwned(t)
	assertNativeSameEntry(t, source.request.to, identity)
	assertNativeHandleBytes(t, incumbent, []byte("first authored result"))
	assertNativeBytes(t, source.request.to, []byte("first authored result"))
}

// exclusiveAttempt reports from a worker without calling testing methods.
type exclusiveAttempt struct {
	source exclusiveSource
	err    error
}

// exclusiveContender waits for the shared gate and reports exactly once.
type exclusiveContender struct {
	source exclusiveSource
	start  <-chan struct{}
	done   chan<- exclusiveAttempt
}

// run performs the actual competing native operation after gate release.
func (contender exclusiveContender) run() {
	<-contender.start
	err := (localFilesystem{}).Install(contender.source.request)
	contender.done <- exclusiveAttempt{source: contender.source, err: err}
}

// competingInstalls starts one worker per authored source before releasing them.
func competingInstalls(t *testing.T, sources []exclusiveSource) []exclusiveAttempt {
	t.Helper()
	start := make(chan struct{})
	done := make(chan exclusiveAttempt, len(sources))
	for _, source := range sources {
		contender := exclusiveContender{source: source, start: start, done: done}
		go contender.run()
	}
	deadline := time.NewTimer(exclusiveWait(t))
	defer deadline.Stop()
	close(start)
	results := make([]exclusiveAttempt, noAttemptResults, len(sources))
	for range sources {
		select {
		case result := <-done:
			results = append(results, result)
		case <-deadline.C:
			t.Fatal("exclusive install workers exceeded bounded receive")
		}
	}
	return results
}

// exclusivePair keeps two different authored byte sequences and inodes.
func exclusivePair(t *testing.T, parent string) []exclusiveSource {
	t.Helper()
	examples := payloadExamples(t)
	return []exclusiveSource{
		newExclusiveSource(t, nativeFixtureSpec{
			parent: parent, name: "left", body: examples.RenameSource,
		}),
		newExclusiveSource(t, nativeFixtureSpec{
			parent: parent, name: "right", body: examples.NativePublication.Binary,
		}),
	}
}

// TestExclusiveInstallHasOneCompetingWinner needs actual kernel exclusion.
func TestExclusiveInstallHasOneCompetingWinner(t *testing.T) {
	sources := exclusivePair(t, t.TempDir())
	results := competingInstalls(t, sources)
	winners := noInstallWinners
	for _, result := range results {
		if result.err == nil {
			winners++
			result.source.assertInstalled(t)
			continue
		}
		if !errors.Is(result.err, unix.EEXIST) {
			t.Fatalf("competing refusal: %v", result.err)
		}
		result.source.assertStillOwned(t)
	}
	// One winner is the defined cardinality of exclusive installation.
	if winners != exclusiveWinnerCount {
		t.Fatalf("competing winners: %d, want exactly one", winners)
	}
}

// TestExclusiveInstallRacersCannotReplaceAnIncumbent is the paired anti-test.
func TestExclusiveInstallRacersCannotReplaceAnIncumbent(t *testing.T) {
	parent := t.TempDir()
	sources := exclusivePair(t, parent)
	incumbent := namedNativeFixture(t, nativeFixtureSpec{
		parent: parent, name: "result", body: []byte("incumbent"),
	})
	identity := nativeFileIdentity(t, incumbent)
	for _, result := range competingInstalls(t, sources) {
		if !errors.Is(result.err, unix.EEXIST) {
			t.Fatalf("racer touched incumbent: %v", result.err)
		}
		result.source.assertStillOwned(t)
	}
	assertNativeSameEntry(t, incumbent.Name(), identity)
	assertNativeBytes(t, incumbent.Name(), []byte("incumbent"))
}

// TestExclusiveInstallKeepsSpecialEntries pins existing directory/link names.
func TestExclusiveInstallKeepsSpecialEntries(t *testing.T) {
	t.Run("directory", checkExclusiveDirectory)
	t.Run("dangling link", checkExclusiveDanglingLink)
	t.Run("authored link target", checkExclusiveLinkedTarget)
}

// checkExclusiveDirectory retains the directory inode and its child.
func checkExclusiveDirectory(t *testing.T) {
	parent := t.TempDir()
	source := newExclusiveSource(t, nativeFixtureSpec{
		parent: parent, name: "source", body: payloadExamples(t).RenameSource,
	})
	if err := os.Mkdir(source.request.to, 0o700); err != nil {
		t.Fatal(err)
	}
	identity, err := os.Stat(source.request.to)
	if err != nil {
		t.Fatal(err)
	}
	child := nativeSibling(t, source.request.to)
	if err := (localFilesystem{}).Install(source.request); !errors.Is(err, unix.EEXIST) {
		t.Fatalf("existing directory refusal: %v", err)
	}
	source.assertStillOwned(t)
	assertNativeSameEntry(t, source.request.to, identity)
	assertNativeSibling(t, child)
}

// checkExclusiveDanglingLink must refuse the link even when its target is absent.
func checkExclusiveDanglingLink(t *testing.T) {
	checkExclusiveLink(t, false)
}

// checkExclusiveLinkedTarget also keeps a real target's authored bytes.
func checkExclusiveLinkedTarget(t *testing.T) {
	checkExclusiveLink(t, true)
}

// checkExclusiveLink compares the link inode itself, not os.Stat of its target.
func checkExclusiveLink(t *testing.T, targetExists bool) {
	parent := t.TempDir()
	source := newExclusiveSource(t, nativeFixtureSpec{
		parent: parent, name: "source", body: payloadExamples(t).RenameSource,
	})
	target := filepath.Join(parent, "target")
	if targetExists {
		namedNativeFixture(t, nativeFixtureSpec{
			parent: parent, name: "target", body: []byte("linked target"),
		})
	}
	if err := os.Symlink(target, source.request.to); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(source.request.to)
	if err != nil {
		t.Fatal(err)
	}
	if err := (localFilesystem{}).Install(source.request); !errors.Is(err, unix.EEXIST) {
		t.Fatalf("existing link refusal: %v", err)
	}
	after, err := os.Lstat(source.request.to)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("link entry changed: %v", err)
	}
	got, err := os.Readlink(source.request.to)
	if err != nil || got != target {
		t.Fatalf("link target changed: %q, %v", got, err)
	}
	source.assertStillOwned(t)
	if targetExists {
		assertNativeBytes(t, target, []byte("linked target"))
		return
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dangling target was created: %v", err)
	}
}

// TestExclusiveInstallRetainsMissingAndInvalidCauses is the paired anti-test.
func TestExclusiveInstallRetainsMissingAndInvalidCauses(t *testing.T) {
	parent := t.TempDir()
	request := installSpec{from: filepath.Join(parent, "absent"),
		to: filepath.Join(parent, "result")}
	if err := (localFilesystem{}).Install(request); !errors.Is(err, unix.ENOENT) {
		t.Fatalf("missing source cause: %v", err)
	}
	source := newExclusiveSource(t, nativeFixtureSpec{
		parent: parent, name: "source", body: payloadExamples(t).RenameSource,
	})
	// x/sys rejects NUL during string conversion before the native syscall.
	// These cases keep the wrapper's refusal. They do not check the kernel.
	invalidFrom := source.request
	invalidFrom.from += "\x00"
	invalidTo := source.request
	invalidTo.to += "\x00"
	for _, invalid := range []installSpec{invalidFrom, invalidTo} {
		if err := (localFilesystem{}).Install(invalid); !errors.Is(err, unix.EINVAL) {
			t.Fatalf("invalid path cause: %v", err)
		}
	}
	source.assertStillOwned(t)
	assertNativeEntryNames(t, parent, []string{"source"})
}

// exclusiveBytesProperty keeps each trial inside the test's private root.
type exclusiveBytesProperty struct {
	parent           string
	minimum, maximum int
}

// exclusiveByteTrial holds independent source and incumbent byte sequences.
type exclusiveByteTrial struct {
	request   installSpec
	body      []byte
	incumbent []byte
	parent    string
	reporter  *rapid.T
}

// trial creates fresh files using the real Rapid reporter for failures.
func (bounds exclusiveBytesProperty) trial(property *rapid.T) exclusiveByteTrial {
	body := rapid.SliceOfN(rapid.Byte(), bounds.minimum,
		bounds.maximum).Draw(property, "source bytes")
	incumbent := rapid.SliceOfN(rapid.Byte(), bounds.minimum,
		bounds.maximum).Draw(property, "incumbent bytes")
	parent, err := os.MkdirTemp(bounds.parent, "trial-")
	if err != nil {
		property.Fatal(err)
	}
	trial := exclusiveByteTrial{
		request: installSpec{from: filepath.Join(parent, "source"),
			to: filepath.Join(parent, "result")},
		body: body, incumbent: incumbent, parent: parent, reporter: property,
	}
	// Rapid runs this named cleanup for every trial, including shrinks.
	property.Cleanup(trial.remove)
	if err := os.WriteFile(trial.request.from, body,
		ownerReadWritePermissions); err != nil {
		property.Fatal(err)
	}
	return trial
}

// remove owns only the unique directory made by this Rapid trial.
func (trial exclusiveByteTrial) remove() {
	if err := os.RemoveAll(trial.parent); err != nil {
		trial.reporter.Errorf("remove owned trial: %v", err)
	}
}

// first checks arbitrary bytes, including an empty file, after first install.
func (bounds exclusiveBytesProperty) first(property *rapid.T) {
	trial := bounds.trial(property)
	if err := (localFilesystem{}).Install(trial.request); err != nil {
		property.Fatal(err)
	}
	got, err := os.ReadFile(trial.request.to)
	if err != nil || !bytes.Equal(got, trial.body) {
		property.Fatalf("installed bytes: %x, %v, want %x", got, err, trial.body)
	}
	if _, err := os.Lstat(trial.request.from); !errors.Is(err, os.ErrNotExist) {
		property.Fatalf("source name remains: %v", err)
	}
}

// keep checks that neither independently generated byte sequence is lost.
func (bounds exclusiveBytesProperty) keep(property *rapid.T) {
	trial := bounds.trial(property)
	if err := os.WriteFile(trial.request.to, trial.incumbent,
		ownerReadWritePermissions); err != nil {
		property.Fatal(err)
	}
	if err := (localFilesystem{}).Install(trial.request); !errors.Is(err, unix.EEXIST) {
		property.Fatalf("incumbent refusal: %v", err)
	}
	source, sourceErr := os.ReadFile(trial.request.from)
	got, err := os.ReadFile(trial.request.to)
	if sourceErr != nil || !bytes.Equal(source, trial.body) ||
		err != nil || !bytes.Equal(got, trial.incumbent) {
		property.Fatalf("source=%x (%v), incumbent=%x (%v)",
			source, sourceErr, got, err)
	}
}

// TestExclusiveInstallGeneratedBytesArriveWhole includes binary and empty files.
func TestExclusiveInstallGeneratedBytesArriveWhole(t *testing.T) {
	examples := payloadExamples(t)
	property := exclusiveBytesProperty{parent: t.TempDir(),
		minimum: examples.MinimumGeneratedBytes,
		maximum: examples.MaximumGeneratedBytes}
	rapid.Check(t, property.first)
}

// TestExclusiveInstallGeneratedIncumbentsRemain is the paired anti-property.
func TestExclusiveInstallGeneratedIncumbentsRemain(t *testing.T) {
	examples := payloadExamples(t)
	property := exclusiveBytesProperty{parent: t.TempDir(),
		minimum: examples.MinimumGeneratedBytes,
		maximum: examples.MaximumGeneratedBytes}
	rapid.Check(t, property.keep)
}
