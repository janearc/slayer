//go:build draw

package audition

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/interfaces/movie"
)

func TestLoadRefusesASheetThatDisagreesAndAMissingTune(t *testing.T) {
	dir := folder(t, "small")
	if _, err := Load(dir, "small"); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, "absent"); err == nil {
		t.Error("a tune with no file was accepted")
	}
	write(t, dir, "small.cuts.json",
		strings.Replace(smallSheet, `"seconds":4`, `"seconds":9`, 1))
	if _, err := Load(dir, "small"); err == nil {
		t.Error("a sheet of the wrong length was accepted")
	}
	write(t, dir, "small.cuts.json", smallSheet)
	write(t, dir, "small.json", "{")
	if _, err := Load(dir, "small"); err == nil {
		t.Error("a broken score was accepted")
	}
}

func TestThePlateHasTheSizeAndShowsTheParts(t *testing.T) {
	tune, err := Load(folder(t, "small"), "small")
	if err != nil {
		t.Fatal(err)
	}
	img, err := Plate(tune)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != wide || img.Bounds().Dy() != tall {
		t.Errorf("plate is %v", img.Bounds())
	}
	x := tune.xAt(1)
	if got := img.RGBAAt(x, stripTop+5); got != marker {
		t.Errorf("no marker where part two starts: %v", got)
	}
	if got := img.RGBAAt(wide-1, tall-1); got != ground {
		t.Errorf("the corner is %v, not the ground", got)
	}
	frame := tune.frame(img, mustWriter(t), 1.5)
	if frame.RGBAAt(tune.xAt(1.5), rollTop+1) != cursor {
		t.Error("the line is not where the second is")
	}
}

func mustWriter(t *testing.T) *writer {
	t.Helper()
	w, err := newWriter()
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestTheFilmIsAMovieOfTheRightSizeWithSound(t *testing.T) {
	tune, err := Load(folder(t, "small"), "small")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "small.mp4")
	if err := Film(tune, path); err != nil {
		t.Skipf("no movie writer here: %v", err)
	}
	facts, err := movie.Probe(path)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Width() != wide || facts.Height() != tall ||
		facts.Rate() != rate || facts.AudioChannels() != 2 {
		t.Errorf("the film is %dx%d at %v with %d channels", facts.Width(),
			facts.Height(), facts.Rate(), facts.AudioChannels())
	}
}

func TestThePageHoldsEveryTuneItsPartsAndBothButtons(t *testing.T) {
	dir := folder(t, "small", "other")
	var tunes []*Tune
	for _, name := range []string{"small", "other"} {
		tune, err := Load(dir, name)
		if err != nil {
			t.Fatal(err)
		}
		tunes = append(tunes, tune)
	}
	b, err := Page(tunes)
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	for _, want := range []string{`id="tune-small"`, `id="tune-other"`,
		"small and quiet", "nothing much", "a test", "the end",
		`data-at="1.00"`, "0:01 second", "arrived", "Keep", "Toss",
		"tests</h2>", "small.mp4"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	for _, bad := range []string{"http://", "https://", "<script src"} {
		if strings.Contains(page, bad) {
			t.Errorf("the page reaches for the network: %q", bad)
		}
	}
	if strings.Count(page, `<article`) != 2 {
		t.Error("the page does not hold one row a tune")
	}
}

func TestAHostileTuneNameCannotBreakThePage(t *testing.T) {
	dir := folder(t, "small")
	write(t, dir, "small.cuts.json", strings.Replace(smallSheet,
		"nothing much", `<script>alert(1)</script>`, 1))
	tune, err := Load(dir, "small")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Page([]*Tune{tune})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "<script>alert") {
		t.Error("sheet text reached the page unescaped")
	}
}

func TestGroupsFollowTheirOrderAndUnknownOnesComeLast(t *testing.T) {
	dir := folder(t, "aaa", "bbb", "ccc")
	for name, group := range map[string]string{"aaa": "zebra",
		"bbb": "playful", "ccc": "sombre and slow"} {
		write(t, dir, name+".cuts.json", strings.Replace(smallSheet,
			`"group":"tests"`, `"group":"`+group+`"`, 1))
	}
	var tunes []*Tune
	for _, name := range []string{"aaa", "bbb", "ccc"} {
		tune, err := Load(dir, name)
		if err != nil {
			t.Fatal(err)
		}
		tunes = append(tunes, tune)
	}
	var got []string
	for _, g := range groups(tunes) {
		got = append(got, g.Name)
	}
	if strings.Join(got, "|") != "sombre and slow|playful|zebra" {
		t.Errorf("the groups are %v", got)
	}
}

func TestAPageServedOverRealHTTPTakesAVoiceAndGivesItBack(t *testing.T) {
	s, dir := serve(t, "alpha")
	tune, err := Load(dir, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	page, err := Page([]*Tune{tune})
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, "index.html", string(page))
	srv := httptest.NewUnstartedServer(nil)
	_, port, _ := strings.Cut(srv.Listener.Addr().String(), ":")
	s.hosts = hostsOf(port)
	srv.Config.Handler = s.Handler()
	srv.Start()
	defer srv.Close()
	got, err := http.Get(srv.URL + "/")
	if err != nil || got.StatusCode != 200 {
		t.Fatalf("the page: %v %v", got, err)
	}
	got.Body.Close()
	resp, err := http.Post(srv.URL+"/votes", "application/json",
		strings.NewReader(`{"name":"alpha","vote":"keep"}`))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("the vote: %v %v", resp, err)
	}
	resp.Body.Close()
	read, err := http.Get(srv.URL + "/votes")
	if err != nil {
		t.Fatal(err)
	}
	defer read.Body.Close()
	var rec voteRecord
	if err := json.NewDecoder(read.Body).Decode(&rec); err != nil {
		t.Fatal(err)
	}
	if rec.Votes["alpha"].Vote != keep || len(rec.Queue) != 1 {
		t.Errorf("the record read back is %+v", rec)
	}
}

func TestEveryFieldThatReachesThePageIsHeldAsText(t *testing.T) {
	const raw = ` "q" <b>&`
	const held = ` &#34;q&#34; &lt;b&gt;&amp;`
	sheet := cheeptune.CutSheet{Group: "group" + raw, Mood: "mood" + raw,
		Feeling: "feeling" + raw, Intent: "intent" + raw,
		ListenFor: "listen" + raw, Voicing: "@dmg-pad", Tempo: 240,
		Meter: 4, Seconds: 4, Parts: []cheeptune.Cut{
			{Name: "one" + raw, Bar: 1, Second: 0, Change: "c",
				Feels: "feels one" + raw},
			{Name: "two" + raw, Bar: 2, Second: 1, Change: "c",
				Feels: "feels two" + raw}}}
	b, err := json.Marshal(sheet)
	if err != nil {
		t.Fatal(err)
	}
	dir := folder(t, "small")
	write(t, dir, "small.cuts.json", string(b))
	tune, err := Load(dir, "small")
	if err != nil {
		t.Fatal(err)
	}
	page, err := Page([]*Tune{tune})
	if err != nil {
		t.Fatal(err)
	}
	text := string(page)
	for _, field := range []string{"group", "mood", "feeling", "intent",
		"listen", "one", "two", "feels one", "feels two"} {
		if !strings.Contains(text, field+held) {
			t.Errorf("%q is not held as text on the page", field)
		}
		if strings.Contains(text, field+raw) ||
			strings.Contains(text, field+` "q" <b>`) {
			t.Errorf("%q reached the page raw", field)
		}
	}
	if strings.Contains(text, "<b>") {
		t.Error("an element the sheet wrote is on the page")
	}
}
