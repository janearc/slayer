package door

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// library is a tune library folder of the small tune, copied where the
// test keeps its files, with more tunes written as name, text pairs.
func library(t *testing.T, more ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"small.json", "small.cuts.json"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "tunes", name))
		if err != nil {
			t.Fatal(err)
		}
		written(t, dir, name, string(raw))
	}
	for i := nothing; i+one < len(more); i += pair {
		written(t, dir, more[i], more[i+one])
	}
	return dir
}

// tuning runs the tunes verb in a folder, as the working directory.
func tuning(t *testing.T, dir string, args ...string) ran {
	t.Helper()
	env := envIn(t, context.Background(), dir)
	return running(Tool(env), append([]string{"tunes"}, args...)...)
}

// a tune that will not load is refused by every switch
// that writes, and nothing is written for it.
// anti-test of TestTunesWritesStillsThePageAndFilms
func TestTunesRefusesATuneThatWillNotLoad(t *testing.T) {
	for _, flag := range []string{"--plates", "--page", "--films"} {
		dir := library(t)
		got := tuning(t, dir, dir, flag, "absent")
		if got.code != one || !strings.Contains(got.wrongSaid,
			"absent.json") {
			t.Errorf("%s absent gave %d: %q", flag, got.code,
				got.wrongSaid)
		}
		if len(files(t, dir)) != pair {
			t.Errorf("%s absent wrote %v", flag, files(t, dir))
		}
	}
}

// what tunes cannot do is refused, by name: no folder, two switches, an
// address with a switch, names with serving, an address off this
// machine, a folder not there or with no tunes.
// anti-test of TestTunesServesUntilStopped
// anti-test of TestTunesWritesStillsAndSoundsWhereOutSays
func TestTunesRefusesWhatItCannotDo(t *testing.T) {
	dir := library(t)
	empty := t.TempDir()
	for _, refusal := range []struct {
		args []string
		said string
	}{
		{nil, "tunes takes a library folder"},
		{[]string{dir, "--films", "--page"}, "tunes does one thing at a " +
			"time"},
		{[]string{dir, "--page", "--address", "127.0.0.1:1"},
			"--address is where the page is served; it does nothing " +
				"with --page"},
		{[]string{dir, "small"}, "serving takes the folder alone"},
		{[]string{dir, "--address", "0.0.0.0:8417"}, "the votes are " +
			"served on this machine only"},
		{[]string{dir, "--address", "nowhere"}, "is not host:port"},
		{[]string{filepath.Join(dir, "gone")}, "cannot be read"},
		{[]string{filepath.Join(dir, "gone"), "--check"}, "cannot be read"},
		{[]string{empty}, "holds no tunes"},
		{[]string{empty, "--page"}, "holds no tunes"},
		{[]string{empty, "--dry-run"}, "holds no tunes"},
		{[]string{dir, "--films", "--out", empty}, "--out says where " +
			"--plates or --sounds write"},
		{[]string{dir, "--out", empty}, "--out says where"},
		{[]string{"--sounds"}, "--sounds takes the voicings to draw"},
		{[]string{"--sounds", "@crypt-dmg"}, "the sheet draws studio " +
			"voicings"},
	} {
		got := tuning(t, dir, refusal.args...)
		if got.code != one || !strings.Contains(got.wrongSaid,
			refusal.said) {
			t.Errorf("tunes %v gave %d: %q, wanted %q", refusal.args,
				got.code, got.wrongSaid, refusal.said)
		}
	}
	if !slices.Equal(files(t, dir), []string{"small.cuts.json",
		"small.json"}) {
		t.Errorf("a refusal wrote %v", files(t, dir))
	}
}

