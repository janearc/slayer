package cues

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"go.uber.org/mock/gomock"
)

const reuseScore = `score:
  tracks:
    lead:
      - {sec: 0.75, at_sec: 1, vel: 80}
      - {at_sec: 1.125, vel: 90}
      - {at_sec: 1.25, vel: 100}
  beats:
    times: [0, 0.5, 1, 1.5, 2]
`

func parsed(t *testing.T, sheet string) *Show {
	t.Helper()
	show, err := Parse([]byte(reuseScore), []byte(sheet))
	if err != nil {
		t.Fatal(err)
	}
	return show
}

func frameAt(t *testing.T, show *Show, seconds, fps float64) Frame {
	t.Helper()
	frame, err := show.At(seconds, fps)
	if err != nil {
		t.Fatal(err)
	}
	return frame
}

func TestExplicitFPSAndHalfOpenLeadWindow(t *testing.T) {
	show := parsed(t, `cues: [{at_sec: 0, strobes: lead}]`)
	for _, each := range []struct {
		seconds, fps   float64
		white, magenta int
	}{
		{1, 8, 1, 0},
		{1, 4, 1, 1},
		{1.125, 8, 0, 1},
		{0.75, 8, 0, 0},
		{1, 0, 1, 1},
		{1, -8, 0, 0},
	} {
		got := frameAt(t, show, each.seconds, each.fps)
		if got.White != each.white || got.Magenta != each.magenta {
			t.Errorf("At(%v,%v): white=%d magenta=%d, want %d %d", each.seconds, each.fps, got.White, got.Magenta, each.white, each.magenta)
		}
	}
}

func TestReuseIsIndependentOfOrderInputsEnvironmentAndOtherShows(t *testing.T) {
	scoreBytes := []byte(reuseScore)
	cueBytes := []byte(`cues: [{at_sec: 0, rig: 0.123456, fire: 1, figures: [wave], banks: [offset]}, {at_sec: 1, rig: 1, fade: 2}]`)
	show, err := Parse(scoreBytes, cueBytes)
	if err != nil {
		t.Fatal(err)
	}
	clear(scoreBytes)
	clear(cueBytes)
	want := frameAt(t, show, 0.25, 24)
	other := parsed(t, `cues: [{at_sec: 0, rig: 0.9, palette: [red]}]`)
	t.Setenv("FPS", "999")
	t.Setenv("CUES", "not-a-file")
	for _, seconds := range []float64{10, -1, 2, 1, 0.25} {
		frameAt(t, show, seconds, 24)
	}
	frameAt(t, other, 2, 12)
	got := frameAt(t, show, 0.25, 24)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reuse changed frame: %#v != %#v", got, want)
	}
	got.FireSince[0] = 999
	got = frameAt(t, show, 0.25, 24)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("mutating a returned frame changed the show")
	}
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				got, err := show.At(0.25, 24)
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Errorf("concurrent At: %#v, %v", got, err)
				}
			}
		}()
	}
	wg.Wait()
}

func TestSheetOrderAndInclusiveCueBoundary(t *testing.T) {
	show := parsed(t, `cues: [{at_sec: 1, rig: 0.9}, {at_sec: 0, rig: 0.2}]`)
	if got := frameAt(t, show, 1, 24).Rig; got != 0.2 {
		t.Fatalf("sheet was sorted: rig %v", got)
	}
	show = parsed(t, `cues: [{at_sec: 1, rig: 0.9}]`)
	if got := frameAt(t, show, math.Nextafter(1, 0), 24).Rig; got != 0 {
		t.Fatal(got)
	}
	if got := frameAt(t, show, 1, 24).Rig; got != 0.9 {
		t.Fatal(got)
	}
}

func TestNilShowIsRefused(t *testing.T) {
	var show *Show
	if _, err := show.At(0, 24); err == nil {
		t.Fatal("accepted a nil show")
	}
}

