package hillside

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/sources"
)

// fetchFlag is the flag that fetches what a set's datasets lack, as it is
// said in an error; and listFile the list the datasets are found in.
const (
	fetchFlag = "--fetch-terrain"
	listFile  = "sources.yaml"
)

// patience is how long one dataset may take to come: a dem of an island
// is gigabytes. picked: thirty minutes, the wait cmd/sources gives a
// source; defined here as the same number.
const patience = 30 * time.Minute

const (
	// exitDrawn is the exit number of a picture written or a check that
	// held; defined: zero, hillside's own.
	exitDrawn = 0
	// exitFailed is the exit number of a set that could not be staged or
	// drawn; defined: one, hillside's own.
	exitFailed = 1
)

// onlyOne is how many of the three terrain flags may be given; defined:
// one.
const onlyOne = 1

// terrainAsk is which of the three things the command line asked to be
// done with a set's datasets before it is staged.
type terrainAsk struct{ fetch, refresh, check bool }

// terrainFlags declares the three flags of terrainAsk on flags.
func terrainFlags(flags *flag.FlagSet, ask *terrainAsk) {
	flags.BoolVar(&ask.fetch, "fetch-terrain", false,
		"fetch the datasets the set names (terrain, moon face) that are "+
			"not on the shelf, then draw")
	flags.BoolVar(&ask.refresh, "refresh-terrain", false,
		"fetch the datasets the set names again, replace the shelf's "+
			"files with good ones, then draw")
	flags.BoolVar(&ask.check, "check-terrain", false,
		"say whether the shelf's datasets are the source's now; "+
			"download nothing, draw nothing")
}

// asked is whether any of the three was asked for.
func (ask terrainAsk) asked() bool { return ask.fetch || ask.refresh || ask.check }

// mode is the mode of the one thing asked.
func (ask terrainAsk) mode() sources.Mode {
	switch {
	case ask.refresh:
		return sources.Refreshing
	case ask.check:
		return sources.Checking
	}
	return sources.Fetching
}

// sound is what is wrong with what was asked of the datasets, by name:
// the flags are for a set, and one at a time.
func (ask terrainAsk) sound(setFile string) error {
	count := nothing
	for _, each := range []bool{ask.fetch, ask.refresh, ask.check} {
		if each {
			count++
		}
	}
	switch {
	case count > onlyOne:
		return fmt.Errorf("%w: only one of -fetch-terrain, "+
			"-refresh-terrain and -check-terrain at a time", errAsked)
	case count == onlyOne && setFile == "":
		return fmt.Errorf("%w: -fetch-terrain, -refresh-terrain and "+
			"-check-terrain need -set FILE, which names the datasets",
			errAsked)
	}
	return nil
}

// tendTerrain does what the terrain flags asked, before the set is
// staged, saying what it did to said. it gives the exit number and
// whether the command is done: a check is done when it has compared, and
// any of the three is done when it failed.
func tendTerrain(given asked, said, wrong io.Writer) (int, bool) {
	err := tendSet(given, said)
	if err != nil {
		fmt.Fprintln(wrong, "hillside:", err)
		return exitFailed, true
	}
	return exitDrawn, given.terrain.check
}

// tendSet tends the datasets a set names, in the mode asked, with the
// shelf the environment names and the sources.yaml nearest the set.
func tendSet(given asked, said io.Writer) error {
	written, err := os.ReadFile(given.set)
	if err != nil {
		return err
	}
	read, err := set.Read(given.set, written)
	if err != nil {
		return err
	}
	dir, err := sources.Here()
	if err != nil {
		return err
	}
	listed, err := nearest(filepath.Dir(given.set), listFile)
	if err != nil {
		return err
	}
	list, err := sources.Read(listed)
	if err != nil {
		return err
	}
	var origin sources.Origin
	wants := datasetsOf(read)
	if len(wants) == nothing {
		fmt.Fprintf(said, "%s names no terrain and no moon face; nothing "+
			"to tend\n", given.set)
		return nil
	}
	client := &http.Client{Timeout: patience}
	shelf := sources.NewShelf(dir, client)
	origin = sources.NewListed(list, client)
	return shelf.Tend(context.Background(), origin, given.terrain.mode(),
		wants, said)
}

// datasetsOf is the datasets a set reads from the shelf: its terrain,
// and the face of its moon where it is lit by one that has a face. these
// are the ones staging reads through pinnedAssetAt.
func datasetsOf(read set.Set) []sources.Want {
	var wants []sources.Want
	if ground, has := read.Ground(); has {
		terrain := sources.NewWant(ground.From(), ground.File())
		wants = append(wants, terrain)
	}
	if moon, has := read.Moonlit(); has && read.Light() == "moon" &&
		moon.Face() != "" {
		face := sources.NewWant(moon.Face(), "")
		wants = append(wants, face)
	}
	return wants
}
