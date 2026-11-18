package main

import (
	"image/color"
	"testing"
)

// the squares of a bar in three, at 320 by 180: the first starts at x
// 83 and each is 43 wide with a gap of 12, from y 73. A fourth square
// in four would cover x 221 to 264.
const (
	threeFirstX  = 104
	threeThirdX  = 214
	fourFourthX  = 250
	squaresY     = 94
	fourthSquare = 3
)

// the fourth beat of a score in three is bar two's first: the white
// square is struck, the third rests, and there is no fourth square.
func TestABarOfThreeHasThreeSquares(t *testing.T) {
	score := parsedScore(t, "  meter: 3\n")
	frame := newFrame()
	drawFrame(frame, score.Beats()[fourthSquare], score, nil)
	white := color.RGBA{255, 255, 255, 255}
	resting := color.RGBA{18, 48, 24, 255}
	if got := frame.RGBAAt(threeFirstX, squaresY); got != white {
		t.Errorf("the one of bar two is %v", got)
	}
	if got := frame.RGBAAt(threeThirdX, squaresY); got != resting {
		t.Errorf("the third square is %v", got)
	}
	if got := frame.RGBAAt(fourFourthX, squaresY); got != ground {
		t.Errorf("where a fourth square would be is %v", got)
	}
}

// the anti-test: the same beat in four is the bar's fourth, so its
// fourth square is struck and its one rests.
func TestTheSameBeatInFourIsTheFourth(t *testing.T) {
	score := parsedScore(t, "")
	frame := newFrame()
	drawFrame(frame, score.Beats()[fourthSquare], score, nil)
	if got := frame.RGBAAt(fourFourthX, squaresY); got !=
		(color.RGBA{60, 255, 90, 255}) {
		t.Errorf("the fourth square in four is %v", got)
	}
	if got := frame.RGBAAt(threeFirstX, squaresY); got ==
		(color.RGBA{255, 255, 255, 255}) {
		t.Errorf("the one in four is struck on the fourth beat")
	}
}
