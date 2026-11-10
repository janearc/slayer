package manufacture_test

import (
	_ "embed"
	"math"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

// firstDescription and positiveInfinity are defined slice/sign identities.
const firstDescription, positiveInfinity = 0, 1

//go:embed testdata/light-placement.yaml
var lightPlacementRaw []byte

type lightPlacementContract struct {
	Seed                   uint64
	Kinds                  []string
	Shift                  maths.Vec
	Scale, Turn, Tolerance float64
}

func TestRetainedDescriptionsPlaceWithoutMutatingTheLocalLights(test *testing.T) {
	var data lightPlacementContract
	if err := yamlfile.Read("testdata/light-placement.yaml", lightPlacementRaw, &data); err != nil {
		test.Fatal(err)
	}
	orientation := maths.About(maths.YAxis, data.Turn*maths.Degree)
	pose := maths.NewSubspace(data.Shift, orientation, data.Scale)
	for _, name := range data.Kinds {
		row, err := manufacture.KindNamed(name)
		if err != nil {
			test.Fatal(err)
		}
		object, err := manufacture.Make(row, manufacture.Sow(data.Seed))
		if err != nil {
			test.Fatal(err)
		}
		local := object.Descriptions()
		if len(local) == firstDescription {
			test.Fatal("missing local description", name)
		}
		before := local[firstDescription].At()
		placed, err := manufacture.PlaceLights(local, pose)
		if err != nil {
			test.Fatal(err)
		}
		if local[firstDescription].At() != before {
			test.Fatal("placement mutated the local cache", name)
		}
		data.assertPlaced(test, local, placed)
		original := local[firstDescription]
		local[firstDescription] = manufacture.AttachedLight{}
		retained := object.Descriptions()[firstDescription]
		if retained.At() != original.At() ||
			retained.Light().Strength() != original.Light().Strength() {
			test.Fatal("description copy aliases the specimen", name)
		}
	}
}

func (data lightPlacementContract) assertPlaced(test *testing.T,
	local, placed []manufacture.AttachedLight,
) {
	test.Helper()
	if len(local) != len(placed) {
		test.Fatal("lost attached light")
	}
	for index, source := range local {
		at, way := source.At(), source.Toward()
		expected := maths.V(data.Shift.X+data.Scale*at.Z,
			data.Shift.Y+data.Scale*at.Y, data.Shift.Z-data.Scale*at.X)
		direction := maths.V(way.Z, way.Y, -way.X)
		actual := placed[index]
		if actual.At().Sub(expected).Len() > data.Tolerance ||
			actual.Toward().Sub(direction).Len() > data.Tolerance ||
			math.Abs(actual.Beyond().InMetres()-source.Beyond().InMetres()*data.Scale) > data.Tolerance ||
			actual.Light().Strength() != source.Light().Strength() {
			test.Fatal("retained light lost the declared quarter-turn, scale or source", actual, source)
		}
	}
}

func TestEmptyLightPlacementStillRefusesInvalidFrames(test *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(positiveInfinity)} {
		origin := maths.V(value, spacecraftNear, spacecraftNear)
		pose := maths.NewSubspace(origin, maths.Identity, positiveInfinity)
		if _, err := manufacture.PlaceLights(nil, pose); err == nil {
			test.Fatal("empty light list accepted an invalid frame", value)
		}
	}
}
