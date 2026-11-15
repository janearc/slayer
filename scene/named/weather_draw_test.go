//go:build draw

package named_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/weather"
)

// gpuLevels is how far a map's colour on the gpu may be from the cpu's,
// in levels of 255, two hours in: picked by the author of this file,
// 2026-11-12, as float32 against float64 over two steps.
const gpuLevels = 2

// With no renderer said the weather goes on the gpu where there is one,
// and says which; asked for the cpu, it is on the cpu.
func TestTheGPUIsTheDefaultWhereThereIsOne(t *testing.T) {
	var said strings.Builder
	ask, err := named.AskMaps("", false, "", nothingSpaced, &said)
	if err != nil {
		t.Fatal(err)
	}
	mapper, err := weatherMaps(t).Build(tiny(), ask)
	if err != nil {
		t.Fatal(err)
	}
	want, words := named.CPU, "on the cpu"
	if weather.HasGPU() {
		want, words = named.GPU, "on the gpu"
	}
	if mapper.Renderer() != want || !strings.Contains(said.String(), words) {
		t.Errorf("unsaid: on %s, said %q", mapper.Renderer(), said.String())
	}
	if built(t, "").Renderer() != named.CPU {
		t.Error("the cpu asked for is not used")
	}
}

// On the gpu a moment is the cpu's to the float32's part in ten
// thousand of each map's colours; a machine with no device skips.
func TestTheGPUDrawsTheCPUsWeather(t *testing.T) {
	if !weather.HasGPU() {
		t.Skip(weather.ErrNoGPU)
	}
	ask, err := named.AskMaps(named.GPU, true, "", nothingSpaced, nil)
	if err != nil {
		t.Fatal(err)
	}
	mapper, err := weatherMaps(t).Build(tiny(), ask)
	if err != nil {
		t.Fatal(err)
	}
	gpu := at(t, mapper, twoHours)
	cpu := at(t, built(t, ""), twoHours)
	for i := range cpu {
		worst, differ := levelsApart(gpu[i].Picture().Pix,
			cpu[i].Picture().Pix)
		t.Logf("%s: %d of %d bytes differ, by %d levels at most",
			cpu[i].Name(), differ, len(cpu[i].Picture().Pix), worst)
		if gpu[i].Name() != cpu[i].Name() || worst > gpuLevels {
			t.Errorf("gpu reached %s, cpu %s, %d levels apart",
				gpu[i].Name(), cpu[i].Name(), worst)
		}
	}
}

// levelsApart is the most two pictures' bytes differ by, and how many
// differ.
func levelsApart(one, other []byte) (worst, differ int) {
	for k := range one {
		apart := int(one[k]) - int(other[k])
		if apart != 0 {
			differ++
		}
		worst = max(worst, apart, -apart)
	}
	return worst, differ
}

// refusing is a writer that takes nothing.
type refusing struct{}

// errRefusing is what refusing says.
var errRefusing = errors.New("the writer takes nothing")

// anti-test: a writer that cannot be written to stops the weather where
// it says what it did, rather than going on unsaid.
func TestAWriterThatRefusesStopsTheWeather(t *testing.T) {
	unsaid, err := named.AskMaps("", false, "", nothingSpaced, refusing{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := weatherMaps(t).Build(tiny(), unsaid); !errors.Is(err,
		errRefusing) {
		t.Errorf("unsaid renderer, refusing writer: %v", err)
	}
	said, err := named.AskMaps(named.CPU, true, "", nothingSpaced,
		refusing{})
	if err != nil {
		t.Fatal(err)
	}
	mapper, err := weatherMaps(t).Build(tiny(), said)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mapper.At(twoHours); !errors.Is(err, errRefusing) {
		t.Errorf("a moment said to a refusing writer: %v", err)
	}
}

// Write refuses.
func (refusing) Write([]byte) (int, error) { return none, errRefusing }
