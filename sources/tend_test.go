package sources_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/janearc/slayer/sources"
)

// the bytes the fake server holds, written by hand.
const (
	moonNew   = "the moon, as the source holds it now"
	moonOld   = "the moon, as an older copy held it"
	demMember = "kil/kil.tif"
	demBytes  = "a small made-up elevation model"
)

// the numbers the tests count with.
const (
	// fileMode is the mode of a file a test writes; defined: the owner
	// writes, all others read.
	fileMode = 0o644
	// dirMode is the mode of a folder a test makes; defined: the owner
	// writes, all others read.
	dirMode = 0o755
	// none is zero; defined: no requests.
	none = 0
	// once is one; defined: one request.
	once = 1
	// halves is two; defined: the cut a server makes, at half.
	halves = 2
	// unsized is a length a server does not give; defined: below zero.
	unsized = -1
)

// sum is the sha256 of text in hex, from the standard library.
func sum(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:])
}

// zipped is a zip holding one member of the given bytes.
func zipped(t *testing.T, member, text string) []byte {
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

// fake is a server on localhost: the bytes it holds by path, how many
// requests came by method, a size it may claim in place of the true
// one, and whether it cuts a download off halfway.
type fake struct {
	*httptest.Server
	mu      sync.Mutex
	held    map[string][]byte
	claim   map[string]int
	cut     bool
	methods map[string]int
}

// serving is a fake holding the bytes by path.
func serving(t *testing.T, held map[string][]byte) *fake {
	t.Helper()
	server := &fake{held: held, claim: map[string]int{},
		methods: map[string]int{}}
	server.Server = httptest.NewServer(server)
	t.Cleanup(server.Close)
	return server
}

// ServeHTTP answers a GET with the bytes, a HEAD with their length, and
// anything else held nowhere with a 404.
func (server *fake) ServeHTTP(answer http.ResponseWriter,
	request *http.Request) {
	server.mu.Lock()
	server.methods[request.Method]++
	body, found := server.held[request.URL.Path]
	size := len(body)
	if claimed, lies := server.claim[request.URL.Path]; lies {
		size = claimed
	}
	cut := server.cut
	server.mu.Unlock()
	if !found {
		http.NotFound(answer, request)
		return
	}
	if size >= none {
		answer.Header().Set("Content-Length", strconv.Itoa(size))
	}
	if request.Method == http.MethodHead {
		return
	}
	if cut {
		answer.Write(body[:len(body)/halves])
		answer.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	}
	answer.Write(body)
}

// asked is how many requests of a method the server saw.
func (server *fake) asked(method string) int {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.methods[method]
}

// shelfWith is a temporary shelf, and a list of a moon face and a dem
// zip both at the server, pinned to the bytes the test means them to be.
func shelfWith(t *testing.T, server *fake) (sources.Shelf, string,
	sources.List) {
	t.Helper()
	dir := t.TempDir()
	zip := server.held["/dem.zip"]
	text := "sources:\n" +
		"  - key: moon-face\n    from: " + server.URL + "/moon.tif\n" +
		"    licence: a test's own\n    sha256: " + sum(moonNew) + "\n" +
		"    bytes: " + strconv.Itoa(len(moonNew)) + "\n" +
		"  - key: dem\n    from: " + server.URL + "/dem.zip\n" +
		"    licence: a test's own\n    sha256: " + sum(string(zip)) + "\n" +
		"    bytes: " + strconv.Itoa(len(zip)) + "\n" +
		"    unpack:\n      member: " + demMember + "\n" +
		"      sha256: " + sum(demBytes) + "\n" +
		"      bytes: " + strconv.Itoa(len(demBytes)) + "\n"
	listed := filepath.Join(t.TempDir(), "sources.yaml")
	if err := os.WriteFile(listed, []byte(text), fileMode); err != nil {
		t.Fatal(err)
	}
	list, err := sources.Read(listed)
	if err != nil {
		t.Fatal(err)
	}
	return sources.NewShelf(dir, server.Client()), dir, list
}

// served is the usual fake: the moon as it is now and the dem zip.
func served(t *testing.T) *fake {
	t.Helper()
	return serving(t, map[string][]byte{"/moon.tif": []byte(moonNew),
		"/dem.zip": zipped(t, demMember, demBytes)})
}

// tend runs one mode over the wants and gives what was said, and the error.
func tend(t *testing.T, shelf sources.Shelf, list sources.List,
	server *fake, mode sources.Mode, wants ...sources.Want) (string, error) {
	t.Helper()
	var said bytes.Buffer
	origin := sources.NewListed(list, server.Client())
	err := shelf.Tend(context.Background(), origin, mode, wants, &said)
	return said.String(), err
}

// each of the words a line must say, or the test fails with the line.
func mustSay(t *testing.T, said string, words ...string) {
	t.Helper()
	for _, word := range words {
		if !strings.Contains(said, word) {
			t.Errorf("the output lacks %q:\n%s", word, said)
		}
	}
}

// a walk is the files under a directory that are not finished ones.
type walk struct{ found []string }

// visit keeps a path that is a .part, or in a folder a fetch made for
// itself.
func (found *walk) visit(at string, _ os.DirEntry, _ error) error {
	if strings.HasSuffix(at, ".part") || strings.Contains(at, ".fetching") {
		found.found = append(found.found, at)
	}
	return nil
}

// leftovers is every file under a directory that is not a finished one:
// a .part, or a folder a fetch made for itself.
func leftovers(t *testing.T, dir string) []string {
	t.Helper()
	var walked walk
	if err := filepath.WalkDir(dir, walked.visit); err != nil {
		t.Fatal(err)
	}
	return walked.found
}

// a dataset missing from the shelf is fetched, verified and put where it
// is read, and the output says what, from where, how many bytes and the sum.
func TestFetchBringsAMissingDatasetAndSaysWhat(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	said, err := tend(t, shelf, list, server, sources.Fetching,
		sources.NewWant("moon-face", ""))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "moon-face.tif"))
	if err != nil || string(got) != moonNew {
		t.Fatalf("the shelf holds %q, %v", got, err)
	}
	size := strconv.Itoa(len(moonNew)) + " bytes"
	mustSay(t, said, "moon-face", "fetched", server.URL+"/moon.tif", size,
		sum(moonNew))
	if found := leftovers(t, dir); len(found) != none {
		t.Errorf("a fetch left %v", found)
	}
}

