package main

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/improv/cascades/flatland"
)

// falloffDistances are the distances from the lone source, as fractions of
// the frame side, where the falloff is printed. Picked: near, middle, far.
var falloffDistances = []float64{0.0625, 0.125, 0.25, 0.4}

// printFalloff compares the fields with the exact answer for a lone disc
// in empty space: the disc subtends 2*asin(radius/distance) at distance,
// so J = brightness * that angle / (2 pi). Cells are read along a row
// through the disc's centre, in the green band.
// The fields are, in order, the reference, the cascades and a third panel.
func printFalloff(side int, fields []flatland.Field) {
	radius := math.Max(lone.radius*float64(side), minimumRadius)
	row := side / 2
	for _, fraction := range falloffDistances {
		column := side/2 + int(fraction*float64(side))
		distance := float64(column) + 0.5 - lone.centreX*float64(side)
		exact := lone.brightness[1] * 2 * math.Asin(radius/distance) / (2 * math.Pi)
		line := fmt.Sprintf("falloff at %.1f cells: exact %.5f", distance, exact)
		for index, name := range []string{"reference", "cascades", "third"} {
			line += fmt.Sprintf(", %s %.5f", name, fields[index].At(column, row)[1])
		}
		fmt.Println(line)
	}
}
