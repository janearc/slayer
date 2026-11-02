// Command sources fetches the published work slayer cites onto this
// machine and checks it against the hashes in sources.yaml. the bytes
// are kept outside any repository, on a shelf in the scratch of whoever
// runs it; the list in the repository is the claim they are held to.
//
//	SLAYER_SHELF=$HOME/slayer/sources \
//	    sources fetch [KEY ...]
//	sources check
//	sources path KEY
//
// run it where sources.yaml is, at slayer's root. it is on pzm. with no
// shelf named, it refuses and says how to name one.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
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

// main runs the command on the list where it is run, with the shelf the
// environment names and the clock on the wall.
func main() {
	dir, err := sources.Here()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	client := &http.Client{Timeout: patience}
	made := tool(place{list: "sources.yaml",
		shelf:    sources.NewShelf(dir, client),
		now:      time.Now,
		research: research.Read})
	os.Exit(made.Run(os.Args[1:], os.Stdout, os.Stderr))
}

// place is where the command works: its list, its shelf, its clock, and
// the research data it holds to the list.
type place struct {
	list     string
	shelf    sources.Shelf
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
// says what it would fetch and fetches nothing.
func (at place) fetch(call pzm.Call) error {
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
			fmt.Fprintf(call.Out(), "%s: would fetch %s\n", source.Key(),
				source.From())
		}
		return nil
	}
	var failed []error
	for _, source := range chosen {
		said, err := at.shelf.Fetch(context.Background(), source, at.now())
		if err != nil {
			failed = append(failed, err)
			continue
		}
		fmt.Fprintln(call.Out(), said)
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
		fmt.Fprintln(call.Out(), said)
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