func TestErrorsReturnWithoutExitingAndKeepEvaluationOrder(t *testing.T) {
	for _, each := range []struct {
		sheet string
		at    float64
		want  string
	}{
		{`cues: [{at: "99"}]`, -1, `cue at "99": bar off the grid`},
		{`cues: [{at: "bad"}]`, 0, `cue at "bad": strconv.Atoi`},
		{`cues: [{at: "1.bad"}]`, 0, `cue at "1.bad": strconv.Atoi`},
		{`cues: [{at_sec: 2, figures: [bad]}]`, 1, ""},
		{`cues: [{at_sec: 2, figures: [bad]}]`, 2, `cue at 2: no figure "bad"`},
		{`cues: [{at_sec: 0, banks: [bad]}]`, 0, `no banks "bad"`},
		{`cues: [{at_sec: 0, search_moves: [bad]}]`, 0, `no move "bad"`},
		{`cues: [{at_sec: 0, figures: [bad]}, {at: "99"}]`, 0, `no figure "bad"`},
	} {
		show := parsed(t, each.sheet)
		_, err := show.At(each.at, 24)
		if each.want == "" {
			if err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), each.want) {
			t.Fatalf("%s: got %v, want %s", each.sheet, err, each.want)
		}
	}
	for _, input := range [][2]string{{"score: [", ""}, {reuseScore, "cues: ["}} {
		if _, err := Parse([]byte(input[0]), []byte(input[1])); err == nil {
			t.Fatal("malformed YAML accepted")
		}
	}
	show := parsed(t, "")
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := show.At(v, 24); err == nil {
			t.Fatal("nonfinite time accepted")
		}
		if _, err := show.At(0, v); err == nil {
			t.Fatal("nonfinite FPS accepted")
		}
	}
}

func TestQuantizationMatchesTextRoundTripWithoutChangingRawFrame(t *testing.T) {
	show := parsed(t, `cues: [{at_sec: 0, rig: 0.123456, exposure: -0.00001, fire: 1, spots: 0.987654, search: 0.567891, palette: [cyan]}]`)
	raw := frameAt(t, show, 1.234567, 24)
	quantized := raw.Quantized()
	if raw.Rig != 0.123456 || quantized.Rig != 0.123 || raw.FireSince[0] == quantized.FireSince[0] {
		t.Fatalf("raw or quantized precision wrong: %#v %#v", raw, quantized)
	}
	if !math.Signbit(quantized.Exposure) {
		t.Fatal("negative zero lost")
	}
	var text bytes.Buffer
	n, err := raw.WriteTo(&text)
	if err != nil || n != int64(text.Len()) {
		t.Fatalf("WriteTo: %d, %v", n, err)
	}
	lines := map[string]string{}
	for _, line := range strings.Split(text.String(), "\n") {
		k, v, _ := strings.Cut(line, "=")
		lines[k] = v
	}
	fields := reflect.ValueOf(quantized)
	for i := range fields.NumField() {
		name := fields.Type().Field(i).Name
		var key strings.Builder
		for j, letter := range name {
			if letter >= 'A' && letter <= 'Z' {
				if j > 0 {
					key.WriteByte('_')
				}
				letter += 'a' - 'A'
			}
			key.WriteRune(letter)
		}
		value := fields.Field(i)
		if value.Kind() != reflect.Float64 {
			continue
		}
		want, err := strconv.ParseFloat(lines[key.String()], 64)
		if err != nil || value.Float() != want || math.Signbit(value.Float()) != math.Signbit(want) {
			t.Errorf("%s = %v, text %q (%v)", name, value.Float(), lines[key.String()], err)
		}
	}
	var roundedText bytes.Buffer
	quantized.WriteTo(&roundedText)
	if text.String() != roundedText.String() {
		t.Fatal("quantization changes serialized bytes")
	}
	quantized.FireSince[0] = 999
	if raw.FireSince[0] == 999 {
		t.Fatal("quantization aliases raw fire ages")
	}
}

func TestWriteToReportsWriterFailure(t *testing.T) {
	frame := frameAt(t, parsed(t, ""), 0, 24)
	failing := mocks.NewMockWriter(gomock.NewController(t))
	failing.EXPECT().Write(gomock.Any()).Return(0, errors.New("writer failed")).
		AnyTimes()
	if n, err := frame.WriteTo(failing); n != 0 || err == nil || err.Error() != "writer failed" {
		t.Fatal(fmt.Sprint(n, err))
	}
}
