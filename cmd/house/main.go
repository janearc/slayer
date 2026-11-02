// Command house draws the shoebox from its set file: the house, its
// lights, the sky and saturn, through the cameras the set names.
//
// doc/sets.md says what a set file may say.
//
// one camera is drawn whole, at the film's frame, and its raw is written
// beside the picture, so the shot can be printed again at another
// exposure or white balance with develop. several are drawn side by
// side in one picture, each at half the frame.
//
//	house [-set sets/shoebox.yaml] [-camera front] [-out out/house.png]
//	house -camera road,garden -at "the box=-1,0,0"
//
// -at says where an asset stands that a camera stands off, in the
// world's space, since a set file does not place its assets yet.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// main reads the set file, draws the cameras asked for, and writes the
// picture
func main() {
	setPath := flag.String("set", "sets/shoebox.yaml", "the set file")
	cameras := flag.String("camera", "front",
		"the camera to draw by name, or several with commas between")
	out := flag.String("out", "out/house.png", "where the picture goes")
	at := places{}
	flag.Var(at, "at", "where a set's asset stands: name=x,y,z")
	flag.Parse()
	said, err := shoot(*setPath, strings.Split(*cameras, ","), at, *out)
	if err != nil {
		fail(err)
	}
	fmt.Print(said)
}

// fail says what went wrong and stops
func fail(err error) {
	fmt.Fprintln(os.Stderr, "house:", err)
	os.Exit(1)
}
