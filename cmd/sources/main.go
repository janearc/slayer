// Command sources fetches the published work slayer cites onto this
// machine and checks it against the hashes in sources.yaml. the bytes
// are kept outside any repository, on a shelf in the scratch of whoever
// runs it; the list in the repository is the claim they are held to.
//
//	SLAYER_SHELF=$HOME/slayer/sources \
//	    sources fetch [KEY ...]
//	sources check
//	sources path KEY
//	sources verify
//
// run it where sources.yaml is, at slayer's root. it is on pzm.
//
// SLAYER_SHELF names the citation shelf, the papers numbers are verified
// against. when it is set it wins. when it is unset the shelf is the
// shelf folder committed beside sources.yaml, so verify, check and path
// need no setup. fetch never fills the tree: the list holds large data
// that must not go there, so with no shelf named it refuses and says
// how to name one.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/sources"
)

// the four stamps game build writes into a command's main package
var build, built, who, sessions string

// patience is how long one source may take to come: a dem of an island
// is gigabytes.
const patience = 30 * time.Minute

// shelfDir is the directory of the citation shelf: the one SLAYER_SHELF
// names, which wins, or the shelf folder beside the list when it is
// unset. a value that is set and not usable is refused.
func shelfDir(list string) (string, error) {
	dir, err := sources.Here()
	if err == nil {
		return dir, nil
	}
	if os.Getenv(sources.Placed) == "" {
		return filepath.Join(filepath.Dir(list), treeFolder), nil
	}
	return "", err
}

// treeFolder is the folder beside the list that holds the committed
// sources.
const treeFolder = "shelf"

// main runs the command on the list where it is run, with the shelf
// shelfDir gives and the clock on the wall.
func main() {
	const list = "sources.yaml"
	dir, err := shelfDir(list)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	_, unnamed := sources.Here()
	client := &http.Client{Timeout: patience}
	made := tool(place{list: list,
		shelf:    sources.NewShelf(dir, client),
		unnamed:  unnamed,
		now:      time.Now,
		research: research.Read})
	os.Exit(made.Run(os.Args[1:], os.Stdout, os.Stderr))
}

// place is where the command works: its list, its shelf, its clock, and
// the research data it holds to the list. unnamed is the refusal fetch
// gives when the environment names no shelf, or nothing.
type place struct {
	list     string
	shelf    sources.Shelf
	unnamed  error
	now      func() time.Time
	research func() (numbers.Numbers, error)
}

// tool is the command and its four verbs.
func tool(at place) *pzm.Tool {
	made := pzm.New("sources",
		"fetch and check the published work slayer cites",
		pzm.Build(build), pzm.Built(built), pzm.Who(who),
		pzm.Sessions(sessions))
	// fetch adds files to the shelf from the network. check hashes every
	// file on the shelf, which holds a core. path and verify only read.
	fetches := verb.NewFacts(verb.WritesTo("the sources shelf"),
		verb.LeavesAlone, verb.Outside, verb.NoOne, verb.Light)
	hashes := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.FullCPU)
	reads := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	made.Verb("fetch", 'f', "bring the listed sources onto this machine",
		fetchPage, "Example_fetch", fetches, at.fetch)
	made.Verb("check", 'c', "hold every pinned source to its hash",
		checkPage, "Example_check", hashes, at.check)
	made.Verb("path", 'p', "where a source's bytes are on this machine",
		pathPage, "Example_path", reads, at.path)
	made.Verb("verify", 'v', "hold every measured number to its source",
		verifyPage, "Example_verify", reads, at.verify)
	return made
}

const fetchPage = `# fetch

fetches every source in sources.yaml onto this machine, or the ones
named after it. a pinned source already here is kept. bytes that do not
match their pin are refused and removed. a source with no pin yet is
kept, and its sha256 said, to be pinned in the list by hand.

it reaches outside the machine, to where each source is published.`

const checkPage = `# check

holds every pinned source on this machine to the sha256 the list pins,
and says each one: as pinned, missing, or changed. a source with no pin
is said to be so. it reaches nothing outside and writes nothing.`

