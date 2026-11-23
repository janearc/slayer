package sources

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// coverSum is the sha256 of text in hex.
func coverSum(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:])
}

// coverZip is a zip holding one member of the given text.
func coverZip(t *testing.T, member, text string) []byte {
	t.Helper()
	var held bytes.Buffer
	writer := zip.NewWriter(&held)
	entry, err := writer.Create(member)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return held.Bytes()
}

// stubOrigin is an origin whose answers a test sets.
type stubOrigin struct {
	products    []*Product
	productsErr error
	body        []byte
	fetchPath   string
	declared    *Declaration
	declaredErr error
}

// Name is the stub's name.
func (stub *stubOrigin) Name() string { return "stub" }

// Products gives the products or the error the test set.
func (stub *stubOrigin) Products(string) ([]*Product, error) {
	return stub.products, stub.productsErr
}

// Fetch writes the body under the product's name, or answers the path the
// test set, which need not exist.
func (stub *stubOrigin) Fetch(_ context.Context, product *Product, into string) (string, error) {
	if stub.fetchPath != "" {
		return stub.fetchPath, nil
	}
	at := filepath.Join(into, product.name)
	return at, os.WriteFile(at, stub.body, 0o644)
}

// Declared gives the declaration or the error the test set.
func (stub *stubOrigin) Declared(context.Context, *Product) (*Declaration, error) {
	return stub.declared, stub.declaredErr
}

// coverRun tends one want in a mode and returns what was said and the error.
func coverRun(shelf Shelf, origin Origin, mode Mode, file string) (string, error) {
	var said bytes.Buffer
	err := shelf.Tend(context.Background(), origin, mode, []Want{NewWant("d", file)}, &said)
	return said.String(), err
}

// coverMust checks that an error holds the text.
func coverMust(t *testing.T, name string, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("%s: error %v, want one holding %q", name, err, want)
	}
}

// coverList reads a list of one archive source with the lines added to its
// unpack block.
func coverList(t *testing.T, unpack string) (List, error) {
	return listed(t, "sources:\n  - key: dem\n    from: http://example.invalid/dem.zip\n    licence: public domain\n    unpack: "+unpack+"\n")
}

// TestListRefusesABadUnpackBlock checks each refusal of an unpack block and
// the values a good one keeps.
func TestListRefusesABadUnpackBlock(t *testing.T) {
	good := "{member: kil/kil.tif, sha256: " + coverSum("tif") + ", bytes: 3}"
	list, err := coverList(t, good)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := list.Find("dem")
	kept := productOf(source).member
	if kept == nil || kept.name != "kil/kil.tif" || kept.sum != coverSum("tif") || kept.size != 3 {
		t.Errorf("member kept as %+v", kept)
	}
	cases := map[string]string{
		"an unknown key":            "{member: a, sha256: " + coverSum("a") + ", bytes: 3, colour: red}",
		"not a block":               "5",
		"a hash of the wrong shape": "{member: a, sha256: nothex, bytes: 3}",
		"no bytes":                  "{member: a, sha256: " + coverSum("a") + ", bytes: 0}",
		"a member that leaves":      "{member: ../a, sha256: " + coverSum("a") + ", bytes: 3}",
	}
	wants := map[string]string{
		"an unknown key":            `the key "colour" is not member, sha256 or bytes`,
		"not a block":               "cannot unmarshal",
		"a hash of the wrong shape": `member sha256 "nothex"`,
		"no bytes":                  "member has 0 bytes",
		"a member that leaves":      "is not a path inside its folder",
	}
	for name, block := range cases {
		_, err := coverList(t, block)
		coverMust(t, name, err, wants[name])
	}
}

