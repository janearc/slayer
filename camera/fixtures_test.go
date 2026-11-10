package camera_test

import (
	_ "embed"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

// nothing is defined as zero items, time or displacement.
const nothing = 0

// whole is defined as one item, second or positive infinity sign.
const whole = 1

//go:embed testdata/paths.yaml
var fixtureRaw []byte

//go:embed testdata/paths.schema.json
var fixtureSchema []byte

// pathFixture is the checked test-data wire format, separate from camera.Move.
type pathFixture struct {
	Numbers map[string]float64 `json:"numbers"`
	Vectors map[string]struct {
		East  float64 `json:"x" yaml:"x"`
		Up    float64 `json:"y" yaml:"y"`
		South float64 `json:"z" yaml:"z"`
	} `json:"vectors"`
	Samples map[string][]struct {
		Seconds float64 `json:"seconds"`
		Point   string  `json:"point"`
	} `json:"samples"`
}

// fixture loads the hand-worked expectations through Puzzlemouth.
func fixture(test *testing.T) *pathFixture {
	test.Helper()
	schema, err := yamlfile.NewSchema("https://slayer.test/camera-paths", fixtureSchema)
	if err != nil {
		test.Fatal(err)
	}
	node, err := schema.Read("testdata/paths.yaml", fixtureRaw)
	if err != nil {
		test.Fatal(err)
	}
	var data pathFixture
	if err := node.Decode(&data); err != nil {
		test.Fatal(err)
	}
	return &data
}

// number fetches a named fixture choice, refusing a misspelling.
func (data *pathFixture) number(test *testing.T, name string) float64 {
	test.Helper()
	value, exists := data.Numbers[name]
	if !exists {
		test.Fatalf("no fixture number %q", name)
	}
	return value
}

// duration expresses a chosen number of seconds in the sampler clock.
func (data *pathFixture) duration(test *testing.T, name string) time.Duration {
	test.Helper()
	seconds := data.number(test, name)
	return time.Duration(seconds * float64(time.Second))
}

// vector fetches a named fixture point, refusing a misspelling.
func (data *pathFixture) vector(test *testing.T, name string) maths.Vec {
	test.Helper()
	value, exists := data.Vectors[name]
	if !exists {
		test.Fatalf("no fixture vector %q", name)
	}
	return maths.V(value.East, value.Up, value.South)
}
