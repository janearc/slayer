package emit_test

import (
	_ "embed"
	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"math"
	"strings"
	"testing"
)

// testZero is defined as no light, length, angle or elapsed time.
const testZero = 0

//go:embed testdata/beam.yaml
var beamRaw []byte

// beamData is authored contract data. Expected power comes from the
// transverse Gaussian integral, independently of the beam implementation.
type beamData struct {
	Source        emit.Lit  `json:"source"`
	Radius        float64   `json:"radius"`
	Reach         float64   `json:"reach"`
	Slope         float64   `json:"slope"`
	Heights       []float64 `json:"heights"`
	Outside       []float64 `json:"outside"`
	GaussianRatio float64   `json:"gaussian-ratio" yaml:"gaussian-ratio"`
	Tolerance     float64   `json:"tolerance"`
	Quadrature    struct {
		Count     int     `json:"count"`
		Step      float64 `json:"step"`
		Midpoint  float64 `json:"midpoint"`
		Turn      float64 `json:"turn"`
		Expected  float64 `json:"expected"`
		Tolerance float64 `json:"tolerance"`
	} `json:"quadrature"`
	InvalidSources []emit.Lit `json:"invalid-sources" yaml:"invalid-sources"`
	InvalidLengths []float64  `json:"invalid-lengths" yaml:"invalid-lengths"`
	InvalidAngles  []float64  `json:"invalid-angles" yaml:"invalid-angles"`
	ValidAngles    []float64  `json:"valid-angles" yaml:"valid-angles"`
	Remote         maths.Vec  `json:"remote"`
	Line           struct {
		Wavelength float64 `json:"wavelength"`
		Strength   float64 `json:"strength"`
		Probe      float64 `json:"probe"`
	} `json:"line"`
	Tilt struct {
		Axis   maths.Vec `json:"axis"`
		Across maths.Vec `json:"across"`
		Length float64   `json:"length"`
	} `json:"tilt"`
	Overflow struct {
		Radius float64 `json:"radius"`
		Reach  float64 `json:"reach"`
		Angle  float64 `json:"angle"`
	} `json:"overflow"`
}

func readBeamData(test *testing.T) beamData {
	test.Helper()
	var data beamData
	if err := yamlfile.Read("testdata/beam.yaml", beamRaw, &data); err != nil {
		test.Fatal(err)
	}
	return data
}

func launchBeam(test *testing.T, data beamData, divergence float64) *emit.Beam {
	test.Helper()
	beam, err := emit.NewBeam(data.Source, measure.Metres(data.Radius), measure.Metres(data.Reach), measure.Radians(divergence))
	if err != nil {
		test.Fatal(err)
	}
	return beam
}

func TestBeamCollimationAndGaussianRadius(test *testing.T) {
	data := readBeamData(test)
	beam := launchBeam(test, data, testZero)
	for _, height := range data.Heights {
		at := beam.Point().At.Add(maths.YAxis.Scale(height))
		toward, distance, light := beam.Light(at, maths.XAxis)
		if toward != maths.YAxis.Neg() || distance != height || light != beam.Point().Strength {
			test.Fatal("collimated light changed with distance", toward, distance, light)
		}
		edgeAt := at.Add(maths.XAxis.Scale(data.Radius))
		_, _, edge := beam.Light(edgeAt, maths.XAxis)
		if math.Abs(edge.R/data.Source.Strength.R-data.GaussianRatio) > data.Tolerance {
			test.Fatal("radius is not the 1/e^2 intensity radius", edge)
		}
	}
}

func TestBeamEndsAndCopies(test *testing.T) {
	data := readBeamData(test)
	beam := launchBeam(test, data, testZero)
	for _, along := range data.Outside {
		at := beam.Point().At.Add(maths.YAxis.Scale(along))
		toward, distance, light := beam.Light(at, maths.YAxis)
		if toward != maths.V(testZero, testZero, testZero) || distance != testZero || light != maths.Grey(testZero) {
			test.Fatal("light escaped finite beam", along, toward, distance, light)
		}
	}
	copy := beam.Point()
	copy.At = maths.V(testZero, testZero, testZero)
	copy.Toward = maths.ZAxis
	if beam.Point().At == copy.At || beam.Point().Toward != maths.YAxis || beam.Radius().InMetres() != data.Radius || beam.Reach().InMetres() != data.Reach {
		test.Fatal("beam metadata or launch copy changed the source")
	}
}

// Separate transverse planes must carry the same analytically known power.
func TestDivergentBeamConservesTransversePower(test *testing.T) {
	data := readBeamData(test)
	beam := launchBeam(test, data, math.Atan(data.Slope))
	if math.Abs(beam.Slope()-data.Slope) > data.Tolerance {
		test.Fatal("wrong radius growth", beam.Slope())
	}
	for _, height := range data.Heights {
		integral := float64(testZero)
		for radial := range data.Quadrature.Count {
			radius := (float64(radial) + data.Quadrature.Midpoint) * data.Quadrature.Step
			at := beam.Point().At.Add(maths.V(radius, height, testZero))
			_, _, light := beam.Light(at, maths.YAxis)
			integral += light.R * data.Quadrature.Turn * radius * data.Quadrature.Step
		}
		if math.Abs(integral-data.Quadrature.Expected) > data.Quadrature.Tolerance {
			test.Fatal("divergent beam changed integrated transverse power", height, integral)
		}
	}
}

