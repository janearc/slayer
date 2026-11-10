package publication

import (
	_ "embed"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// authoredPayloadData keeps the original byte examples in checked local data.
//
//go:embed testdata/payloads.yaml
var authoredPayloadData []byte

// noWrittenBytes is defined as the byte count of an empty write.
const noWrittenBytes = 0

// singleByteShortfall is defined as the gap one byte below a complete write.
const singleByteShortfall = 1

// nativePayloadCase retains one authored label and its read-only payload.
type nativePayloadCase struct {
	name string
	body []byte
}

// nativePayloadData keeps each binary witness separate from its nil empty one.
type nativePayloadData struct {
	Binary []byte `yaml:"binary"`
	Empty  []byte `yaml:"empty"`
}

// cases keeps the original binary-then-empty labels and execution order.
func (examples nativePayloadData) cases() []nativePayloadCase {
	return []nativePayloadCase{
		{name: "binary", body: examples.Binary},
		{name: "empty", body: examples.Empty},
	}
}

// requireShapes refuses omitted binary inputs or a changed empty-slice shape.
func (examples nativePayloadData) requireShapes(t *testing.T) {
	t.Helper()
	if len(examples.Binary) == noWrittenBytes || examples.Empty != nil {
		t.Fatal("authored native payloads lost a binary or nil-empty example")
	}
}

// replacementPayloadData preserves all four original payload roles.
type replacementPayloadData struct {
	Nil    []byte `yaml:"nil_payload"`
	Empty  []byte `yaml:"empty_payload"`
	Binary []byte `yaml:"binary_payload"`
	Text   []byte `yaml:"text_payload"`
}

// bodies returns nil, allocated-empty, binary and text in their original order.
func (examples replacementPayloadData) bodies() [][]byte {
	return [][]byte{examples.Nil, examples.Empty, examples.Binary, examples.Text}
}

// requireShapes keeps a nil and an allocated empty write as distinct inputs.
func (examples replacementPayloadData) requireShapes(t *testing.T) {
	t.Helper()
	if examples.Nil != nil || examples.Empty == nil ||
		len(examples.Empty) != noWrittenBytes ||
		len(examples.Binary) == noWrittenBytes ||
		len(examples.Text) == noWrittenBytes {
		t.Fatal("authored replacement payloads lost a distinct input role")
	}
}

// publicationPayloadData decodes bytes and lengths without a custom parser.
type publicationPayloadData struct {
	LocalWrite            nativePayloadData      `yaml:"local_write"`
	NativePublication     nativePayloadData      `yaml:"native_publication"`
	RenameSource          []byte                 `yaml:"rename_source"`
	Replacement           replacementPayloadData `yaml:"replacement"`
	FailedWriteCount      int                    `yaml:"failed_write_count"`
	ShortWriteGap         int                    `yaml:"short_write_gap"`
	MinimumGeneratedBytes int                    `yaml:"minimum_generated_bytes"`
	MaximumGeneratedBytes int                    `yaml:"maximum_generated_bytes"`
}

// payloadExamples refuses lost payload roles, the one-byte gap or empty bound.
func payloadExamples(t *testing.T) publicationPayloadData {
	t.Helper()
	var examples publicationPayloadData
	if err := yamlfile.Read("testdata/payloads.yaml", authoredPayloadData,
		&examples); err != nil {
		t.Fatal(err)
	}
	examples.LocalWrite.requireShapes(t)
	examples.NativePublication.requireShapes(t)
	examples.Replacement.requireShapes(t)
	if len(examples.RenameSource) == noWrittenBytes ||
		examples.ShortWriteGap != singleByteShortfall ||
		examples.MinimumGeneratedBytes != noWrittenBytes ||
		examples.MaximumGeneratedBytes <= noWrittenBytes {
		t.Fatal("authored publication examples lost a byte or generation boundary")
	}
	return examples
}

// requirePartial checks the authored counts against the actual fixture body.
func (examples publicationPayloadData) requirePartial(t *testing.T,
	body []byte) {
	t.Helper()
	if examples.FailedWriteCount <= noWrittenBytes ||
		examples.FailedWriteCount >= len(body) ||
		examples.ShortWriteGap >= len(body) {
		t.Fatal("authored partial-write examples need a nonempty incomplete write")
	}
}

// publicationFixture keeps generated boundaries and authored paths together.
type publicationFixture struct {
	controller *gomock.Controller
	filesystem *Mockfilesystem
	file       *Mocktemporary
	publisher  publisher
	request    replacement
	temporary  string
}

// publicationTest creates mocks without granting any filesystem calls.
func publicationTest(reporter gomock.TestReporter) publicationFixture {
	controller := gomock.NewController(reporter)
	filesystem := NewMockfilesystem(controller)
	return publicationFixture{
		controller: controller,
		filesystem: filesystem, file: NewMocktemporary(controller),
		publisher: publisher{filesystem: filesystem},
		request: replacement{parent: "/queue", name: "state",
			body: []byte("snapshot")},
		temporary: "/queue/.state-private",
	}
}

// creation expects a same-directory private file, not the destination itself.
func (fixture publicationFixture) creation() *gomock.Call {
	return fixture.filesystem.EXPECT().CreateTemp(tempSpec{
		parent: fixture.request.parent, pattern: "." + fixture.request.name + "-*",
	}).Return(openedTemp{file: fixture.file, name: fixture.temporary})
}

// publication expects the complete successful sequence and exact payload.
func (fixture publicationFixture) publication() {
	written := writeResult{written: len(fixture.request.body)}
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(writeSpec{body: fixture.request.body}).
			Return(written),
		fixture.file.EXPECT().Close().Return(nil),
		fixture.filesystem.EXPECT().Rename(renameSpec{
			from: fixture.temporary,
			to:   filepath.Join(fixture.request.parent, fixture.request.name),
		}).Return(nil))
}

