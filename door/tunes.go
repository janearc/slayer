package door

// tunes is a tune library folder's audition, behind the door: it serves
// the folder's page and records keep and toss votes on this machine
// only, as cheeptune/audition's server does.
//
// With one switch in place of serving it checks the tunes, or writes
// their films, their stills, the page or the voicings' instrument
// sheet. It replaces cmd/tunefilms, whose five verbs are the switches.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/cheeptune/audition"
	"github.com/janearc/slayer/interfaces/file"
)

// tunesHelp is tunes' page.
const tunesHelp = `# tunes

slayer tunes DIR serves a tune library folder's audition page and
records each keep and toss vote in the folder's votes.json. It listens
on this machine only: --address is a loopback address, 127.0.0.1:PORT
or localhost:PORT, and any other is refused. It says the address to
open, and serves until control-c.

One switch in place of serving, each over every tune of the folder or
the names given after it:

--check loads each tune and says its length, its peak and its voicing,
and fails if any will not load. --films writes NAME.mp4 beside each
tune. --plates writes NAME.png, the still a film is made from, beside
each tune or in the folder --out names. --page writes index.html, the
page the votes are cast on.

slayer tunes --sounds VOICING... draws one note of each instrument of
the studio voicings named, a row a voicing, to --out, or to
out/sounds.png. slayer list voicings says the voicings.

--dry-run says what would be written and writes nothing.
`

// The address tunes listens on when none is said: this machine only.
// The port is picked, by the door verbs' slice, 2026-11-12: one no
// other Slayer tool serves on.
const tunesAddress = "127.0.0.1:8417"

// The names of tunes' flags.
const (
	addressFlag = "address"
	checkFlag   = "check"
	filmsFlag   = "films"
	platesFlag  = "plates"
	pageFlag    = "page"
	soundsFlag  = "sounds"
)

// soundsName is the instrument sheet's picture when --out says none.
const soundsName = "sounds.png"

// What tunes writes, beside the tunes.
const (
	pageName  = "index.html"
	votesName = "votes.json"
)

// shutdownWait is how long a stopped server waits for the requests it is
// answering. Picked, by the door verbs' slice, 2026-11-12: a vote is a
// name and a word.
const shutdownWait = 2 * time.Second

// tuneJob is one of the switches: what it does to the tunes named in a
// folder, and what it writes, in words, for a dry run.
type tuneJob struct {
	flag   string
	writes string // "%s" is a tune's name; empty writes nothing
	do     func(at tuneAt, names []string) error
}

// tuneAt is where a switch works: what it says to, the library folder,
// and the folder it writes in, the library's unless --out says another.
type tuneAt struct {
	out     io.Writer
	dir, to string
}

// tuneJobs is every switch, in the order the help says them.
var tuneJobs = []tuneJob{
	{checkFlag, "", checkTunes},
	{filmsFlag, "%s.mp4", filmTunes},
	{platesFlag, "%s.png", plateTunes},
	{pageFlag, pageName, pageTunes},
}

// auditioning is the tunes verb, with what it runs under.
type auditioning struct{ env Env }

// tunes is the verb: a switch's job over the folder's tunes, or the
// folder served.
func (held auditioning) tunes(call pzm.Call) error {
	if call.On(soundsFlag) {
		return held.sounds(call)
	}
	if len(call.Args()) == nothing {
		return errors.New("tunes takes a library folder: " +
			"slayer tunes DIR [NAME...]")
	}
	dir := call.Args()[nothing]
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(held.env.directory, dir)
	}
	chosen, err := chosenJob(call)
	if err != nil {
		return err
	}
	if chosen == nil {
		return held.serve(call, dir)
	}
	names, err := tuneNames(dir, call.Args()[one:])
	if err != nil {
		return err
	}
	at := tuneAt{out: call.Out(), dir: dir, to: dir}
	if call.Given("out") {
		at.to = resolved(call.Value("out"), held.env.directory)
	}
	if call.DryRun() {
		return wouldWrite(*chosen, at, names)
	}
	return chosen.do(at, names)
}

// sounds draws the instrument sheet of the voicings named, or in a dry
// run says where it would.
func (held auditioning) sounds(call pzm.Call) error {
	if len(call.Args()) == nothing {
		return errors.New("--sounds takes the voicings to draw: slayer " +
			"tunes --sounds VOICING...; slayer list voicings says them")
	}
	path := filepath.Join(held.env.directory, outFolder, soundsName)
	if call.Given("out") {
		path = resolved(call.Value("out"), held.env.directory)
	}
	if call.DryRun() {
		_, err := fmt.Fprintln(call.Out(), "would write "+path)
		return err
	}
	sheet, err := cheeptune.InstrumentSheet(call.Args())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), folderMode); err != nil {
		return err
	}
	if err := file.PNG(path, sheet, square); err != nil {
		return err
	}
	_, err = fmt.Fprintln(call.Out(), "wrote "+path)
	return err
}