// a dataset on the shelf and correct is not fetched, and the output says so.
func TestFetchLeavesAPresentDatasetAndSaysSo(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	if err := os.WriteFile(filepath.Join(dir, "moon-face.tif"),
		[]byte(moonNew), fileMode); err != nil {
		t.Fatal(err)
	}
	said, err := tend(t, shelf, list, server, sources.Fetching,
		sources.NewWant("moon-face", ""))
	if err != nil {
		t.Fatal(err)
	}
	mustSay(t, said, "moon-face", "not fetched", sum(moonNew))
	if got := server.asked("GET") + server.asked("HEAD"); got != none {
		t.Errorf("a present dataset made %d requests", got)
	}
}

// a download that is not what the list pins is refused, and nothing of
// it stays on the shelf.
func TestFetchRefusesACorruptDownloadAndRemovesIt(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	server.held["/moon.tif"] = []byte(moonNew[:len(moonNew)-once] + "?")
	_, err := tend(t, shelf, list, server, sources.Fetching,
		sources.NewWant("moon-face", ""))
	if err == nil || !strings.Contains(err.Error(), "moon-face") {
		t.Fatalf("a corrupt download was taken: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		t.Errorf("the shelf holds %s after a refused download", entry.Name())
	}
}

// a refresh fetches again even when present, and replaces the old file
// only with a good one; the output gives the old and the new sums.
func TestRefreshReplacesTheOldFileAndSaysWhatChanged(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	at := filepath.Join(dir, "moon-face.tif")
	if err := os.WriteFile(at, []byte(moonOld), fileMode); err != nil {
		t.Fatal(err)
	}
	said, err := tend(t, shelf, list, server, sources.Refreshing,
		sources.NewWant("moon-face", ""))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(at)
	if string(got) != moonNew {
		t.Fatalf("the shelf holds %q", got)
	}
	mustSay(t, said, "changed", sum(moonOld), sum(moonNew))
	if found := leftovers(t, dir); len(found) != none {
		t.Errorf("a refresh left %v", found)
	}
}

// a refresh of a file that is already right says the bytes are the same,
// and still asked the server.
func TestRefreshSaysWhenTheBytesAreTheSame(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	if err := os.WriteFile(filepath.Join(dir, "moon-face.tif"),
		[]byte(moonNew), fileMode); err != nil {
		t.Fatal(err)
	}
	said, err := tend(t, shelf, list, server, sources.Refreshing,
		sources.NewWant("moon-face", ""))
	if err != nil {
		t.Fatal(err)
	}
	mustSay(t, said, "same bytes", sum(moonNew))
	if server.asked("GET") != once {
		t.Errorf("a refresh made %d downloads, want 1", server.asked("GET"))
	}
}

// a fetch that dies halfway leaves the old file on the shelf, still good.
func TestRefreshKeepsTheOldFileWhenTheFetchDiesHalfway(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	at := filepath.Join(dir, "moon-face.tif")
	if err := os.WriteFile(at, []byte(moonOld), fileMode); err != nil {
		t.Fatal(err)
	}
	server.cut = true
	_, err := tend(t, shelf, list, server, sources.Refreshing,
		sources.NewWant("moon-face", ""))
	if err == nil {
		t.Fatal("a download cut off halfway was taken")
	}
	got, readErr := os.ReadFile(at)
	if readErr != nil || string(got) != moonOld {
		t.Fatalf("the old file is %q, %v; want it kept", got, readErr)
	}
	if found := leftovers(t, dir); len(found) != none {
		t.Errorf("a cut fetch left %v", found)
	}
}

// check downloads nothing, and says what it compared.
func TestCheckSaysSameAndNamesWhatWasCompared(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	if err := os.WriteFile(filepath.Join(dir, "moon-face.tif"),
		[]byte(moonNew), fileMode); err != nil {
		t.Fatal(err)
	}
	said, err := tend(t, shelf, list, server, sources.Checking,
		sources.NewWant("moon-face", ""))
	if err != nil {
		t.Fatalf("%v\n%s", err, said)
	}
	mustSay(t, said, "moon-face", "same", "sha256", "sources.yaml",
		"size", "the server", server.URL+"/moon.tif")
	if server.asked("GET") != none || server.asked("HEAD") != once {
		t.Errorf("check made %d GET and %d HEAD, want 0 and 1",
			server.asked("GET"), server.asked("HEAD"))
	}
}

// a difference is one way a shelf can differ from its source: what the
// shelf holds, a size the server claims in place of the true one, whether
// the server is gone, and what check must say.
type difference struct {
	name, shelf string
	claim       int
	down        bool
	want        []string
}

// run is check of a moon face that differs so, which must fail, say so,
// and not say the same.
func (fault difference) run(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	if fault.shelf != "" {
		at := filepath.Join(dir, "moon-face.tif")
		if err := os.WriteFile(at, []byte(fault.shelf), fileMode); err != nil {
			t.Fatal(err)
		}
	}
	if fault.claim != none {
		server.claim["/moon.tif"] = fault.claim
	}
	if fault.down {
		server.Close()
	}
	said, err := tend(t, shelf, list, server, sources.Checking,
		sources.NewWant("moon-face", ""))
	if err == nil {
		t.Fatalf("check passed:\n%s", said)
	}
	mustSay(t, said+err.Error(), fault.want...)
	if strings.Contains(said+err.Error(), "same") {
		t.Errorf("check overclaimed:\n%s", said)
	}
}

// check says plainly each way a file can differ, and fails.
func TestCheckSaysEachDifference(t *testing.T) {
	size := strconv.Itoa(len(moonNew) + len(moonOld))
	for _, fault := range []difference{
		{name: "checksum", shelf: moonOld, want: []string{"sha256",
			"differs", sum(moonOld), sum(moonNew)}},
		{name: "size", shelf: moonNew, claim: len(moonNew) + len(moonOld),
			want: []string{"size", "differs", size}},
		{name: "missing", want: []string{"missing", "moon-face"}},
		{name: "unreachable", shelf: moonNew, down: true,
			want: []string{"could not reach"}},
	} {
		t.Run(fault.name, fault.run)
	}
}

// check says only what it could compare: a server that gives no size
// is not said to agree on size.
func TestCheckDoesNotClaimASizeTheServerDidNotGive(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	os.WriteFile(filepath.Join(dir, "moon-face.tif"), []byte(moonNew), fileMode)
	server.claim["/moon.tif"] = unsized
	said, err := tend(t, shelf, list, server, sources.Checking,
		sources.NewWant("moon-face", ""))
	if err != nil {
		t.Fatalf("%v\n%s", err, said)
	}
	mustSay(t, said, "the server gave no size", "not compared")
	if strings.Contains(said, "size matches") {
		t.Errorf("claimed a size the server did not give:\n%s", said)
	}
}

// a dataset in a zip is fetched, its one member unpacked to unpacked/ and
// its path, and the member checked against its own pin.
func TestFetchUnpacksTheNamedMember(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	said, err := tend(t, shelf, list, server, sources.Fetching,
		sources.NewWant("dem", "unpacked/"+demMember))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "unpacked", demMember))
	if err != nil || string(got) != demBytes {
		t.Fatalf("the member is %q, %v", got, err)
	}
	mustSay(t, said, "dem", "fetched", "unpacked", demMember, sum(demBytes))
	again, err := tend(t, shelf, list, server, sources.Fetching,
		sources.NewWant("dem", "unpacked/"+demMember))
	if err != nil {
		t.Fatal(err)
	}
	mustSay(t, again, "not fetched")
	if server.asked("GET") != once {
		t.Errorf("the dem was downloaded %d times", server.asked("GET"))
	}
}

