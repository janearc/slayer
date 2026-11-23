package audition

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// smallScore is four bars of four at 240 a minute: a second a bar.
const smallScore = `{"name":"small","tempo":240,"meter":4,"parts":[
 {"role":"lead","bars":["c5:4","e5:4","g5:4","c6:4"]},
 {"role":"pad","bars":["-:4","(c4 e4 g4):4","(c4 e4 g4):4","(c4 e4 g4):4"]},
 {"role":"bass","bars":["c3:4","c3:4","g2:4","c3:4"]},
 {"role":"perc","bars":["-:4","k - s -","k - s -","k:4"]}]}`

const smallSheet = `{"group":"tests","mood":"small and quiet",
 "feeling":"nothing much","intent":"a test","listen_for":"the end",
 "voicing":"@dmg-pad","tempo":240,"meter":4,"seconds":4,"parts":[
 {"name":"first","bar":1,"second":0,"change":"one voice","feels":"alone"},
 {"name":"second","bar":2,"second":1,"change":"all four","feels":"arrived"},
 {"name":"last","bar":4,"second":3,"change":"a stop","feels":"done"}]}`

// folder is a library folder holding the small tune under a name.
func folder(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		write(t, dir, name+".json", smallScore)
		write(t, dir, name+".cuts.json", smallSheet)
	}
	return dir
}

