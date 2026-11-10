package set_test

import (
	_ "embed"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

//go:embed testdata/shared-laser-pointings.yaml
var sharedLaserRaw []byte

type sharedLaserCase struct {
	Name                                    string
	Azimuth, Altitude, Horizontal, Vertical float64
	ExpectedAzimuth                         float64 `yaml:"expected-azimuth"`
	ExpectedAltitude                        float64 `yaml:"expected-altitude"`
	Separation                              float64 `yaml:"separation-arcsec"`
}

type sharedLaserContract struct {
	Tolerance        float64 `yaml:"tolerance-deg"`
	AngularTolerance float64 `yaml:"tolerance-arcsec"`
	Cases            []sharedLaserCase
	Fields           []sharedLaserCase `yaml:"field-checks"`
	Invalid          []struct{ Before, After string }
}

func sharedLaserData(test *testing.T) sharedLaserContract {
	test.Helper()
	var data sharedLaserContract
	if err := yamlfile.Read("shared-laser-pointings.yaml",
		sharedLaserRaw, &data); err != nil {
		test.Fatal(err)
	}
	return data
}

// source declares the named direction after the launch to hold forward lookup.
func (data sharedLaserCase) source(test *testing.T, raw string) string {
	test.Helper()
	pointing := "pointing: {azimuth: 30, altitude: 70}"
	if !strings.Contains(raw, pointing) {
		test.Fatal("base laser fixture no longer has expected pointing")
	}
	replacement := fmt.Sprintf("pointing: shared science\n"+
		"      guide-star-offset:\n"+
		"        horizontal: {value: %g, unit: arcsec}\n"+
		"        vertical: {value: %g, unit: arcsec}",
		data.Horizontal, data.Vertical)
	raw = strings.Replace(raw, pointing, replacement, singleLaser)
	return raw + fmt.Sprintf("\n  laser-pointings:\n"+
		"    shared science: {azimuth: %g, altitude: %g}\n",
		data.Azimuth, data.Altitude)
}

type sharedLaserProbe struct {
	row       sharedLaserCase
	tolerance float64
}

func TestSharedLaserPointingKeepsOneAxisAndUsesArcseconds(test *testing.T) {
	data := sharedLaserData(test)
	for _, row := range data.Cases {
		probe := sharedLaserProbe{row: row, tolerance: data.Tolerance}
		test.Run(row.Name, probe.checkDirection)
	}
}

// checkDirection holds absolute angular fixtures through both readers.
func (probe sharedLaserProbe) checkDirection(test *testing.T) {
	test.Helper()
	for _, raw := range laserSources() {
		source := probe.row.source(test, raw)
		read, err := set.Read("shared-laser.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		azimuth, altitude := read.Lasers()[firstLaser].Pointing()
		if math.Abs(azimuth-probe.row.ExpectedAzimuth) > probe.tolerance ||
			math.Abs(altitude-probe.row.ExpectedAltitude) > probe.tolerance {
			test.Fatal("shared direction or angular unit changed",
				azimuth, altitude)
		}
	}
}

func TestSharedLaserOffsetsKeepTheirSkyAngleAtZenith(test *testing.T) {
	data := sharedLaserData(test)
	for _, row := range data.Fields {
		probe := sharedLaserProbe{row: row, tolerance: data.AngularTolerance}
		test.Run(row.Name, probe.checkSeparation)
	}
}

// checkSeparation measures a great-circle angle, not an azimuth difference.
func (probe sharedLaserProbe) checkSeparation(test *testing.T) {
	test.Helper()
	for _, raw := range laserSources() {
		source := probe.row.source(test, raw)
		read, err := set.Read("shared-laser.yaml", []byte(source))
		if err != nil {
			test.Fatal(err)
		}
		azimuth, altitude := read.Lasers()[firstLaser].Pointing()
		before := laserSkyDirection(probe.row.Azimuth, probe.row.Altitude)
		after := laserSkyDirection(azimuth, altitude)
		dot := before.Dot(after)
		dot = min(singleLaser, max(-singleLaser, dot))
		separation := math.Acos(dot) / maths.Degree * arcsecondsInDegree
		if math.Abs(separation-probe.row.Separation) > probe.tolerance {
			test.Fatal("offset changed with boresight elevation", separation)
		}
	}
}

// arcsecondsInDegree is defined by sixty arcminutes of sixty arcseconds.
const arcsecondsInDegree = 3600

// laserSkyDirection converts only the final angles, independent of offset math.
func laserSkyDirection(azimuth, altitude float64) maths.Vec {
	azimuth, altitude = azimuth*maths.Degree, altitude*maths.Degree
	return maths.V(math.Sin(azimuth)*math.Cos(altitude),
		math.Sin(altitude), -math.Cos(azimuth)*math.Cos(altitude))
}

func TestSharedLaserPointingRefusesBrokenNamesUnitsAndFields(test *testing.T) {
	data := sharedLaserData(test)
	row := data.Cases[singleLaser]
	for _, bad := range data.Invalid {
		for _, raw := range laserSources() {
			valid := row.source(test, raw)
			broken := strings.ReplaceAll(valid, bad.Before, bad.After)
			if broken == valid {
				test.Fatal("refusal fixture did not apply", bad)
			}
			if _, err := set.Read("bad-shared-laser.yaml", []byte(broken)); err == nil {
				test.Fatal("accepted broken shared pointing", bad)
			}
		}
	}
}