// TestReplacementPublishesAuthoredPayloads checks binary bytes and empty files.
func TestReplacementPublishesAuthoredPayloads(t *testing.T) {
	examples := payloadExamples(t)
	for _, body := range examples.Replacement.bodies() {
		fixture := publicationTest(t)
		fixture.request.body = body
		fixture.publication()
		got := fixture.publisher.replace(fixture.request)
		if got.stage != publicationVisible || got.err != nil {
			t.Fatalf("publication: %+v", got)
		}
	}
}

// TestReplacementRefusesInvalidPaths is the anti-test: refusal precedes I/O.
func TestReplacementRefusesInvalidPaths(t *testing.T) {
	for _, request := range []replacement{
		{parent: "relative", name: "state"}, {name: "state"},
		{parent: "/queue"}, {parent: "/queue", name: "."},
		{parent: "/queue", name: ".."},
		{parent: "/queue", name: "../state"},
		{parent: "/queue", name: "/state"},
		{parent: "/queue", name: "state\x00suffix"},
	} {
		fixture := publicationTest(t)
		got := fixture.publisher.replace(request)
		if got.stage != publicationRefused || got.err == nil {
			t.Errorf("accepted invalid path %+v: %+v", request, got)
		}
	}
}

// TestReplacementCreatesBesideTheDestination checks parent and basename.
func TestReplacementCreatesBesideTheDestination(t *testing.T) {
	fixture := publicationTest(t)
	fixture.request.parent = "/queue/nested"
	fixture.request.name = "state.v2"
	fixture.temporary = "/queue/nested/.state.v2-private"
	fixture.publication()
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationVisible {
		t.Fatalf("nested publication: %+v", got)
	}
}

// TestReplacementCreateFailureOwnsNothing is the anti-test: no close or remove.
func TestReplacementCreateFailureOwnsNothing(t *testing.T) {
	fixture := publicationTest(t)
	cause := errors.New("create refused")
	fixture.filesystem.EXPECT().CreateTemp(gomock.Any()).
		Return(openedTemp{err: cause})
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationCreateFailed || !errors.Is(got.err, cause) {
		t.Fatalf("create failure: %+v", got)
	}
}

// TestReplacementClosesBeforePublishing checks the boundary's order.
func TestReplacementClosesBeforePublishing(t *testing.T) {
	fixture := publicationTest(t)
	fixture.publication()
	if got := fixture.publisher.replace(fixture.request); got.err != nil {
		t.Fatal(got.err)
	}
}

// TestReplacementCloseFailureNeverRenames is the anti-test: no retry or rename.
func TestReplacementCloseFailureNeverRenames(t *testing.T) {
	fixture := publicationTest(t)
	cause := errors.New("close refused")
	written := writeResult{written: len(fixture.request.body)}
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(writeSpec{body: fixture.request.body}).
			Return(written),
		fixture.file.EXPECT().Close().Return(cause),
		fixture.filesystem.EXPECT().Remove(pathSpec{path: fixture.temporary}).
			Return(nil))
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationCloseFailed || !errors.Is(got.err, cause) {
		t.Fatalf("close failure: %+v", got)
	}
}

