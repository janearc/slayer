package main

import (
	"fmt"
	"strconv"
	"strings"
)

// cues is the moment's cue state as lightshow reads it: the lines
// show.Moment.Lines writes, the keys the rig drawing needs.
type cues struct {
	rig, sweep, gate, gateLeft, gateRight float64
	beams, white, magenta, parity         int
	colour                                [3]float64
	figure, searchMove                    string
	figurePhase, search, searchPhase      float64
	searchMovePhase, searchGate           float64
	floodLeft, floodRight, overhead       float64
	master, exposure                      float64
}

// numbers are the cue keys read as numbers, and where each goes.
func (state *cues) numbers() map[string]*float64 {
	return map[string]*float64{"rig": &state.rig, "sweep": &state.sweep,
		"gate": &state.gate, "gate_left": &state.gateLeft,
		"gate_right": &state.gateRight, "figure_phase": &state.figurePhase,
		"search": &state.search, "search_phase": &state.searchPhase,
		"search_move_phase": &state.searchMovePhase,
		"search_gate":       &state.searchGate, "flood_left": &state.floodLeft,
		"flood_right": &state.floodRight, "overhead": &state.overhead,
		"master": &state.master, "exposure": &state.exposure}
}

// counts are the cue keys read as whole numbers.
func (state *cues) counts() map[string]*int {
	return map[string]*int{"beams": &state.beams, "white": &state.white,
		"magenta": &state.magenta, "parity": &state.parity}
}

// cuesOf reads a moment's lines. A key it does not use is left alone; a
// number that is not one is refused by its key.
func cuesOf(lines string) (cues, error) {
	state := cues{figure: "fan", searchMove: "sweep", gateLeft: -1,
		gateRight: -1, parity: -1}
	numbers, counts := state.numbers(), state.counts()
	for _, line := range strings.Split(strings.TrimSpace(lines), "\n") {
		key, value, _ := strings.Cut(line, "=")
		var err error
		switch {
		case numbers[key] != nil:
			*numbers[key], err = strconv.ParseFloat(value, 64)
		case counts[key] != nil:
			*counts[key], err = strconv.Atoi(value)
		case key == "figure":
			state.figure = value
		case key == "search_move":
			state.searchMove = value
		case key == "colour":
			err = state.readColour(value)
		}
		if err != nil {
			return cues{}, fmt.Errorf("the cue %s=%q: %w", key, value, err)
		}
	}
	return state, nil
}

// readColour reads the beams' colour, three numbers.
func (state *cues) readColour(value string) error {
	parts := strings.Split(value, ",")
	if len(parts) != len(state.colour) {
		return fmt.Errorf("a colour is three numbers")
	}
	for c, part := range parts {
		number, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return err
		}
		state.colour[c] = number
	}
	return nil
}

// gates is each bank's pulse, as lightshow takes them: the one gate,
// unless the cues give a bank its own.
func (state cues) gates() [2]float64 {
	gates := [2]float64{state.gate, state.gate}
	if state.gateLeft >= 0 {
		gates[0] = state.gateLeft
	}
	if state.gateRight >= 0 {
		gates[1] = state.gateRight
	}
	return gates
}