// chosenJob is the switch given, or nil to serve. Two switches, or an
// address with a switch, are refused by name.
func chosenJob(call pzm.Call) (*tuneJob, error) {
	var chosen []tuneJob
	for _, each := range tuneJobs {
		if call.On(each.flag) {
			chosen = append(chosen, each)
		}
	}
	switch {
	case len(chosen) > one:
		return nil, fmt.Errorf("tunes does one thing at a time; give "+
			"one of --%s, --%s, --%s or --%s", checkFlag, filmsFlag,
			platesFlag, pageFlag)
	case len(chosen) == one && call.Given(addressFlag):
		return nil, fmt.Errorf("--%s is where the page is served; it "+
			"does nothing with --%s", addressFlag, chosen[nothing].flag)
	case call.Given("out") && (len(chosen) == nothing ||
		chosen[nothing].flag != platesFlag):
		return nil, fmt.Errorf("--out says where --plates or --sounds " +
			"write; the rest write beside the tunes")
	case len(chosen) == one:
		return &chosen[nothing], nil
	}
	return nil, nil
}

// tuneNames is the names given, or every tune of the folder when none
// is; a folder with no tunes is refused by name.
func tuneNames(dir string, given []string) ([]string, error) {
	if len(given) > nothing {
		return given, nil
	}
	names, err := audition.Names(dir)
	if err != nil {
		return nil, fmt.Errorf("the folder %s cannot be read: %w", dir,
			err)
	}
	if len(names) == nothing {
		return nil, fmt.Errorf("the folder %s holds no tunes; name the "+
			"folder that has NAME.json files", dir)
	}
	return names, nil
}

// wouldWrite says each file a switch would write, and writes none.
func wouldWrite(chosen tuneJob, at tuneAt, names []string) error {
	var lines []string
	switch {
	case chosen.writes == "":
		lines = []string{"would write nothing"}
	case !strings.Contains(chosen.writes, "%s"):
		lines = []string{"would write " + filepath.Join(at.to,
			chosen.writes)}
	default:
		for _, name := range names {
			lines = append(lines, "would write "+filepath.Join(at.to,
				fmt.Sprintf(chosen.writes, name)))
		}
	}
	_, err := fmt.Fprintln(at.out, strings.Join(lines, "\n"))
	return err
}

// checkTunes loads each tune and says what it is; it fails, after saying
// them all, if any would not load.
func checkTunes(at tuneAt, names []string) error {
	out, dir := at.out, at.dir
	bad := nothing
	for _, name := range names {
		tune, err := audition.Load(dir, name)
		if err != nil {
			fmt.Fprintf(out, "FAIL %s\n%v\n", name, err)
			bad++
			continue
		}
		fmt.Fprintf(out, "ok   %-16s %6.1f s  peak %.2f  %s\n", name,
			tune.Piece.Duration().Seconds(), tune.Piece.Peak,
			tune.Sheet.Voicing)
	}
	if bad > nothing {
		return fmt.Errorf("%d of %d tunes would not load", bad, len(names))
	}
	return nil
}

// filmTunes writes NAME.mp4 beside each tune.
func filmTunes(at tuneAt, names []string) error {
	out, dir := at.out, at.dir
	for _, name := range names {
		tune, err := audition.Load(dir, name)
		if err != nil {
			return err
		}
		path := filepath.Join(dir, name+movieExtension)
		if err := audition.Film(tune, path); err != nil {
			return err
		}
		fmt.Fprintln(out, "wrote "+path)
	}
	return nil
}

// plateTunes writes NAME.png beside each tune, or in --out's folder:
// the still its film is made from, to look at without a player.
func plateTunes(at tuneAt, names []string) error {
	out, dir := at.out, at.dir
	if err := os.MkdirAll(at.to, folderMode); err != nil {
		return err
	}
	for _, name := range names {
		tune, err := audition.Load(dir, name)
		if err != nil {
			return err
		}
		still, err := audition.Plate(tune)
		if err != nil {
			return err
		}
		path := filepath.Join(at.to, name+pictureExtension)
		if err := file.PNG(path, still, square); err != nil {
			return err
		}
		fmt.Fprintln(out, "wrote "+path)
	}
	return nil
}

