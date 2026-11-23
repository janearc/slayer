package session

import (
	"fmt"

	"github.com/janearc/slayer/beats"
	"github.com/janearc/slayer/cheeptune"
	"github.com/janearc/slayer/cuesheet"
	"github.com/janearc/slayer/interfaces/movie"
	"github.com/janearc/slayer/job"
)

// soundtrackClock shares song time with the film's soundtrack. A supplied
// score avoids re-estimation; without one the file is analysed once at Open.
func soundtrackClock(track job.Sound) (cuesheet.Clock, error) {
	if score, voicing, tune := track.Tune(); tune {
		piece, err := cheeptune.Open(score, voicing)
		if err != nil {
			return nil, err
		}
		return cuesheet.Piece(piece)
	}
	path, _ := track.File()
	grid, err := audioGrid(track, path)
	if err != nil {
		return nil, err
	}
	from, _ := track.From()
	return cuesheet.Measured(grid, from)
}

// audioGrid checks that a retained grid belongs to these audio bytes.
// Estimated grids stay estimates; a user should check the metronome first.
func audioGrid(track job.Sound, path string) (*beats.Grid, error) {
	if score, supplied := track.Beats(); supplied {
		grid, err := beats.Read(score)
		if err != nil {
			return nil, err
		}
		if err := grid.CheckAudio(path); err != nil {
			return nil, err
		}
		return grid, nil
	}
	sound, err := movie.ReadSound(path)
	if err != nil {
		return nil, err
	}
	grid, err := beats.Detect(sound.Samples(), sound.Rate(),
		beats.Settings{Meter: 4, First: -1})
	if err != nil {
		return nil, fmt.Errorf("audio grid: %w; supply a checked score", err)
	}
	return grid, nil
}