// serving says the address to open, answers there with the votes, and
// stops cleanly when the context is cancelled; a dry run says where it
// would serve and listens on nothing.
func TestTunesServesUntilStopped(t *testing.T) {
	dir := library(t)
	dry := tuning(t, dir, dir, "--dry-run")
	if dry.code != nothing || !strings.HasPrefix(dry.said,
		"would serve "+dir+" on http://"+tunesAddress+"/") {
		t.Errorf("a dry run said %q %q", dry.said, dry.wrongSaid)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reading, writing := io.Pipe()
	done := make(chan ran, one)
	go func() {
		var wrong strings.Builder
		code := Tool(envIn(t, ctx, dir)).RunWith([]string{"tunes", dir,
			"--address", "127.0.0.1:0"}, strings.NewReader(""), writing,
			&wrong)
		writing.Close()
		done <- ran{code: code, wrongSaid: wrong.String()}
	}()
	line, err := bufio.NewReader(reading).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var address string
	if _, err := fmt.Sscanf(line, "open http://%s", &address); err != nil {
		t.Fatalf("serving said %q: %v", line, err)
	}
	answer, err := http.Get("http://" + address + "votes")
	if err != nil || answer.StatusCode != http.StatusOK {
		t.Errorf("the votes answered %v, %v", answer, err)
	} else {
		answer.Body.Close()
	}
	cancel()
	if stopped := <-done; stopped.code != nothing {
		t.Errorf("a stopped server gave %d: %s", stopped.code,
			stopped.wrongSaid)
	}
}

// an address already taken is refused, and nothing is
// served.
// anti-test of TestTunesServesUntilStopped
func TestTunesRefusesAnAddressTaken(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()
	dir := library(t)
	got := tuning(t, dir, dir, "--address", taken.Addr().String())
	if got.code != one || !strings.Contains(got.wrongSaid,
		"address already in use") {
		t.Errorf("a taken address gave %d: %q", got.code, got.wrongSaid)
	}
}

// tunes' example: the page a library would get, drawn dry.
func Example_tunes() {
	here, _ := os.Getwd()
	env, _ := NewEnv(Given{Context: context.Background(),
		Environment: func(string) (string, bool) { return "", false },
		Now:         noon, Directory: here})
	var said strings.Builder
	Tool(env).Run([]string{"tunes", "testdata/tunes", "--page",
		"--dry-run"}, &said, &said)
	fmt.Print(strings.ReplaceAll(said.String(), here, "."))
	// Output: would write ./testdata/tunes/index.html
}

// a request still open when the context is cancelled, as a film being
// streamed is, does not turn a clean stop into a failure: after the
// wait it is closed and listen gives nothing.
func TestTunesStopsCleanlyWithARequestOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	streaming := make(chan struct{})
	held := http.HandlerFunc(func(answer http.ResponseWriter,
		_ *http.Request) {
		answer.WriteHeader(http.StatusOK)
		answer.(http.Flusher).Flush()
		close(streaming)
		<-t.Context().Done()
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	reading, writing := io.Pipe()
	done := make(chan error, one)
	go func() {
		done <- serveOn(ctx, writing, listener, listener.Addr().String(),
			held)
		writing.Close()
	}()
	line, err := bufio.NewReader(reading).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	go io.Copy(io.Discard, reading)
	var address string
	if _, err := fmt.Sscanf(line, "open http://%s", &address); err != nil {
		t.Fatalf("listen said %q: %v", line, err)
	}
	answer, err := http.Get("http://" + address + "film.mp4")
	if err != nil {
		t.Fatal(err)
	}
	defer answer.Body.Close()
	<-streaming
	cancel()
	if err := <-done; err != nil {
		t.Errorf("a stop with a request open gave %v", err)
	}
}

// served by the name localhost, the address said is localhost with the
// port bound, and the server's own Host rule takes a request to it.
func TestTunesServedAsLocalhostAnswersAtTheAddressItSays(t *testing.T) {
	dir := library(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reading, writing := io.Pipe()
	done := make(chan ran, one)
	go func() {
		var wrong strings.Builder
		code := Tool(envIn(t, ctx, dir)).RunWith([]string{"tunes", dir,
			"--address", "localhost:0"}, strings.NewReader(""), writing,
			&wrong)
		writing.Close()
		done <- ran{code: code, wrongSaid: wrong.String()}
	}()
	line, err := bufio.NewReader(reading).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var address string
	if _, err := fmt.Sscanf(line, "open http://%s", &address); err != nil ||
		!strings.HasPrefix(address, "localhost:") {
		t.Fatalf("serving said %q: %v", line, err)
	}
	answer, err := http.Get("http://" + address + "votes")
	if err != nil || answer.StatusCode != http.StatusOK {
		t.Errorf("the votes at %s answered %v, %v", address, answer, err)
	} else {
		answer.Body.Close()
	}
	cancel()
	if stopped := <-done; stopped.code != nothing {
		t.Errorf("a stopped server gave %d: %s", stopped.code,
			stopped.wrongSaid)
	}
}
