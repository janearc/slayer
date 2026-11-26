package profilecheck

import "strings"

// Figure is an authored range and its provenance.
type Figure struct {
	Least, Most float64
	Basis       string
}

// Constant is an authored number, unit and provenance.
type Constant struct {
	Value       float64
	Unit, Basis string
}

// Based checks a choice's basis against the book's named sources.
func Based(sources map[string]string, basis string) bool {
	word, rest, _ := strings.Cut(basis, ":")
	switch word {
	case "picked", "derived", "recalled":
		return true
	case "cited":
		for name := range sources {
			if strings.Contains(rest, name) {
				return true
			}
		}
	}
	return false
}

// ColourShares checks three finite reflectance shares.
func ColourShares(channels []float64) bool {
	return len(channels) == 3 && channels[0] >= 0 && channels[0] <= 1 &&
		channels[1] >= 0 && channels[1] <= 1 && channels[2] >= 0 && channels[2] <= 1
}
