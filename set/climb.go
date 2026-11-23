package set

import (
	yaml "gopkg.in/yaml.v3"

	"github.com/janearc/slayer/manufacture/cities"
)

// The most rows and runs a climb may ask for, and the widest a lane may
// be in metres. They bound a typing slip, not a town: a hundred rows
// of a hundred runs is more than any window of ground holds.
const (
	mostRows = 100
	mostRuns = 100
	mostLane = 1000
)

// The two words a climb says its door in: the fronts of its rows, where the
// doors are, look down the slope or up it.
const (
	doorDownhill = "downhill"
	doorUphill   = "uphill"
)

// Climb is how a stand's rows of terraces climb the slope, and whether the
// stand says one. Only a stand of the habit contour does.
func (made Stand) Climb() (cities.Climb, bool) {
	if made.climb == nil {
		return cities.Climb{}, false
	}
	return *made.climb, true
}

// climb reads how rows of terraces climb a slope, every part said: how
// many rows, how many runs of terrace make a row, the lane between a row
// and the one above it, and which way the doors look.
func (reading reader) climb(into **cities.Climb) step {
	return func(_, value *yaml.Node) error {
		var rows, runs float64
		var door string
		got := cities.Climb{}
		err := reading.keys(value, map[string]step{
			"rows": reading.whole(&rows, 1, mostRows),
			"runs": reading.whole(&runs, 1, mostRuns),
			"lane": reading.quantity(&got.Lane, "m", 0, mostLane),
			"door": reading.word(&door, doorDownhill, doorUphill),
		}, nil)
		if err == nil {
			err = reading.all(value, "a climb", "rows", "runs", "lane",
				"door")
		}
		if err != nil {
			return err
		}
		got.Rows, got.Runs = int(rows), int(runs)
		got.Uphill = door == doorUphill
		*into = &got
		return nil
	}
}
