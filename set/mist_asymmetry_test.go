package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// mistSeed is the laser fixture's grove mist's last line, where an
// asymmetry is written after it.
const mistSeed = "      seed: 23\n"

// lobedSource is each laser fixture, with and without its schema, its
// grove mist given an asymmetry line, or left as it is for none.
func lobedSource(test *testing.T, line string) []string {
	test.Helper()
	var sources []string
	for _, raw := range laserSources() {
		if !strings.Contains(raw, mistSeed) {
			test.Fatal("the laser fixture's grove mist has moved")
		}
		sources = append(sources,
			strings.Replace(raw, mistSeed, mistSeed+line, 1))
	}
	return sources
}

// TestAMistSaysItsAsymmetry: a mist's asymmetry is read, through the
// schema and without it; a mist that does not say one scatters evenly.
func TestAMistSaysItsAsymmetry(test *testing.T) {
	for want, line := range map[float64]string{
		0.85: "      asymmetry: 0.85\n", -0.3: "      asymmetry: -0.3\n",
		0: ""} {
		for _, source := range lobedSource(test, line) {
			made, err := set.Read("lobed.yaml", []byte(source))
			if err != nil {
				test.Fatal(err)
			}
			mist, found := made.Mist("grove mist")
			if !found || mist.Asymmetry() != want {
				test.Errorf("%q: asymmetry %v, want %v", line,
					mist.Asymmetry(), want)
			}
		}
	}
}

// TestAMistsLobeIsRefusedPastOne: an asymmetry at or past the lobe's
// edge either way, or not a number, is refused; one just inside is not.
func TestAMistsLobeIsRefusedPastOne(test *testing.T) {
	for _, line := range []string{"      asymmetry: 1\n",
		"      asymmetry: -1\n", "      asymmetry: 0.995\n",
		"      asymmetry: forward\n"} {
		for _, source := range lobedSource(test, line) {
			if _, err := set.Read("lobed.yaml", []byte(source)); err == nil {
				test.Errorf("%q was taken", line)
			}
		}
	}
	for _, source := range lobedSource(test, "      asymmetry: 0.99\n") {
		if _, err := set.Read("lobed.yaml", []byte(source)); err != nil {
			test.Errorf("the lobe's edge was refused: %v", err)
		}
	}
}
