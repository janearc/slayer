package mocks_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/mocks"
	"github.com/janearc/slayer/ninjafilm/record"
	"github.com/janearc/slayer/ninjafilm/roll"
	"go.uber.org/mock/gomock"
)

// once is a call that happens exactly once, defined as one.
const once = 1

// none is no amount at all, defined as nought.
const none = 0

// The mocks are made by mockgen, and the packages whose tests lean on
// them test those packages. These tests hold each mock to what it is
// told: it answers each call as its expectation says, and it fails the
// test for a call nobody expected.
func TestAMockLayAnswersAsItIsTold(t *testing.T) {
	ctrl := gomock.NewController(t)
	laid := mocks.NewMockLay(ctrl)
	across := data.Whole("roll-across-most")
	laid.EXPECT().Across(gomock.Any()).Return(across).Times(once)
	var records []record.Record
	filter := maths.Grey(data.Picked("roll-bright-most"))
	laid.EXPECT().Site(records, gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any()).Return(records, filter).Times(once)
	if got := laid.Across(data.Picked("roll-chip-least")); got != across {
		t.Errorf("the mock lay put %d sites across, told %d", got, across)
	}
	var seed uint64
	var x, y, frame int64
	kept, light := laid.Site(records, seed, x, y, frame)
	if len(kept) != len(records) || light != filter {
		t.Errorf("the mock lay gave %v and %v, told %v", kept, light,
			filter)
	}
}

func TestAMockShelfAnswersAsItIsTold(t *testing.T) {
	ctrl := gomock.NewController(t)
	shelf := mocks.NewMockShelf(ctrl)
	var stock roll.Stock
	refusal := errors.New("no such stock")
	shelf.EXPECT().Find("any").Return(stock, refusal).Times(once)
	got, err := shelf.Find("any")
	if !reflect.DeepEqual(got, stock) || !errors.Is(err, refusal) {
		t.Errorf("the mock shelf gave %v, %v, told %v", got, err, refusal)
	}
}

func TestAMockProjectionAnswersAsItIsTold(t *testing.T) {
	ctrl := gomock.NewController(t)
	glass := mocks.NewMockProjection(ctrl)
	band := maths.Band{Nanometres: data.Picked("film-band-most")}
	told := maths.Ray{Dir: maths.V(data.Picked("film-wide-least"),
		data.Picked("film-wide-least"), -data.Picked("film-wide-most"))}
	glass.EXPECT().Ray(gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), band).Return(told).Times(once)
	var x, y, across, down int
	if got := glass.Ray(x, y, across, down, band); got != told {
		t.Errorf("the mock projection gave %v, told %v", got, told)
	}
}

func TestAMockEmitterAnswersAsItIsTold(t *testing.T) {
	ctrl := gomock.NewController(t)
	lamp := mocks.NewMockEmitter(ctrl)
	toward := maths.V(data.Picked("film-wide-least"), none, none)
	dist := data.Picked("film-wide-most")
	light := maths.Grey(data.Picked("film-strength-most"))
	lamp.EXPECT().Light(gomock.Any(), gomock.Any()).Return(toward, dist,
		light).Times(once)
	gotToward, gotDist, gotLight := lamp.Light(maths.Vec{}, maths.Up)
	if gotToward != toward || gotDist != dist || gotLight != light {
		t.Errorf("the mock emitter gave %v, %v, %v", gotToward, gotDist,
			gotLight)
	}
}

func TestAMockBeamerAnswersAsItIsTold(t *testing.T) {
	ctrl := gomock.NewController(t)
	beam := mocks.NewMockBeamer(ctrl)
	band := maths.Band{Nanometres: data.Picked("film-band-most")}
	strength := data.Picked("film-strength-most")
	beam.EXPECT().Beams(band).Return(strength).Times(once)
	beam.EXPECT().Light(gomock.Any(), gomock.Any()).Return(maths.Vec{},
		strength, maths.Light{}).Times(once)
	if got := beam.Beams(band); got != strength {
		t.Errorf("the mock beamer beams %v, told %v", got, strength)
	}
	if _, got, _ := beam.Light(maths.Vec{}, maths.Up); got != strength {
		t.Errorf("the mock beamer is %v away, told %v", got, strength)
	}
}