// a member whose bytes are not its pin is refused and removed.
func TestFetchRefusesAMemberThatIsNotItsPin(t *testing.T) {
	server := served(t)
	server.held["/dem.zip"] = zipped(t, demMember, "another model entirely")
	shelf, dir, list := shelfWith(t, server)
	_, err := tend(t, shelf, list, server, sources.Fetching,
		sources.NewWant("dem", "unpacked/"+demMember))
	if err == nil {
		t.Fatal("a member that is not its pin was taken")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "unpacked", demMember)); statErr == nil {
		t.Error("the refused member is on the shelf")
	}
	if found := leftovers(t, dir); len(found) != none {
		t.Errorf("a refused member left %v", found)
	}
}

// a member whose name climbs out of its folder is refused, and nothing
// is written outside the shelf.
func TestUnpackRefusesAMemberThatEscapes(t *testing.T) {
	server := served(t)
	_, _, _ = shelfWith(t, server)
	text := "sources:\n  - key: evil\n    from: " + server.URL + "/dem.zip\n" +
		"    licence: a test's own\n    unpack:\n      member: ../evil.tif\n" +
		"      sha256: " + sum(demBytes) + "\n      bytes: 5\n"
	listed := filepath.Join(t.TempDir(), "sources.yaml")
	os.WriteFile(listed, []byte(text), fileMode)
	if _, err := sources.Read(listed); err == nil ||
		!strings.Contains(err.Error(), "../evil.tif") {
		t.Fatalf("the list took an escaping member: %v", err)
	}
}

