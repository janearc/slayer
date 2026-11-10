package set_test

import (
	_ "embed"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/set"
	"strings"
	"testing"
)

// firstLaser is defined as the first position in a Go slice.
const firstLaser = 0

// singleLaser is defined as one launch in this fixture.
const singleLaser = 1

//go:embed testdata/laser.yaml
var laserFixture []byte

//go:embed testdata/laser-contract.yaml
var laserContractRaw []byte

// laserContract keeps independent expected values and input refusals in data.
type laserContract struct {
	Name       string  `json:"name"`
	Object     string  `json:"object"`
	East       float64 `json:"east"`
	Up         float64 `json:"up"`
	South      float64 `json:"south"`
	Azimuth    float64 `json:"azimuth"`
	Altitude   float64 `json:"altitude"`
	Radius     float64 `json:"radius"`
	Reach      float64 `json:"reach"`
	Divergence float64 `json:"divergence"`
	Wavelength float64 `json:"wavelength"`
	Strength   float64 `json:"strength"`
	Invalid    []struct {
		Before string `json:"before"`
		After  string `json:"after"`
	} `json:"invalid"`
}

func readLaserContract(test *testing.T) laserContract {
	test.Helper()
	var contract laserContract
	if err := yamlfile.Read("testdata/laser-contract.yaml", laserContractRaw, &contract); err != nil {
		test.Fatal(err)
	}
	return contract
}

// laserSources exercises both the opt-in schema and the legacy domain reader.
func laserSources() []string {
	raw := string(laserFixture)
	legacy := strings.ReplaceAll(raw, "$schema: "+set.EcosystemSchema+"\n", "")
	return []string{raw, legacy}
}

func TestLaserSchemaAndReaderAcceptTheSamePhysicalControls(test *testing.T) {
	contract := readLaserContract(test)
	for _, source := range laserSources() {
		made, err := set.Read("laser.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		lasers := made.Lasers()
		if len(lasers) != singleLaser || lasers[firstLaser].Name() != contract.Name {
			test.Fatal("lost laser", lasers)
		}
		contract.assertLaunch(test, lasers[firstLaser])
		var empty set.Laser
		lasers[firstLaser] = empty
		if made.Lasers()[firstLaser].Name() != contract.Name {
			test.Fatal("caller changed retained launches")
		}
	}
}

func (want laserContract) assertLaunch(test *testing.T, laser set.Laser) {
	test.Helper()
	object, x, y, z := laser.Origin()
	azimuth, altitude := laser.Pointing()
	radius, reach, divergence := laser.Envelope()
	wavelength, strength := laser.Spectrum()
	if object != want.Object || x != want.East || y != want.Up || z != want.South || azimuth != want.Azimuth || altitude != want.Altitude || radius != want.Radius || reach != want.Reach || divergence != want.Divergence || wavelength != want.Wavelength || strength != want.Strength {
		test.Fatal("laser controls changed", laser)
	}
}

func TestLaserRefusesBadUnitsMissingCoordinatesAndUnknownOrigins(test *testing.T) {
	contract := readLaserContract(test)
	for _, replacement := range contract.Invalid {
		for _, raw := range laserSources() {
			source := strings.ReplaceAll(raw, replacement.Before, replacement.After)
			if source == raw {
				test.Fatal("invalid fixture replacement")
			}
			if _, err := set.Read("bad-laser.yaml", []byte(source)); err == nil {
				test.Fatal("accepted invalid laser", replacement)
			}
		}
	}
}