func TestBeamRefusesInvalidLaunches(test *testing.T) {
	data := readBeamData(test)
	for _, invalid := range data.InvalidSources {
		beam, err := emit.NewBeam(invalid, measure.Metres(data.Radius), measure.Metres(data.Reach), measure.Radians(testZero))
		if err == nil || beam != nil {
			test.Fatal("accepted invalid source", invalid, err)
		}
	}
	for _, value := range data.InvalidLengths {
		beam, err := emit.NewBeam(data.Source, measure.Metres(value), measure.Metres(data.Reach), measure.Radians(testZero))
		if err == nil || beam != nil {
			test.Fatal("accepted invalid radius", value, err)
		}
		beam, err = emit.NewBeam(data.Source, measure.Metres(data.Radius), measure.Metres(value), measure.Radians(testZero))
		if err == nil || beam != nil {
			test.Fatal("accepted invalid reach", value, err)
		}
	}
	for _, angle := range data.InvalidAngles {
		beam, err := emit.NewBeam(data.Source, measure.Metres(data.Radius), measure.Metres(data.Reach), measure.Radians(angle))
		if err == nil || beam != nil {
			test.Fatal("accepted invalid divergence", angle, err)
		}
	}
}

func TestBeamCarriesTheSignedRGBOfAMonochromaticSpectrum(test *testing.T) {
	data := readBeamData(test)
	spectrum := bands.Lines(map[float64]float64{data.Line.Wavelength: data.Line.Strength})
	point := emit.NewPoint(data.Source.At, maths.YAxis, spectrum)
	data.Source = point.When(testZero)
	beam := launchBeam(test, data, testZero)
	at := data.Source.At.Add(maths.YAxis.Scale(data.Line.Probe))
	_, _, light := beam.Light(at, maths.YAxis)
	if light != data.Source.Strength || light.Luminance() <= testZero {
		test.Fatal("lost monochromatic RGB", light)
	}
	var band maths.Band
	band.Nanometres = data.Line.Wavelength
	if beam.In(band) != data.Line.Strength || beam.Lands(at, band) != data.Line.Strength {
		test.Fatal("lost collimated spectral power")
	}
	edge := at.Add(maths.XAxis.Scale(data.Radius))
	want := data.Line.Strength * data.GaussianRatio
	if math.Abs(beam.Lands(edge, band)-want) > data.Tolerance {
		test.Fatal("spectral beam lost its transverse envelope")
	}
	for _, height := range data.Outside {
		outside := data.Source.At.Add(maths.YAxis.Scale(height))
		if beam.Lands(outside, band) != testZero {
			test.Fatal("spectral light escaped the finite beam")
		}
	}
}

func TestBeamProfileFollowsATiltedLaunch(test *testing.T) {
	data := readBeamData(test)
	data.Source.Toward = data.Tilt.Axis.Scale(data.Tilt.Length)
	beam := launchBeam(test, data, testZero)
	for _, height := range data.Heights {
		at := data.Source.At.Add(data.Tilt.Axis.Scale(height))
		toward, distance, light := beam.Light(at, maths.ZAxis)
		if toward.Add(data.Tilt.Axis).Len() > data.Tolerance ||
			math.Abs(distance-height) > data.Tolerance || light != data.Source.Strength {
			test.Fatal("tilted launch did not follow its axis", toward, distance, light)
		}
		edge := at.Add(data.Tilt.Across.Scale(data.Radius))
		_, _, light = beam.Light(edge, maths.ZAxis)
		if math.Abs(light.R/data.Source.Strength.R-data.GaussianRatio) > data.Tolerance {
			test.Fatal("tilted transverse radius changed", light)
		}
	}
}

func TestBeamRefusesAnOverflowingFiniteEnvelope(test *testing.T) {
	data := readBeamData(test)
	bad := data.Overflow
	radius, reach := measure.Metres(bad.Radius), measure.Metres(bad.Reach)
	if math.IsInf(radius.InMetres(), testZero) ||
		math.IsInf(reach.InMetres(), testZero) {
		test.Fatal("fixture overflowed in unit conversion")
	}
	beam, err := emit.NewBeam(data.Source, radius, reach,
		measure.Radians(bad.Angle))
	if err == nil || beam != nil ||
		!strings.Contains(err.Error(), "non-finite beam envelope") {
		test.Fatal("accepted overflowing extent", err)
	}
}

func TestBeamAcceptsDarkSourcesAndEveryValidDivergence(test *testing.T) {
	data := readBeamData(test)
	data.Source.Strength = maths.Grey(testZero)
	for _, angle := range data.ValidAngles {
		beam := launchBeam(test, data, angle)
		at := data.Source.At.Add(maths.YAxis.Scale(data.Line.Probe))
		_, _, light := beam.Light(at, maths.YAxis)
		if light != maths.Grey(testZero) {
			test.Fatal("dark beam emitted light", light)
		}
	}
}

func TestBeamIsDarkAtAnUnresolvablyDistantTransversePoint(test *testing.T) {
	data := readBeamData(test)
	beam := launchBeam(test, data, testZero)
	toward, distance, light := beam.Light(data.Remote, maths.YAxis)
	if toward != maths.V(testZero, testZero, testZero) ||
		distance != testZero || light != maths.Grey(testZero) {
		test.Fatal("remote point did not remain dark", toward, distance, light)
	}
}