// reading what is not on the shelf downloads nothing and fails with one
// error that names the dataset, the path and the flag.
func TestReadsOfAnAbsentDatasetNamesDatasetPathAndFlag(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	source, _ := list.Find("moon-face")
	_, err := shelf.Reads(source, "", "--fetch-terrain")
	if err == nil {
		t.Fatal("an absent dataset was read")
	}
	mustSay(t, err.Error(), "moon-face", filepath.Join(dir, "moon-face.tif"),
		"--fetch-terrain")
	if server.asked("GET")+server.asked("HEAD") != none {
		t.Error("reading an absent dataset made a request")
	}
}

// a member on the shelf is read only when its bytes are its pin.
func TestReadsOfAMemberHoldItToItsPin(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	at := filepath.Join(dir, "unpacked", demMember)
	os.MkdirAll(filepath.Dir(at), dirMode)
	os.WriteFile(at, []byte(demBytes), fileMode)
	source, _ := list.Find("dem")
	got, err := shelf.Reads(source, "unpacked/"+demMember, "--fetch-terrain")
	if err != nil || got != at {
		t.Fatalf("a good member: %q, %v", got, err)
	}
	os.WriteFile(at, []byte("not the model"), fileMode)
	_, err = shelf.Reads(source, "unpacked/"+demMember, "--fetch-terrain")
	if err == nil || !strings.Contains(err.Error(), sum("not the model")) {
		t.Fatalf("a bent member was read: %v", err)
	}
}

// tinyHandler sends nine bytes of moon to every request.
func tinyHandler(answer http.ResponseWriter, _ *http.Request) {
	answer.Write([]byte("tiny moon"))
}