// TestReplacementWriteFailureCleansItsTemporary checks ownership after create.
func TestReplacementWriteFailureCleansItsTemporary(t *testing.T) {
	examples := payloadExamples(t)
	fixture := publicationTest(t)
	examples.requirePartial(t, fixture.request.body)
	cause := errors.New("write refused")
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(writeSpec{body: fixture.request.body}).
			Return(writeResult{written: examples.FailedWriteCount, err: cause}),
		fixture.file.EXPECT().Close().Return(nil),
		fixture.filesystem.EXPECT().Remove(pathSpec{path: fixture.temporary}).
			Return(nil))
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationWriteFailed || !errors.Is(got.err, cause) {
		t.Fatalf("write failure: %+v", got)
	}
}

// TestReplacementShortWriteNeverPublishes is the anti-test: require all bytes.
func TestReplacementShortWriteNeverPublishes(t *testing.T) {
	examples := payloadExamples(t)
	fixture := publicationTest(t)
	examples.requirePartial(t, fixture.request.body)
	partial := writeResult{
		written: len(fixture.request.body) - examples.ShortWriteGap}
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(writeSpec{body: fixture.request.body}).
			Return(partial),
		fixture.file.EXPECT().Close().Return(nil),
		fixture.filesystem.EXPECT().Remove(pathSpec{path: fixture.temporary}).
			Return(nil))
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationWriteFailed ||
		!errors.Is(got.err, io.ErrShortWrite) {
		t.Fatalf("short write: %+v", got)
	}
}

// TestReplacementPreservesAllFailureCauses checks primary and cleanup causes.
func TestReplacementPreservesAllFailureCauses(t *testing.T) {
	fixture := publicationTest(t)
	writeCause, closeCause := errors.New("write"), errors.New("close")
	removeCause := errors.New("remove")
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(gomock.Any()).
			Return(writeResult{err: writeCause}),
		fixture.file.EXPECT().Close().Return(closeCause),
		fixture.filesystem.EXPECT().Remove(pathSpec{path: fixture.temporary}).
			Return(removeCause))
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationWriteFailed {
		t.Fatalf("cleanup changed primary stage: %+v", got)
	}
	for _, cause := range []error{writeCause, closeCause, removeCause} {
		if !errors.Is(got.err, cause) {
			t.Errorf("lost cause %v in %v", cause, got.err)
		}
	}
}

// TestReplacementRenameFailureLeavesDestinationAlone is the anti-test:
// only our temporary name is eligible for cleanup.
func TestReplacementRenameFailureLeavesDestinationAlone(t *testing.T) {
	fixture := publicationTest(t)
	cause, cleanup := errors.New("rename"), errors.New("cleanup")
	written := writeResult{written: len(fixture.request.body)}
	gomock.InOrder(fixture.creation(),
		fixture.file.EXPECT().Write(gomock.Any()).
			Return(written),
		fixture.file.EXPECT().Close().Return(nil),
		fixture.filesystem.EXPECT().Rename(renameSpec{
			from: fixture.temporary, to: "/queue/state",
		}).Return(cause),
		fixture.filesystem.EXPECT().Remove(pathSpec{path: fixture.temporary}).
			Return(cleanup))
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationRenameUncertain ||
		!errors.Is(got.err, cause) || !errors.Is(got.err, cleanup) {
		t.Fatalf("rename failure: %+v", got)
	}
}

// publicationBytesProperty names byte preservation without capturing a parent.
type publicationBytesProperty struct {
	minimum, maximum int
}

// check owns fresh generated boundaries through the actual Rapid trial.
func (bounds publicationBytesProperty) check(property *rapid.T) {
	fixture := publicationTest(property)
	defer fixture.controller.Finish()
	fixture.request.body = rapid.SliceOfN(rapid.Byte(), bounds.minimum,
		bounds.maximum).
		Draw(property, "payload")
	fixture.publication()
	got := fixture.publisher.replace(fixture.request)
	if got.stage != publicationVisible || got.err != nil {
		property.Fatalf("byte publication: %+v", got)
	}
}

// TestReplacementGeneratedBytesArriveWhole checks byte preservation.
func TestReplacementGeneratedBytesArriveWhole(t *testing.T) {
	examples := payloadExamples(t)
	property := publicationBytesProperty{minimum: examples.MinimumGeneratedBytes,
		maximum: examples.MaximumGeneratedBytes}
	rapid.Check(t, property.check)
}

// TestReplacementZeroValuesNeverPublish is the anti-test: no zero authority.
func TestReplacementZeroValuesNeverPublish(t *testing.T) {
	if (publicationResult{}).stage == publicationVisible {
		t.Fatal("zero result claims publication")
	}
	got := (publisher{}).replace(replacement{parent: "/queue", name: "state"})
	if got.stage != publicationRefused || got.err == nil {
		t.Fatalf("zero publisher: %+v", got)
	}
}