// TestListedOriginRefusesWhatItCannotFetch checks a source that is never
// fetched, a fetch into a place that cannot be made, a file already there,
// a request that cannot be made and a server that does not answer 200.
func TestListedOriginRefusesWhatItCannotFetch(t *testing.T) {
	server, _ := served(t)
	list, err := listed(t, "sources:\n  - {key: gone, never: withdrawn}\n  - {key: paper, from: "+server.URL+"/paper.pdf, licence: pd}\n")
	if err != nil {
		t.Fatal(err)
	}
	origin := NewListed(list, server.Client())
	_, err = origin.Products("gone")
	coverMust(t, "never", err, "gone: never fetched: withdrawn")
	products, err := origin.Products("paper")
	if err != nil || len(products) != 1 {
		t.Fatalf("products %v %v", products, err)
	}
	product := products[0]
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = origin.Fetch(context.Background(), product, filepath.Join(file, "inside"))
	coverMust(t, "folder under a file", err, "not a directory")
	into := t.TempDir()
	if err := os.WriteFile(filepath.Join(into, product.name), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = origin.Fetch(context.Background(), product, into)
	if !errors.Is(err, errFetch) || !strings.Contains(err.Error(), "never replaces a file") {
		t.Errorf("a file already there: %v", err)
	}
	bad := NewProduct("paper", "paper.pdf", "http://bad\x00url", "", 0)
	_, err = origin.Declared(context.Background(), bad)
	coverMust(t, "bad url", err, "invalid control character")
	missing := NewProduct("paper", "paper.pdf", server.URL+"/missing.pdf", "", 0)
	_, err = origin.Declared(context.Background(), missing)
	coverMust(t, "not found", err, "it answered 404 Not Found")
}

// TestTendSaysWhenAnOriginGivesNoOrManyProducts checks the refusals made
// before any file is looked at, in a fetch and in a check.
func TestTendSaysWhenAnOriginGivesNoOrManyProducts(t *testing.T) {
	shelf := NewShelf(t.TempDir(), nil)
	broken := &stubOrigin{productsErr: errors.New("no such dataset")}
	_, err := coverRun(shelf, broken, Fetching, "")
	coverMust(t, "fetch of an unknown dataset", err, "no such dataset")
	said, err := coverRun(shelf, broken, Checking, "")
	coverMust(t, "check of an unknown dataset", err, "1 of 1 datasets differ from stub or are missing")
	if !strings.Contains(said, "d: no such dataset") {
		t.Errorf("a check said %q", said)
	}
	two := &stubOrigin{products: []*Product{{dataset: "d", name: "a"}, {dataset: "d", name: "b"}}}
	_, err = coverRun(shelf, two, Fetching, "")
	coverMust(t, "two products", err, "d: stub holds 2 products, want one")
}

// TestFetchRefusesAFileThatIsNotAsPinned checks that a fetch leaves a wrong
// file alone and says so, that a directory in its place is an error, and that
// a pinned archive that is wrong is refused before it is unpacked.
func TestFetchRefusesAFileThatIsNotAsPinned(t *testing.T) {
	dir := t.TempDir()
	shelf := NewShelf(dir, nil)
	pinned := &stubOrigin{products: []*Product{{dataset: "d", name: "d.txt", from: "http://stub/d.txt", sum: coverSum("right")}}}
	if err := os.WriteFile(filepath.Join(dir, "d.txt"), []byte("wrong"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := coverRun(shelf, pinned, Fetching, "")
	coverMust(t, "a wrong file", err, "pins "+coverSum("right"))
	coverMust(t, "a wrong file names the flag", err, "--refresh-terrain replaces it")
	if raw, _ := os.ReadFile(filepath.Join(dir, "d.txt")); string(raw) != "wrong" {
		t.Errorf("the wrong file was replaced by %q", raw)
	}
	holder := t.TempDir()
	if err := os.Mkdir(filepath.Join(holder, "d.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = coverRun(NewShelf(holder, nil), pinned, Fetching, "")
	coverMust(t, "a directory in its place", err, "is a directory")
	archive := []byte("not what is pinned")
	member := &packed{name: "m.tif", sum: coverSum("tif"), size: 3}
	archived := &stubOrigin{products: []*Product{{dataset: "d", name: "d.zip", from: "http://stub/d.zip", sum: coverSum("the archive"), member: member}}}
	unpacking := t.TempDir()
	if err := os.WriteFile(filepath.Join(unpacking, "d.zip"), archive, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = coverRun(NewShelf(unpacking, nil), archived, Fetching, "unpacked/m.tif")
	coverMust(t, "an archive that is not as pinned", err, "pins "+coverSum("the archive"))
}

// TestFetchRefusesWhatCannotBeMadeOrRead checks the errors of a fetch whose
// shelf cannot be written to, and of an origin that names a file that is not
// there.
func TestFetchRefusesWhatCannotBeMadeOrRead(t *testing.T) {
	product := &Product{dataset: "d", name: "d.txt", from: "http://stub/d.txt"}
	stub := &stubOrigin{products: []*Product{product}, body: []byte("body")}
	if os.Geteuid() != 0 {
		parent := t.TempDir()
		if err := os.Chmod(parent, 0o555); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(parent, 0o755)
		inside := NewShelf(filepath.Join(parent, "shelf"), nil)
		_, err := coverRun(inside, stub, Fetching, "")
		coverMust(t, "a shelf that cannot be made", err, "permission denied")
		_, err = coverRun(NewShelf(parent, nil), stub, Fetching, "")
		coverMust(t, "a folder that cannot be made", err, "permission denied")
	}
	lost := &stubOrigin{products: []*Product{product}, fetchPath: filepath.Join(t.TempDir(), "nowhere")}
	bare := NewShelf(t.TempDir(), nil)
	_, err := coverRun(bare, lost, Fetching, "")
	coverMust(t, "a file that is not there", err, "no such file")
}

// TestRefreshOfAnArchiveSaysWhatItUnpackedAndWhatChanged checks a refresh
// that replaces an archive and unpacks its member, then one whose archive is
// not a zip, and one whose old files cannot be read.
func TestRefreshOfAnArchiveSaysWhatItUnpackedAndWhatChanged(t *testing.T) {
	body := coverZip(t, "m.tif", "tif")
	member := &packed{name: "m.tif", sum: coverSum("tif"), size: 3}
	product := &Product{dataset: "d", name: "d.zip", from: "http://stub/d.zip", member: member}
	dir := t.TempDir()
	shelf := NewShelf(dir, nil)
	stub := &stubOrigin{products: []*Product{product}, body: body}
	said, err := coverRun(shelf, stub, Refreshing, "unpacked/m.tif")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"refreshed", "it was not on the shelf", "unpacked m.tif to " + filepath.Join(dir, "unpacked", "m.tif")} {
		if !strings.Contains(said, want) {
			t.Errorf("the refresh said %q, want it to hold %q", said, want)
		}
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "unpacked", "m.tif")); string(raw) != "tif" {
		t.Errorf("unpacked %q", raw)
	}
	said, err = coverRun(shelf, stub, Refreshing, "unpacked/m.tif")
	if err != nil || strings.Count(said, "same bytes as before") != 2 {
		t.Errorf("a second refresh said %q, %v", said, err)
	}
	stub.body = []byte("not a zip")
	unpacking := NewShelf(t.TempDir(), nil)
	_, err = coverRun(unpacking, stub, Refreshing, "unpacked/m.tif")
	coverMust(t, "an archive that is not a zip", err, "d: ")
	archiveDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(archiveDir, "d.zip"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = coverRun(NewShelf(archiveDir, nil), stub, Refreshing, "")
	coverMust(t, "an old archive that is a directory", err, "is a directory")
	readDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(readDir, "unpacked", "m.tif"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = coverRun(NewShelf(readDir, nil), stub, Refreshing, "unpacked/m.tif")
	coverMust(t, "an old member that is a directory", err, "is a directory")
}

// TestCheckSaysWhatItCouldNotCompare checks the sha256 a list does not pin,
// a member whose archive is not on the shelf, and an archive path that
// cannot be examined, each said and none counted as a difference.
func TestCheckSaysWhatItCouldNotCompare(t *testing.T) {
	dir := t.TempDir()
	shelf := NewShelf(dir, nil)
	if err := os.WriteFile(filepath.Join(dir, "d.txt"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	unpinned := &stubOrigin{products: []*Product{{dataset: "d", name: "d.txt"}}, declared: NewDeclaration(4, true, "", "HEAD x")}
	said, err := coverRun(shelf, unpinned, Checking, "")
	if err != nil || !strings.Contains(said, "sha256 not compared: stub pins none; the file's is "+coverSum("text")) {
		t.Errorf("an unpinned check: %q, %v", said, err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "unpacked"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unpacked", "m.tif"), []byte("tif"), 0o644); err != nil {
		t.Fatal(err)
	}
	member := &packed{name: "m.tif", sum: coverSum("tif"), size: 3}
	absent := &stubOrigin{products: []*Product{{dataset: "d", name: "d.zip", member: member}}, declared: NewDeclaration(99, true, "", "HEAD y")}
	said, err = coverRun(shelf, absent, Checking, "unpacked/m.tif")
	archive := filepath.Join(dir, "d.zip")
	if err != nil || !strings.Contains(said, "size not compared: "+archive+" is not on the shelf; the server reports 99 bytes") {
		t.Errorf("a member with no archive: %q, %v", said, err)
	}
	long := strings.Repeat("n", 300)
	unreadable := &stubOrigin{products: []*Product{{dataset: "d", name: long, member: member}}, declared: NewDeclaration(99, true, "", "HEAD z")}
	said, err = coverRun(shelf, unreadable, Checking, "unpacked/m.tif")
	if err != nil || !strings.Contains(said, "size not compared: ") || strings.Contains(said, "is not on the shelf") {
		t.Errorf("an archive path that cannot be examined: %q, %v", said, err)
	}
	if _, err := coverRun(shelf, &stubOrigin{products: []*Product{{dataset: "d", name: "d.zip", member: member}}, declaredErr: errors.New("unreachable")}, Checking, "unpacked/m.tif"); err == nil {
		t.Error("a server that cannot be reached was not an error")
	}
	dirOnly := t.TempDir()
	if err := os.Mkdir(filepath.Join(dirOnly, "d.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	said, err = coverRun(NewShelf(dirOnly, nil), unpinned, Checking, "")
	coverMust(t, "a directory in a check", err, "1 of 1 datasets differ")
	if !strings.Contains(said, "is a directory") {
		t.Errorf("said %q", said)
	}
}

// TestUnpackMemberRefusesWhatIsNotAMember checks each way the unpacking of a
// member fails and that a good member is written in place.
func TestUnpackMemberRefusesWhatIsNotAMember(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.zip")
	if err := os.WriteFile(archive, coverZip(t, "m.tif", "tif"), 0o644); err != nil {
		t.Fatal(err)
	}
	member := &packed{name: "m.tif", sum: coverSum("tif"), size: 3}
	_, err := unpackMember(archive, &packed{name: "../m.tif", sum: member.sum, size: 3}, filepath.Join(dir, "out"))
	coverMust(t, "a member that leaves", err, "is not a path inside its folder")
	notZip := filepath.Join(dir, "plain.zip")
	if err := os.WriteFile(notZip, []byte("plain"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = unpackMember(notZip, member, filepath.Join(dir, "out"))
	coverMust(t, "not a zip", err, notZip)
	_, err = unpackMember(archive, &packed{name: "other.tif", sum: member.sum, size: 3}, filepath.Join(dir, "out"))
	coverMust(t, "no such member", err, `holds no member "other.tif"`)
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = unpackMember(archive, member, filepath.Join(blocker, "inside", "out"))
	coverMust(t, "a folder under a file", err, "not a directory")
	taken := filepath.Join(dir, "taken")
	if err := os.Mkdir(taken+partEnding, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = unpackMember(archive, member, taken)
	coverMust(t, "a part file that is a directory", err, "is a directory")
	sum, err := unpackMember(archive, member, filepath.Join(dir, "good"))
	if err != nil || sum != member.sum {
		t.Errorf("a good member: %q, %v", sum, err)
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "good")); string(raw) != "tif" {
		t.Errorf("the member holds %q", raw)
	}
}

// TestUnpackMemberRefusesAnAlgorithmItCannotRead checks a member stored with
// a compression method the reader does not have.
func TestUnpackMemberRefusesAnAlgorithmItCannotRead(t *testing.T) {
	var held bytes.Buffer
	writer := zip.NewWriter(&held)
	entry, err := writer.CreateRaw(&zip.FileHeader{Name: "m.tif", Method: 99, CompressedSize64: 3, UncompressedSize64: 3})
	if err != nil {
		t.Fatal(err)
	}
	entry.Write([]byte("abc"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.zip")
	if err := os.WriteFile(archive, held.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = unpackMember(archive, &packed{name: "m.tif", sum: coverSum("abc"), size: 3}, filepath.Join(dir, "out"))
	coverMust(t, "unsupported algorithm", err, "unsupported compression algorithm")
	if _, statErr := os.Stat(filepath.Join(dir, "out")); statErr == nil {
		t.Error("a member was written from an archive that could not be read")
	}
}

// TestShelfUnpacksOnlyFromAnArchiveItHolds checks that unpacking a source's
// member without its archive on the shelf is an error that names the source.
func TestShelfUnpacksOnlyFromAnArchiveItHolds(t *testing.T) {
	shelf := NewShelf(t.TempDir(), nil)
	list, err := coverList(t, "{member: m.tif, sha256: "+coverSum("tif")+", bytes: 3}")
	if err != nil {
		t.Fatal(err)
	}
	source, _ := list.Find("dem")
	_, err = shelf.unpackedFor(source)
	coverMust(t, "no archive", err, "dem: ")
}

// TestReadsRefusesAFileThatIsNotAsPinned checks that reading a plain
// dataset whose file is on the shelf and wrong is the check's error.
func TestReadsRefusesAFileThatIsNotAsPinned(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	dir := t.TempDir()
	shelf := NewShelf(dir, nil)
	source := Source{row: row{Key: "paper", From: server.URL + "/paper.pdf", Licence: "pd", Sha256: coverSum("right")}}
	if err := os.WriteFile(shelf.Path(source), []byte("wrong"), 0o644); err != nil {
		t.Fatal(err)
	}
	at, err := shelf.Reads(source, "", "")
	coverMust(t, "a wrong file", err, "the list pins "+coverSum("right"))
	if at != shelf.Path(source) {
		t.Errorf("path %q", at)
	}
}
