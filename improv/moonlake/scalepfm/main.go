// Command scalepfm is improv/moonlake's: a raw's light multiplied by two
// to the power of some stops, so a frame kept as half-float OpenEXR holds
// it where half-floats are fine. The scene's raw light sits near 1e-5 to
// 1e-8 of a joule a square metre, below half's precision; scaled by the
// 23 stops develop gives the moon, it sits near 0.01 to 100.
//
//	scalepfm in.pfm out.pfm stops
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
)

// main scales the raw by the stops and writes it.
func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "scalepfm in.pfm out.pfm stops")
		os.Exit(2)
	}
	stops, err := strconv.ParseFloat(os.Args[3], 64)
	if err != nil {
		fail(err)
	}
	in, err := os.Open(os.Args[1])
	if err != nil {
		fail(err)
	}
	defer in.Close()
	r := bufio.NewReader(in)
	var kind string
	var width, height int
	var scale float64
	if _, err := fmt.Fscan(r, &kind, &width, &height, &scale); err != nil {
		fail(err)
	}
	r.ReadByte()
	if kind != "PF" {
		fail(fmt.Errorf("%s is not a colour pfm", os.Args[1]))
	}
	var order binary.ByteOrder = binary.LittleEndian
	if scale > 0 {
		order = binary.BigEndian
	}
	light := make([]float32, width*height*3)
	if err := binary.Read(r, order, light); err != nil {
		fail(err)
	}
	k := float32(math.Exp2(stops))
	for i := range light {
		light[i] *= k
	}
	out, err := os.Create(os.Args[2])
	if err != nil {
		fail(err)
	}
	w := bufio.NewWriter(out)
	fmt.Fprintf(w, "PF\n%d %d\n%g\n", width, height, scale)
	if err := binary.Write(w, order, light); err != nil {
		fail(err)
	}
	if err := w.Flush(); err != nil {
		fail(err)
	}
	if err := out.Close(); err != nil {
		fail(err)
	}
}

// fail says what went wrong and stops with a failing status.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "scalepfm:", err)
	os.Exit(1)
}