const pathPage = `# path

says where a source's bytes are on this machine, by its key, so fermat
and a person can open the page a number came from.`

// fetch fetches the listed sources, or those named. With --dry-run it
// says what it would fetch and fetches nothing. it refuses when no
// shelf is named, as the list holds large data that must stay out of
// the tree.
func (at place) fetch(call pzm.Call) error {
	if at.unnamed != nil {
		return at.unnamed
	}
	list, err := sources.Read(at.list)
	if err != nil {
		return err
	}
	chosen, err := chosen(list, call.Args())
	if err != nil {
		return err
	}
	if call.DryRun() {
		for _, source := range chosen {
			if _, err := fmt.Fprintf(call.Out(), "%s: would fetch %s\n", source.Key(), source.From()); err != nil {
				return err
			}
		}
		return nil
	}
	if columns, live := terminalColumns(call.Out()); live {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		return at.fetchLive(ctx, call.Out(), columns, chosen)
	}
	return at.fetchLines(call.Out(), chosen)
}

// fetchLive fetches the sources with the live status drawn on out, and
// ends with one summary line. a failure is printed above the status and
// the rest are fetched. when ctx ends, it stops and leaves the terminal
// as it found it.
func (at place) fetchLive(ctx context.Context, out io.Writer,
	columns func() int, chosen []sources.Source) error {
	screen := newDisplay(out, columns, len(chosen))
	shelf := at.shelf.Counting(screen.report)
	if len(chosen) > 0 {
		screen.start(chosen[0].Key())
	}
	screen.begin()
	for _, source := range chosen {
		if ctx.Err() != nil {
			break
		}
		screen.start(source.Key())
		said, err := shelf.Fetch(ctx, source, at.now())
		if ctx.Err() != nil {
			break
		}
		screen.record(source.Key(), said, err)
	}
	screen.finish(shelf.Path(sources.Source{}), ctx.Err() != nil)
	if ctx.Err() != nil {
		return errors.New("interrupted")
	}
	if failures := screen.count.failed; failures > 0 {
		return fmt.Errorf("%d failed, said above", failures)
	}
	return nil
}

// fetchLines fetches the sources and says one line for each, as it
// always has. it is what runs when out is not a terminal.
func (at place) fetchLines(out io.Writer, chosen []sources.Source) error {
	var failed []error
	for _, source := range chosen {
		said, err := at.shelf.Fetch(context.Background(), source, at.now())
		if err != nil {
			failed = append(failed, err)
			continue
		}
		if _, err := fmt.Fprintln(out, said); err != nil {
			return errors.Join(append(failed, err)...)
		}
	}
	return errors.Join(failed...)
}

// chosen is the sources named, or every source when none is.
func chosen(list sources.List, keys []string) ([]sources.Source, error) {
	if len(keys) == 0 {
		return list.Sources(), nil
	}
	var picked []sources.Source
	for _, key := range keys {
		source, found := list.Find(key)
		if !found {
			return nil, fmt.Errorf("%s is not in the list", key)
		}
		picked = append(picked, source)
	}
	return picked, nil
}

// check holds every source to its pin.
func (at place) check(call pzm.Call) error {
	list, err := sources.Read(at.list)
	if err != nil {
		return err
	}
	var failed []error
	for _, source := range list.Sources() {
		said, err := at.shelf.Check(source)
		if err != nil {
			failed = append(failed, err)
			continue
		}
		if _, err := fmt.Fprintln(call.Out(), said); err != nil {
			return errors.Join(append(failed, err)...)
		}
	}
	return errors.Join(failed...)
}

// path says where one source's bytes are.
func (at place) path(call pzm.Call) error {
	if len(call.Args()) != 1 {
		return errors.New("name one source by its key")
	}
	list, err := sources.Read(at.list)
	if err != nil {
		return err
	}
	source, found := list.Find(call.Args()[0])
	if !found {
		return fmt.Errorf("%s is not in the list", call.Args()[0])
	}
	_, err = fmt.Fprintln(call.Out(), at.shelf.Path(source))
	return err
}
