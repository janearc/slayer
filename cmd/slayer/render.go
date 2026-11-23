package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/janearc/slayer/session"
)

// where render's arguments hold the file, and the settings after it.
const (
	fileAt     = 0 // defined: the file comes first
	settingsAt = 1 // defined: the settings come after the file
)

// render draws a set file the way hillside does: the arguments are the
// file, then settings: --width, --height and --rays, --fetch-terrain,
// --refresh-terrain and --check-terrain, and hillside's other flags
// that may be given with a set.
//
// it says what it did to said and what went wrong to wrong, and gives
// back hillside's exit number: nought for a picture written, one for a set
// it could not stage or draw, two for a setting it could not take. a call
// with no file says how render is used and gives two.
func render(arguments []string, said, wrong io.Writer) int {
	if len(arguments) == nothing || strings.HasPrefix(arguments[fileAt], "-") {
		fmt.Fprintln(wrong, "usage: slayer render FILE [settings]")
		return exitUsage
	}
	given, err := settings(arguments[settingsAt:])
	if err != nil {
		fmt.Fprintln(wrong, err)
		return exitUsage
	}
	settings := append([]string{"-set", arguments[fileAt]}, given...)
	stamp := session.Stamped(build, built, who, sessions)
	return session.Run(settings, said, wrong, stamp)
}

// valueAfter is how far past a flag its value stands, when it is the next
// word.
const valueAfter = 1 // defined: the next word

// the size flags render takes, as puzzlemouth names a flag: lowercase words
// after two hyphens. each stands for the flag hillside reads it as.
var sizeFlags = map[string]string{
	"--width":  "-w",
	"--height": "-h",
	"--rays":   "-n",
}

// the terrain flags render takes, which are switches with no value, each
// standing for the flag of the same name hillside reads with one hyphen.
var terrainFlags = map[string]string{
	"--fetch-terrain":   "-fetch-terrain",
	"--refresh-terrain": "-refresh-terrain",
	"--check-terrain":   "-check-terrain",
}

// the flags hillside reads as the size, which render does not take, and the
// message that says what it takes in their place.
var (
	shortSizeFlags = map[string]bool{"-w": true, "-h": true, "-n": true}
	errSizeFlag    = errors.New("slayer render: the size flags are " +
		"--width, --height and --rays, and the terrain flags are " +
		"--fetch-terrain, --refresh-terrain and --check-terrain")
)

// settings is render's settings as hillside reads them: --fetch-terrain,
// --refresh-terrain and --check-terrain stand for hillside's flags of the
// same names; --width, --height and --rays, each with its value as the next word or after an equals
// sign, become -w, -h and -n, and the rest go through as they are. it
// refuses -w, -h and -n, any other word with two hyphens, and a size flag
// with no value, each by name, and says the long flags it takes.
func settings(given []string) ([]string, error) {
	var read []string
	for at := nothing; at < len(given); at++ {
		word := given[at]
		name, value, joined := strings.Cut(word, "=")
		short, long := sizeFlags[name]
		switched, terrain := terrainFlags[word]
		switch {
		case terrain:
			read = append(read, switched)
		case long && joined:
			read = append(read, short, value)
		case long && at+valueAfter < len(given):
			at++
			read = append(read, short, given[at])
		case long:
			return nil, fmt.Errorf("%w: %s needs a value", errSizeFlag, name)
		case shortSizeFlags[word] || strings.HasPrefix(word, "--"):
			return nil, fmt.Errorf("%w: unknown flag %s", errSizeFlag, word)
		default:
			read = append(read, word)
		}
	}
	return read, nil
}
