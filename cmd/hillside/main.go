// Command hillside draws a hillside of made things: as many spruces, ferns,
// stones, logs, cans, campfires, spotlights and masers as are asked for, each
// from the manufactory, each from its own seed or one of a variety, stood on a
// ground at seeded places.
//
//	hillside [-seed 1] [-spruce 5] [-fern 24] [-boulder 8] ...
//	         [-variety 0] [-apart 2.5] [-night] [-above] [-from 0]
//	         [-wide 55] [-tilt 90] [-earth] [-sky painted|black]
//	         [-gpu] [-at 0s]
//	         [-w 320] [-h 200] [-n 2] [-out out/hillside.png]
//	         [-raw out/hillside.pfm] [-cpuprofile path]
//	hillside [the same flags] -prove out/hillside.pfm [-proof dir]
//	         [-pixel X,Y ...] [-many 64]
//	hillside -help
//
// -raw keeps the picture's light as well as its print. -prove draws
// nothing: with the flags that drew a raw, it makes 64 of its pixels
// again and proves each is the raw's bit for bit, with a sheet of their
// work in -proof, by default beside the raw (see slayer's fermat).
//
// -pixel names a pixel to prove first, and may be given again; -many is
// how many more the raw's bytes pick. the picks find a fault that runs
// through a picture, and a single pixel changed only where it is named.
//
// -h is the picture's height, as in slayer's other commands, so help
// is -help.
//
// there is a flag for every kind the manufactory has; a new kind there is a new
// flag here. the same flags and the same seed draw the same picture.
//
// the command adds up the faces its things keep, each distinct mesh once,
// and refuses a hillside past -faces of them. the default is a memory
// budget of 5 GB at the 1,300 bytes a kept face measured on 2026-11-10 (see
// hillside's bytesPerKeptFace), which is about 3.8 million faces. a needled
// spruce keeps about ninety-seven thousand.
//
// -variety makes that many of each kind and stands each wherever it is
// picked, so a forest keeps a few trees' faces however many it draws.
// with none, every thing is its own.
package main

import (
	"os"
	"slices"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/session"
)

// build, built, who and sessions are set by the linker; they say which build
// this is.
var build, built, who, sessions string

// main is hillside.Run, and the number it says. hillside --age and --who say
// which build this is.
func main() {
	arguments := os.Args[1:]
	if slices.Contains(arguments, "--age") || slices.Contains(arguments, "--who") {
		tool := stamped()
		os.Exit(tool.Run(arguments, os.Stdout, os.Stderr))
	}
	stamp := session.Stamped(build, built, who, sessions)
	os.Exit(session.Run(arguments, os.Stdout, os.Stderr, stamp))
}

// stamped is the tool that answers which build this is.
func stamped() *pzm.Tool {
	return pzm.New("hillside", "draw procedural scenery and frame sequences",
		pzm.Build(build), pzm.Built(built), pzm.Who(who), pzm.Sessions(sessions))
}
