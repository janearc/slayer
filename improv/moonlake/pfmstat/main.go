// Command pfmstat says a pfm's largest value, its mean, and the mean of
// each tenth of its rows from the top, so a dark raw says where its light is.
//
//	pfmstat in.pfm
package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
)

// main reads the pfm and says where its light is.
func main() {
	if len(os.Args) != 2 {
		fail(errors.New("pfmstat in.pfm"))
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		fail(err)
	}
	defer file.Close()
	in := bufio.NewReader(file)
	var kind string
	var width, height int
	var scale float64
	if _, err := fmt.Fscan(in, &kind, &width, &height, &scale); err != nil {
		fail(err)
	}
	in.ReadByte()
	channels := 3
	switch kind {
	case "PF":
	case "Pf":
		channels = 1
	default:
		fail(fmt.Errorf("%s is %s, not a pfm", os.Args[1], kind))
	}
	var order binary.ByteOrder = binary.LittleEndian
	if scale > 0 {
		order = binary.BigEndian
	}
	values := make([]float32, width*height*channels)
	if err := binary.Read(in, order, values); err != nil {
		fail(err)
	}
	most, sum := 0.0, 0.0
	bands := make([]float64, 10)
	for row := 0; row < height; row++ {
		for i := 0; i < width*channels; i++ {
			value := float64(values[row*width*channels+i])
			most = math.Max(most, value)
			sum += value
			// pfm rows run bottom to top
			bands[(height-1-row)*10/height] += value
		}
	}
	fmt.Printf("%dx%d max %.4g mean %.4g\n", width, height, most,
		sum/float64(len(values)))
	for band, total := range bands {
		fmt.Printf("tenth %d from top: mean %.4g\n", band,
			total/float64(width*channels*height/10))
	}
}

// fail says what went wrong on stderr and stops with a failing status.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "pfmstat:", err)
	os.Exit(1)
}