func write(t *testing.T, dir, name, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNamesListsTunesAndNotSheetsOrVotes(t *testing.T) {
	dir := folder(t, "beta", "alpha")
	write(t, dir, votesFile, "{}")
	write(t, dir, "index.html", "x")
	write(t, dir, "bare.json", smallScore)
	write(t, dir, "orphan.cuts.json", smallSheet)
	names, err := Names(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(names, " ") != "alpha beta" {
		t.Errorf("names are %v", names)
	}
	if _, err := Names(filepath.Join(dir, "none")); err == nil {
		t.Error("a missing folder was accepted")
	}
}

func TestNamesRefusesANameThatIsNotLowercaseWordsAndHyphens(t *testing.T) {
	for _, bad := range []string{"Loud", "two words", "a_b", "tune2", "-a",
		"a--b", "a-", "<b>", "a\"b", "a&b"} {
		dir := folder(t, bad)
		if _, err := Names(dir); err == nil ||
			!strings.Contains(err.Error(), bad+".json") {
			t.Errorf("%q gave %v, want it named", bad, err)
		}
		if _, err := Load(dir, bad); err == nil {
			t.Errorf("Load took the name %q", bad)
		}
	}
	for _, good := range []string{"a", "glass-hour", "one-two-three"} {
		if _, err := Names(folder(t, good)); err != nil {
			t.Errorf("%q: %v", good, err)
		}
	}
}

func TestLoadNamesAPairThatClips(t *testing.T) {
	dir := folder(t, "small")
	write(t, dir, "small.cuts.json",
		strings.Replace(smallSheet, "@dmg-pad", "@no-such", 1))
	if _, err := Load(dir, "small"); err == nil ||
		!strings.Contains(err.Error(), "@no-such") {
		t.Errorf("an unknown voicing gave %v, want it named", err)
	}
}

func TestClockWritesMinutesAndSeconds(t *testing.T) {
	for in, want := range map[float64]string{0: "0:00", 14: "0:14",
		59.6: "1:00", 75: "1:15"} {
		if got := clock(in); got != want {
			t.Errorf("clock(%v) is %s, want %s", in, got, want)
		}
	}
}

// serve is a server over a folder with the clock fixed.
func serve(t *testing.T, names ...string) (*Server, string) {
	t.Helper()
	dir := folder(t, names...)
	s, err := NewServer(dir, testPort)
	if err != nil {
		t.Fatal(err)
	}
	s.ledger.now = func() time.Time {
		return time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	}
	return s, dir
}

// testPort is the port the test servers say they listen on; testHost is
// the Host header a page on this machine sends to it.
const (
	testPort = "8080"
	testHost = "127.0.0.1:" + testPort
)

// send is one request through the handler, as the page sends it: the right
// Host, and for a body the right content type.
func send(s *Server, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = testHost
	if method == http.MethodPost {
		req.Header.Set("Content-Type", jsonType)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func post(s *Server, body string) *httptest.ResponseRecorder {
	return send(s, http.MethodPost, "/votes", body)
}

func record(t *testing.T, s *Server) voteRecord {
	t.Helper()
	rec := get(s, "/votes")
	var out voteRecord
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAVoteIsWrittenAndReadBack(t *testing.T) {
	s, dir := serve(t, "alpha", "beta")
	if got := record(t, s); len(got.Votes) != 0 || len(got.Queue) != 0 {
		t.Fatalf("a new folder has votes: %+v", got)
	}
	for _, body := range []string{`{"name":"beta","vote":"keep"}`,
		`{"name":"alpha","vote":"keep"}`} {
		if rec := post(s, body); rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", body, rec.Code, rec.Body)
		}
	}
	got := record(t, s)
	if strings.Join(got.Queue, " ") != "beta alpha" {
		t.Errorf("the queue is %v, want the order picked", got.Queue)
	}
	if got.Votes["alpha"] != (castVote{Vote: keep, At: "2026-11-12T15:00:00Z"}) {
		t.Errorf("alpha's vote is %+v", got.Votes["alpha"])
	}
	if _, err := os.Stat(filepath.Join(dir, votesFile)); err != nil {
		t.Errorf("no record on disk: %v", err)
	}
	// a second server over the same folder reads what the first wrote.
	again, err := NewServer(dir, testPort)
	if err != nil {
		t.Fatal(err)
	}
	if len(record(t, again).Queue) != 2 {
		t.Error("the record did not survive a new server")
	}
}

func TestTossAndClearTakeATuneOutOfTheQueue(t *testing.T) {
	s, _ := serve(t, "alpha", "beta")
	post(s, `{"name":"alpha","vote":"keep"}`)
	post(s, `{"name":"beta","vote":"keep"}`)
	post(s, `{"name":"alpha","vote":"toss"}`)
	got := record(t, s)
	if strings.Join(got.Queue, " ") != "beta" ||
		got.Votes["alpha"].Vote != toss {
		t.Errorf("after a toss: %+v", got)
	}
	post(s, `{"name":"alpha","vote":"clear"}`)
	if _, there := record(t, s).Votes["alpha"]; there {
		t.Error("a cleared vote is still there")
	}
}

func TestAVoteForAnUnknownTuneOrWordIsRefusedAndWritesNothing(t *testing.T) {
	s, dir := serve(t, "alpha")
	for _, body := range []string{`{"name":"nope","vote":"keep"}`,
		`{"name":"alpha","vote":"love"}`, `{"name":"../x","vote":"keep"}`,
		`{"name":"alpha","vote":"keep","extra":1}`, `not json`, ``,
		`{"name":"alpha","vote":"` + strings.Repeat("x", maxBody) + `"}`} {
		if rec := post(s, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%.40q gave %d, want 400", body, rec.Code)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, votesFile)); err == nil {
		t.Error("a refused vote wrote a record")
	}
}

func TestOnlyGetAndPostReachTheVotes(t *testing.T) {
	s, _ := serve(t, "alpha")
	rec := send(s, http.MethodDelete, "/votes", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE gave %d", rec.Code)
	}
}

func TestACorruptRecordIsReportedNotOverwritten(t *testing.T) {
	s, dir := serve(t, "alpha")
	write(t, dir, votesFile, "{not json")
	if rec := post(s, `{"name":"alpha","vote":"keep"}`); rec.Code == 200 {
		t.Error("a vote was written over a corrupt record")
	}
	b, _ := os.ReadFile(filepath.Join(dir, votesFile))
	if string(b) != "{not json" {
		t.Errorf("the corrupt record was changed to %q", b)
	}
}

func TestTheServerServesTheFolderAndNothingOutsideIt(t *testing.T) {
	s, dir := serve(t, "alpha")
	write(t, filepath.Dir(dir), "secret-"+filepath.Base(dir)+".txt", "no")
	secret := "secret-" + filepath.Base(dir) + ".txt"
	outside := filepath.Join(filepath.Dir(dir), secret)
	if err := os.Symlink(outside,
		filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/alpha.json", "/alpha.cuts.json"} {
		if code := get(s, path).Code; code != 200 {
			t.Errorf("%s gave %d, want 200", path, code)
		}
	}
	// a path that climbs out, an encoded one, a link that leads out and a
	// missing file are none of them served.
	for _, path := range []string{"/..%2f" + secret, "/../" + secret,
		"/link.txt", "/missing"} {
		rec := get(s, path)
		if rec.Code == 200 || rec.Body.String() == "no" {
			t.Errorf("%s gave %d: %q", path, rec.Code, rec.Body)
		}
	}
}

// get is one GET through the handler.
func get(s *Server, path string) *httptest.ResponseRecorder {
	return send(s, http.MethodGet, path, "")
}

func TestAFolderWithNoTunesOrNoFolderIsRefused(t *testing.T) {
	if _, err := NewServer(t.TempDir(), testPort); err == nil ||
		!strings.Contains(err.Error(), "no tunes") {
		t.Errorf("an empty folder gave %v", err)
	}
	missing := filepath.Join(t.TempDir(), "none")
	if _, err := NewServer(missing, testPort); err == nil {
		t.Error("a missing folder was accepted")
	}
}

func TestOnlyALoopbackAddressIsServed(t *testing.T) {
	for _, ok := range []string{"127.0.0.1:8080", "localhost:9"} {
		if err := Loopback(ok); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{":8080", "0.0.0.0:8080", "192.168.1.5:80",
		"example.com:80", "127.0.0.1", "[::]:80", "127.0.0.2:80", ""} {
		if err := Loopback(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
	if err := Serve(t.TempDir(), "0.0.0.0:0"); err == nil ||
		!strings.Contains(err.Error(), "refused") {
		t.Errorf("Serve on every interface gave %v", err)
	}
	if err := Serve(t.TempDir(), "127.0.0.1:0"); err == nil {
		t.Error("Serve of an empty folder did not stop")
	}
}

func TestATuneAddedWhileTheServerRunsCanBeVotedOn(t *testing.T) {
	s, dir := serve(t, "alpha")
	if rec := post(s, `{"name":"later","vote":"keep"}`); rec.Code != 400 {
		t.Fatalf("a tune not there yet gave %d, want 400", rec.Code)
	}
	write(t, dir, "later.json", smallScore)
	if rec := post(s, `{"name":"later","vote":"keep"}`); rec.Code != 200 {
		t.Errorf("a tune added to the folder gave %d: %s", rec.Code, rec.Body)
	}
	// a name is a file's base name: a path to a file is not a name, even
	// where the file is there.
	write(t, dir, "x.json", smallScore)
	for _, name := range []string{"../x", "./x", "x/../x", "x.json", ".",
		"..", "", "sub/x"} {
		body, _ := json.Marshal(ballot{Name: name, Vote: keep})
		if rec := post(s, string(body)); rec.Code != 400 {
			t.Errorf("name %q gave %d, want 400", name, rec.Code)
		}
	}
}

func TestAVoteForATuneGoneFromTheFolderCanBeCleared(t *testing.T) {
	s, dir := serve(t, "alpha", "beta")
	post(s, `{"name":"alpha","vote":"keep"}`)
	post(s, `{"name":"beta","vote":"toss"}`)
	for _, name := range []string{"alpha", "beta"} {
		os.Remove(filepath.Join(dir, name+".json"))
	}
	for _, name := range []string{"alpha", "beta"} {
		body := `{"name":"` + name + `","vote":"clear"}`
		if rec := post(s, body); rec.Code != 200 {
			t.Errorf("clearing %s gave %d: %s", name, rec.Code, rec.Body)
		}
	}
	if got := record(t, s); len(got.Votes) != 0 || len(got.Queue) != 0 {
		t.Errorf("the record still holds %+v", got)
	}
	// anti: a name that is not in the record, and not a tune, is refused;
	// and a gone tune cannot be newly voted on.
	for _, body := range []string{`{"name":"gamma","vote":"clear"}`,
		`{"name":"alpha","vote":"toss"}`} {
		if rec := post(s, body); rec.Code != 400 {
			t.Errorf("%s gave %d, want 400", body, rec.Code)
		}
	}
}

func TestAVoteNeedsJSONAndThisMachinesNameAndPort(t *testing.T) {
	s, dir := serve(t, "alpha")
	body := `{"name":"alpha","vote":"keep"}`
	try := func(host, kind string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/votes",
			strings.NewReader(body))
		req.Host = host
		if kind != "" {
			req.Header.Set("Content-Type", kind)
		}
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec
	}
	for _, kind := range []string{"", "text/plain", "text/plain;charset=UTF-8",
		"application/x-www-form-urlencoded", "multipart/form-data",
		"application/jsonx", "application/json;;"} {
		if got := try(testHost, kind).Code; got != http.StatusUnsupportedMediaType {
			t.Errorf("content type %q gave %d, want 415", kind, got)
		}
	}
	for _, host := range []string{"evil.example:8080", "evil.example",
		"127.0.0.1", "localhost", "127.0.0.1:9", "localhost:80",
		"127.0.0.1:8080.evil.example", "", "[::1]:8080", "0.0.0.0:8080"} {
		if got := try(host, jsonType).Code; got != http.StatusForbidden {
			t.Errorf("host %q gave %d, want 403", host, got)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, votesFile)); err == nil {
		t.Error("a refused request wrote a record")
	}
	for _, host := range []string{testHost, "localhost:" + testPort} {
		for _, kind := range []string{jsonType, "application/json; charset=utf-8",
			"Application/JSON"} {
			if got := try(host, kind).Code; got != 200 {
				t.Errorf("host %q type %q gave %d, want 200", host, kind, got)
			}
		}
	}
	// the files and the record are behind the same host check.
	req := httptest.NewRequest(http.MethodGet, "/alpha.json", nil)
	req.Host = "evil.example:8080"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a file under a hostile name gave %d", rec.Code)
	}
}

func TestAServerRefusesAPortThatIsNotAPort(t *testing.T) {
	dir := folder(t, "alpha")
	for _, bad := range []string{"", "http", "-1", "65536", "80 "} {
		if _, err := NewServer(dir, bad); err == nil {
			t.Errorf("port %q was accepted", bad)
		}
	}
	for _, ok := range []string{"0", "8080", "65535"} {
		if _, err := NewServer(dir, ok); err != nil {
			t.Errorf("port %q: %v", ok, err)
		}
	}
}

func TestServeAnswersToItsOwnPortAndNoOther(t *testing.T) {
	dir := folder(t, "alpha")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("no loopback here: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	go Serve(dir, addr)
	var resp *http.Response
	for range 50 {
		resp, err = http.Get("http://" + addr + "/votes")
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("Serve did not answer: %v %v", resp, err)
	}
	resp.Body.Close()
	_, port, _ := net.SplitHostPort(addr)
	req, _ := http.NewRequest(http.MethodGet, "http://"+addr+"/votes", nil)
	req.Host = "evil.example:" + port
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Errorf("a hostile Host through Serve gave %v %v", resp, err)
	}
	resp.Body.Close()
}

func TestServeOnAPortThatIsTakenStopsAndSaysSo(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("no loopback here: %v", err)
	}
	defer ln.Close()
	if err := Serve(folder(t, "alpha"), ln.Addr().String()); err == nil {
		t.Error("Serve on a taken port did not stop")
	}
}
