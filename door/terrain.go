package door

// Terrain tending: draw --fetch-terrain and --refresh-terrain tend the
// datasets of the set the file names, and then draw; assess FILE
// --check-terrain compares them with the source and draws nothing, and
// its exit says whether the shelf agrees. The tending is the session's
// (session.TendTerrain): the shelf is SLAYER_SHELF's, the list of
// sources the sources.yaml nearest the set. At most one is asked.

import (
	"errors"
	"fmt"
	"io"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/session"
	"github.com/janearc/slayer/sources"
)

// The names of the terrain flags.
const (
	fetchTerrainFlag   = "fetch-terrain"
	refreshTerrainFlag = "refresh-terrain"
	checkTerrainFlag   = "check-terrain"
)

// tendWay is one way of tending a set's datasets: the flag that asks
// for it, its mode, and what a dry run says it would do, with the set's
// path in it.
type tendWay struct {
	flag  string
	mode  sources.Mode
	would string
}

// The ways of tending, each the flag's own.
var (
	fetchWay = tendWay{fetchTerrainFlag, sources.Fetching,
		"would fetch what the shelf lacks of the datasets %s names"}
	refreshWay = tendWay{refreshTerrainFlag, sources.Refreshing,
		"would fetch the datasets %s names again and replace the " +
			"shelf's with good ones"}
	checkWay = tendWay{checkTerrainFlag, sources.Checking,
		"would compare the shelf with the source for the datasets %s " +
			"names, downloading nothing"}
)

// drawTends is the ways draw tends before it draws.
var drawTends = []tendWay{fetchWay, refreshWay}

// errManyTends is the refusal of more than one way of tending at a
// time.
var errManyTends = errors.New("only one of --fetch-terrain, " +
	"--refresh-terrain and --check-terrain at a time")

// tendAsked is the way the call asks of those its verb declares, and
// whether it asks any; asking for more than one is refused.
func tendAsked(call pzm.Call, ways []tendWay) (tendWay, bool, error) {
	var asked []tendWay
	for _, each := range ways {
		if call.On(each.flag) {
			asked = append(asked, each)
		}
	}
	switch len(asked) {
	case nothing:
		return tendWay{}, false, nil
	case one:
		return asked[nothing], true, nil
	}
	return tendWay{}, false, errManyTends
}

// setToTend is the path of the set a job's scene is; a job of any other
// scene has no datasets to tend, and is refused by the way's flag.
func setToTend(wanted job.Job, way tendWay) (string, error) {
	path, isSet := wanted.Scene().Set()
	if !isSet {
		return "", fmt.Errorf("--%s tends the datasets a set names, and "+
			"this job's scene is not a set", way.flag)
	}
	return path, nil
}

// tended tends the datasets of the job's set in the way given, saying
// what it did on out.
func tended(out io.Writer, way tendWay, wanted job.Job) error {
	path, err := setToTend(wanted, way)
	if err != nil {
		return err
	}
	return session.TendTerrain(path, way.mode, out)
}

// wouldTend says on out the tending a dry run would do, and does none.
func wouldTend(out io.Writer, way tendWay, wanted job.Job) error {
	path, err := setToTend(wanted, way)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, way.would+"\n", path)
	return err
}

// terrain is assess's check of the shelf: the file's set compared with
// the source, nothing downloaded and nothing drawn. It fails where the
// shelf differs from the source or lacks a dataset.
func (held assessing) terrain(call pzm.Call, given string) error {
	if err := strays(call, terrainTakes, checkTerrainFlag); err != nil {
		return err
	}
	wanted, err := jobOf(given)
	if err != nil {
		return err
	}
	if call.DryRun() {
		return wouldTend(call.Out(), checkWay, wanted)
	}
	return tended(call.Out(), checkWay, wanted)
}