// pageTunes writes index.html for the tunes.
func pageTunes(at tuneAt, names []string) error {
	out, dir := at.out, at.dir
	var tunes []*audition.Tune
	for _, name := range names {
		tune, err := audition.Read(dir, name)
		if err != nil {
			return err
		}
		tunes = append(tunes, tune)
	}
	page, err := audition.Page(tunes)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, pageName)
	if err := os.WriteFile(path, page, fileMode); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "wrote "+path)
	return err
}

// serve serves the folder on the address asked for, a loopback one,
// until the verb's context is cancelled. A dry run checks the folder and
// says where it would serve, and listens on nothing.
func (held auditioning) serve(call pzm.Call, dir string) error {
	if len(call.Args()) > one {
		return errors.New("serving takes the folder alone; names go " +
			"with --check, --films, --plates or --page")
	}
	address := tunesAddress
	if call.Given(addressFlag) {
		address = call.Value(addressFlag)
	}
	if err := audition.Loopback(address); err != nil {
		return err
	}
	if call.DryRun() {
		if _, err := tuneNames(dir, nil); err != nil {
			return err
		}
		_, err := fmt.Fprintf(call.Out(), "would serve %s on http://%s/ "+
			"and write each vote to %s\n", dir, address,
			filepath.Join(dir, votesName))
		return err
	}
	listener, err := (&net.ListenConfig{}).Listen(held.env.ctx, "tcp",
		address)
	if err != nil {
		return err
	}
	shown, port, err := shownAddress(address, listener.Addr())
	if err != nil {
		return errors.Join(err, listener.Close())
	}
	server, err := audition.NewServer(dir, port)
	if err != nil {
		return errors.Join(err, listener.Close())
	}
	defer server.Close()
	return serveOn(held.env.ctx, call.Out(), listener, shown,
		server.Handler())
}

// shownAddress is the address to open: the host asked for, which the
// server's Host rule takes, and the port bound, which may have been
// asked as nought. A listener for localhost may be bound to ::1, and
// that name the rule does not take.
func shownAddress(asked string, bound net.Addr) (string, string, error) {
	host, _, err := net.SplitHostPort(asked)
	if err != nil {
		return "", "", err
	}
	_, port, err := net.SplitHostPort(bound.String())
	if err != nil {
		return "", "", err
	}
	return net.JoinHostPort(host, port), port, nil
}

// serveOn serves a handler on a listener until the context is cancelled,
// having said the address to open. A request still open when the wait
// for it ends, a film being streamed, is closed, and the stop is clean.
func serveOn(ctx context.Context, out io.Writer, listener net.Listener,
	shown string, handler http.Handler) error {
	served := &http.Server{Handler: handler,
		ReadHeaderTimeout: shutdownWait}
	stopped := make(chan error, one)
	go func() { stopped <- served.Serve(listener) }()
	fmt.Fprintf(out, "open http://%s/ to listen and vote; control-c "+
		"stops\n", shown)
	select {
	case err := <-stopped:
		return err
	case <-ctx.Done():
	}
	quiet, cancel := context.WithTimeout(context.Background(), shutdownWait)
	defer cancel()
	err := served.Shutdown(quiet)
	if errors.Is(err, context.DeadlineExceeded) {
		err = served.Close()
	}
	if err != nil {
		return err
	}
	if err := <-stopped; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// declareTunes adds tunes to the tool: it writes beside the tunes.
func declareTunes(made *pzm.Tool, env Env) {
	writes := verb.WritesTo("the library folder: votes, films, stills " +
		"or the page")
	serves := verb.NewFacts(writes, verb.Changes, verb.ThisMachine,
		verb.NoOne, verb.Light)
	held := auditioning{env: env}
	made.Verb("tunes", verb.NoShort, "serve a tune library's audition "+
		"page, or check, film or page its tunes", tunesHelp,
		"Example_tunes", serves, held.tunes,
		pzm.Value(addressFlag, verb.NoShort, "where to serve the page, "+
			"on this machine; "+tunesAddress+" unsaid"),
		pzm.Switch(checkFlag, verb.NoShort, "check each tune loads"),
		pzm.Switch(filmsFlag, verb.NoShort, "write each tune's film"),
		pzm.Switch(platesFlag, verb.NoShort, "write each tune's still"),
		pzm.Switch(pageFlag, verb.NoShort, "write the voting page"),
		pzm.Switch(soundsFlag, verb.NoShort, "draw the voicings' "+
			"instrument sheet"),
		pzm.Value("out", verb.NoShort, "the folder for --plates, or "+
			"the picture for --sounds"))
}
