// Accelerated search only. Every reported point is checked by noise.Fractal.
package main

import (
	"fmt"
	"sync"

	"github.com/janearc/slayer/maths/noise"
)

func mix(z uint64) uint64 {
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func search(sign int) {
	var corners [8]uint64
	var scores [8][12]int
	for corner := range 8 {
		dx, dy, dz := corner&1, (corner>>1)&1, (corner>>2)&1
		i, j, k := uint64(5016+dx), uint64(5016+dy), uint64(5016+dz)
		corners[corner] = mix(i + mix(j+mix(k+0x9e3779b97f4a7c15)))
		x, y, z := 1-2*dx, 1-2*dy, 1-2*dz
		gradients := [12]int{x + y, -x + y, x - y, -x - y, x + z, -x + z, x - z, -x - z, y + z, -y + z, y - z, -y - z}
		for h, v := range gradients {
			scores[corner][h] = sign * v / 2
		}
	}
	for seed := uint64(0); seed < 4_000_000_000; seed++ {
		deficit := 0
		for corner := range 8 {
			deficit += 1 - scores[corner][mix(seed^corners[corner])%12]
			if deficit > 1 {
				break
			}
		}
		if deficit > 1 {
			continue
		}
		const p = 5016.5
		actual := noise.Fractal(seed, p, p, p, 3)
		if float64(sign)*actual > .81 {
			fmt.Printf("FOUND sign=%d seed=%d p=%.17g,%.17g,%.17g noise=%.17g\n", sign, seed, p, p, p, actual)
			return
		}
	}
	fmt.Printf("NOT FOUND sign=%d after 4000000000 seeds; empirical, not proof\n", sign)
}

func main() {
	var jobs sync.WaitGroup
	for _, sign := range []int{-1, 1} {
		jobs.Add(1)
		go func() { defer jobs.Done(); search(sign) }()
	}
	jobs.Wait()
}
