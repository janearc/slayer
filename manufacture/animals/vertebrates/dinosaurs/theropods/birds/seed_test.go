package birds

import "github.com/janearc/slayer/maths/noise"

// This test seed reproduces the maker's deterministic seed stream without
// making the family package depend on the root manufacture package.
type seed uint64

const picking = 0

func sow(number uint64) seed { return seed(number) }
func (from seed) draw(a, b, c int) float64 {
	z := uint64(from) + noise.Golden*uint64(a+1) +
		0xbf58476d1ce4e5b9*uint64(b+1) + 0x94d049bb133111eb*uint64(c+1)
	return noise.Stirred(z)
}
func (from seed) Draw(a, b, c int) float64  { return from.draw(a, b, c) }
func (from seed) Swing(a, b, c int) float64 { return 2*from.draw(a, b, c) - 1 }

// birdVertebraOf is the region of a vertebra's name, T, Ca or C, and
// whether it is one.
func birdVertebraOf(name string) (string, bool) {
	for _, region := range []string{"Ca", "C", "T"} {
		rest := name[min(len(region), len(name)):]
		if len(name) > len(region) && name[:len(region)] == region &&
			rest[0] >= '0' && rest[0] <= '9' {
			return region, true
		}
	}
	return "", false
}
