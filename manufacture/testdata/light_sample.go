// Record authored lamp positions, spectra, beams and flicker.
package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

type lightRecord struct {
	Kind                         string
	Seed                         uint64
	Pose, Index                  int
	At, Toward                   maths.Vec
	Beyond, Strength, Half, Soft float64
	Shares                       []float64
	First, Last                  int
	Power                        []float64
}

func main() {
	poses := []maths.Subspace{{},
		{Origin: maths.V(3, -2, 7), Scale: .5, Orientation: maths.Rotation(maths.V(1, 2, 3), .7)},
		{Origin: maths.V(-4, 5, -6), Scale: 3, Orientation: maths.Rotation(maths.V(2, -1, 1), -.4)}}
	var rows []lightRecord
	for _, name := range []string{"campfire", "bonfire", "spotlight", "maser"} {
		kind, err := manufacture.KindNamed(name)
		if err != nil {
			panic(err)
		}
		for _, seed := range []uint64{5, 11} {
			made, err := manufacture.Make(kind, manufacture.Sow(seed))
			if err != nil {
				panic(err)
			}
			for pose, space := range poses {
				lights, err := made.Lights(space)
				if err != nil {
					panic(err)
				}
				for i, source := range lights {
					light := source.Light()
					half, soft := light.Angles()
					row := lightRecord{Kind: name, Seed: seed, Pose: pose, Index: i,
						At: source.At(), Toward: source.Toward(), Beyond: source.Beyond().InMetres(),
						Strength: light.Strength(), Half: half.InRadians(), Soft: soft.InRadians()}
					for _, at := range []time.Duration{0, time.Second / 3, time.Second, 7 * time.Second, 13 * time.Second} {
						row.Shares = append(row.Shares, light.Share(at))
					}
					spectrum := light.Bands()
					first, last := spectrum.Span()
					row.First, row.Last = int(first), int(last)
					for band := first; band <= last; band++ {
						row.Power = append(row.Power, spectrum.Power(band))
					}
					rows = append(rows, row)
				}
			}
		}
	}
	out := json.NewEncoder(os.Stdout)
	out.SetIndent("", "  ")
	if err := out.Encode(rows); err != nil {
		panic(err)
	}
}