// the example of what a person sees from one fetch.
func ExampleShelf_Tend() {
	server := httptest.NewServer(http.HandlerFunc(tinyHandler))
	defer server.Close()
	dir, _ := os.MkdirTemp("", "shelf")
	defer os.RemoveAll(dir)
	listed := filepath.Join(dir, "sources.yaml")
	digest := sha256.Sum256([]byte("tiny moon"))
	pin := hex.EncodeToString(digest[:])
	text := "sources:\n  - key: moon\n    from: " + server.URL +
		"/m.tif\n    licence: x\n    sha256: " + pin + "\n"
	os.WriteFile(listed, []byte(text), fileMode)
	list, _ := sources.Read(listed)
	shelf := sources.NewShelf(filepath.Join(dir, "shelf"), server.Client())
	origin := sources.NewListed(list, server.Client())
	var said strings.Builder
	shelf.Tend(context.Background(), origin, sources.Fetching,
		[]sources.Want{sources.NewWant("moon", "")}, &said)
	told := said.String()
	fmt.Println(strings.Contains(told, "fetched 9 bytes"))
	// Output: true
}

// the sources command's fetch of an archive with a pinned member brings
// the archive and unpacks the member, so a fresh shelf is enough; a
// second fetch downloads nothing and leaves the member.
func TestShelfFetchUnpacksThePinnedMember(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	source, _ := list.Find("dem")
	said, err := shelf.Fetch(context.Background(), source, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "unpacked", demMember))
	if err != nil || string(got) != demBytes {
		t.Fatalf("the member is %q, %v", got, err)
	}
	mustSay(t, said, "unpacked", demMember)
	if _, err := shelf.Fetch(context.Background(), source, time.Now()); err != nil {
		t.Fatal(err)
	}
	if server.asked("GET") != once {
		t.Errorf("the second fetch downloaded again: %d GETs", server.asked("GET"))
	}
}

// the dem's member of the real list is pinned: a file in its place that is
// not the pin is refused with the pin said, as measured once by `shasum -a
// 256` of the unpacked file.
func TestTheRealListPinsTheKilaueaMember(t *testing.T) {
	const pin = "696a6ef3084bd871df8e51464c3b03084a47ad8625c5a32c094d613a20a959c4"
	const member = "unpacked/kil_2019dem/kil_2019dem.tif"
	list, err := sources.Read("../sources.yaml")
	if err != nil {
		t.Fatal(err)
	}
	source, found := list.Find("usgs-kilauea-2019-dem")
	if !found {
		t.Fatal("the list lacks the dem")
	}
	dir := t.TempDir()
	at := filepath.Join(dir, filepath.FromSlash(member))
	if err := os.MkdirAll(filepath.Dir(at), dirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte("not the dem"), fileMode); err != nil {
		t.Fatal(err)
	}
	_, err = sources.NewShelf(dir, nil).Reads(source, member, "--fetch-terrain")
	if err == nil || !strings.Contains(err.Error(), pin) {
		t.Fatalf("a bent member was read, or the pin was not said: %v", err)
	}
}

// a shelf hashed by a function asks it, and holds its answer to the pin.
func TestReadsOfAMemberAskTheShelfsHash(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	at := filepath.Join(dir, "unpacked", demMember)
	os.MkdirAll(filepath.Dir(at), dirMode)
	os.WriteFile(at, []byte("not the model"), fileMode)
	source, _ := list.Find("dem")
	var asked []string
	kept := shelf.HashedBy(func(path string) (string, error) {
		asked = append(asked, path)
		return sum(demBytes), nil
	})
	got, err := kept.Reads(source, "unpacked/"+demMember, "--fetch-terrain")
	if err != nil || got != at || len(asked) != 1 || asked[0] != at {
		t.Fatalf("read %q, %v; asked %v", got, err, asked)
	}
}

// a shelf's hash that answers otherwise than the pin is refused, and one
// that fails says so.
func TestReadsOfAMemberRefuseTheShelfsHashWhereItIsNotThePin(t *testing.T) {
	server := served(t)
	shelf, dir, list := shelfWith(t, server)
	at := filepath.Join(dir, "unpacked", demMember)
	os.MkdirAll(filepath.Dir(at), dirMode)
	os.WriteFile(at, []byte(demBytes), fileMode)
	source, _ := list.Find("dem")
	for _, hash := range []func(string) (string, error){
		func(string) (string, error) { return sum("other"), nil },
		func(string) (string, error) { return "", os.ErrPermission },
	} {
		_, err := shelf.HashedBy(hash).Reads(source,
			"unpacked/"+demMember, "--fetch-terrain")
		if err == nil {
			t.Error("a hash that is not the pin was taken")
		}
	}
}
