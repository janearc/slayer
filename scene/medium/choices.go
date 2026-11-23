package medium

import (
	_ "embed"
	"fmt"

	"github.com/janearc/puzzlemouth/numbers"
)

//go:embed choices.yaml
var choicesRaw []byte

// shapeChoices holds the numbers the media's shapes are drawn with, which are
// chosen for the picture, after Puzzlemouth reads them. Density queries do
// not parse data or search the numeric catalogue.
type shapeChoices struct {
	skirtWidths, streamBell, streamEighth       float64
	jetLowerSide, jetKnotSpacing, jetKnotWeight float64
	alongTheAxis                                float64
	jetKnotOctaves, probesLeast, probesMost     int

	cloudFrequency, cloudEdge, cloudWhole, cloudTop float64
	cloudOctaves                                    int

	auroraFloor, auroraFall, auroraSoft, auroraRays float64
	auroraFolds, auroraStretch, auroraRayWeight     float64
	auroraOctaves                                   int
	auroraFoldSeed                                  uint64

	plumeCore, plumeThin, plumeSkin, plumePeak float64
	plumeFoot, plumeRise, plumeSkirt           float64

	fizzGrain, fizzSpread          float64
	fizzOctaves, wispyOctaves      int
	wakeFade, wakeSwept, wakeRidge float64
	wakeBank, wakeLift             float64
	exponentMost, clearCloudWidth  float64
	wavelengthRed, wavelengthGreen float64
	wavelengthBlue                 float64
}

// chosen is the shapes' choices, read once when the package starts.
var chosen = readShapeChoices()

// choiceFile is choices.yaml, read once when the package starts. A package
// variable cannot return an error, so a file or row the loader refuses stops
// the program at start with words that name the file and the row.
var choiceFile = readChoiceFile()

// readChoiceFile reads the embedded file; every row it refuses is named.
func readChoiceFile() numbers.Numbers {
	read, err := numbers.Read("choices.yaml", choicesRaw)
	if err != nil {
		panic(fmt.Errorf("medium: choices.yaml refused, row by row: %w", err))
	}
	return read
}

// choice is the named choice. A name the file does not hold is refused in
// words that name the row.
func choice(name string) float64 {
	found, err := choiceFile.Get(name)
	if err != nil {
		panic(fmt.Errorf("medium: choices.yaml has no usable row %q: %w",
			name, err))
	}
	return found.Value
}

// choiceSeed is the named choice, which is a whole number added to a seed.
func choiceSeed(name string) uint64 { return uint64(choice(name)) }

// choiceCount is the named choice, which is a whole number.
func choiceCount(name string) int { return int(choice(name)) }

// readShapeChoices reads the shapes' choices.
func readShapeChoices() shapeChoices {
	pick, count := choice, choiceCount
	return shapeChoices{
		skirtWidths: pick("skirt-widths"), streamBell: pick("stream-bell"),
		streamEighth: pick("stream-eighth"), jetLowerSide: pick("jet-lower-side"),
		jetKnotSpacing: pick("jet-knot-spacing"),
		jetKnotWeight:  pick("jet-knot-weight"),
		alongTheAxis:   pick("along-the-axis"),
		jetKnotOctaves: count("jet-knot-octaves"),
		probesLeast:    count("probes-least"), probesMost: count("probes-most"),
		cloudFrequency: pick("cloud-frequency"), cloudEdge: pick("cloud-edge"),
		cloudWhole: pick("cloud-whole"), cloudTop: pick("cloud-top"),
		cloudOctaves: count("cloud-octaves"),
		auroraFloor:  pick("aurora-floor"), auroraFall: pick("aurora-fall"),
		auroraSoft: pick("aurora-soft"), auroraRays: pick("aurora-rays"),
		auroraFolds: pick("aurora-folds"), auroraStretch: pick("aurora-stretch"),
		auroraRayWeight: pick("aurora-ray-weight"),
		auroraOctaves:   count("aurora-octaves"),
		auroraFoldSeed:  uint64(count("aurora-fold-seed")),
		plumeCore:       pick("plume-core"), plumeThin: pick("plume-thin"),
		plumeSkin: pick("plume-skin"), plumePeak: pick("plume-peak"),
		plumeFoot: pick("plume-foot"), plumeRise: pick("plume-rise"),
		plumeSkirt: pick("plume-skirt"), fizzGrain: pick("fizz-grain"),
		fizzSpread:  pick("fizz-spread"),
		fizzOctaves: count("fizz-octaves"), wispyOctaves: count("wispy-octaves"),
		wakeFade: pick("wake-fade"), wakeSwept: pick("wake-swept"),
		wakeRidge: pick("wake-ridge"), wakeBank: pick("wake-bank"),
		wakeLift: pick("wake-lift"), exponentMost: pick("exponent-most"),
		clearCloudWidth: pick("clear-cloud-width"),
		wavelengthRed:   pick("wavelength-red"),
		wavelengthGreen: pick("wavelength-green"),
		wavelengthBlue:  pick("wavelength-blue"),
	}
}
