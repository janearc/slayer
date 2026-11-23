package medium_test

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/burning-trousers/mocks/mediummock"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"go.uber.org/mock/gomock"
)

// forwardedTime is the time the wrappers are ticked to: defined by these
// tests alone.
const forwardedTime = 3 * time.Second

// The csv files the forwarding tests read their wispy noise and puffs from.
const (
	forwardingWispiesFile = "testdata/forwarding-wispies.csv"
	forwardingPuffsFile   = "testdata/forwarding-puffs.csv"
)

// forwardingWispy is a wispy medium of what it is given, with the noise in
// the wispies file.
func forwardingWispy(t *testing.T, of medium.Medium) medium.Wispy {
	t.Helper()
	row := readSheet(t, forwardingWispiesFile).row(t, forwardingWispiesFile,
		"noise")
	return medium.Wispy{
		Of:     of,
		Seed:   uint64(cellCount(t, row, "seed")),
		Scale:  cell(t, row, "scale"),
		Amount: cell(t, row, "amount"),
	}
}

// forwardingPuff is the puff in the puffs file called name.
func forwardingPuff(t *testing.T, name string) medium.Puff {
	t.Helper()
	row := readSheet(t, forwardingPuffsFile).row(t, forwardingPuffsFile, name)
	return medium.Puff{
		Radius: cell(t, row, "radius"),
		Thick:  cell(t, row, "thick"),
	}
}

// forwardingOrigin is the point at the middle of space.
func forwardingOrigin() maths.Vec { return maths.V(nought, nought, nought) }

// a wispy medium moves as the medium it makes ragged does: ticked, it ticks
// what it wraps, and keeps its noise.
func TestWispyForwardsTickToWhatItWraps(t *testing.T) {
	ctrl := gomock.NewController(t)
	inner := mediummock.NewMockTimed(ctrl)
	later := mediummock.NewMockMedium(ctrl)
	inner.EXPECT().Tick(forwardedTime).Return(later)
	wispy := forwardingWispy(t, inner)
	ticked, ok := wispy.Tick(forwardedTime).(medium.Wispy)
	if !ok {
		t.Fatal("a ticked wispy medium is not a wispy medium")
	}
	if ticked.Of != medium.Medium(later) {
		t.Errorf("the ticked wispy wraps %v, want the medium ticked to",
			ticked.Of)
	}
	if ticked.Seed != wispy.Seed || ticked.Scale != wispy.Scale ||
		ticked.Amount != wispy.Amount {
		t.Errorf("the ticked wispy's noise changed: %+v", ticked)
	}
}

// a wispy medium of a medium that stands still, or of none, ticks to itself
// and asks nothing of it.
func TestWispyOfAStaticMediumTicksToItself(t *testing.T) {
	puff := forwardingPuff(t, "static")
	wispy := forwardingWispy(t, puff)
	if ticked := wispy.Tick(forwardedTime); ticked != medium.Medium(wispy) {
		t.Errorf("a wispy puff ticked to %v", ticked)
	}
	var none medium.Wispy
	if ticked := none.Tick(forwardedTime); ticked != medium.Medium(none) {
		t.Errorf("a wispy none ticked to %v", ticked)
	}
}

// refusal is whether a call panics with words that name the constructor.
func refusal(t *testing.T, what string, call func()) {
	t.Helper()
	defer forwardingNamesConstructor(t, what)
	call()
}

// forwardingNamesConstructor recovers a panic and fails the test unless the
// panic's words name the constructor of a clear cloud.
func forwardingNamesConstructor(t *testing.T, what string) {
	t.Helper()
	text, ok := recover().(string)
	if !ok || !strings.Contains(text, "NewClearCloud") {
		t.Errorf("%s: refused with %q, want words naming NewClearCloud",
			what, text)
	}
}

// forwardingZero is a clear cloud that was not made by its constructor, with
// each use of it as a call that takes and gives nothing.
type forwardingZero struct{ cloud medium.ClearCloud }

// bounds uses the cloud's bounds.
func (zero forwardingZero) bounds() { zero.cloud.Bounds() }

// density uses the cloud's density at the middle of space.
func (zero forwardingZero) density() {
	zero.cloud.Density(forwardingOrigin())
}

// span uses the cloud's span for the zero ray, from nought to unity.
func (zero forwardingZero) span() {
	zero.cloud.Span(maths.Ray{}, nought, unity)
}

// a clear cloud that was not made by its constructor wraps nothing, and
// refuses in words, each use of it, and not by a nil dereference.
func TestAZeroClearCloudRefusesInWords(t *testing.T) {
	var cloud medium.ClearCloud
	zero := forwardingZero{cloud}
	refusal(t, "Bounds", zero.bounds)
	refusal(t, "Density", zero.density)
	refusal(t, "Span", zero.span)
}

// a clear cloud that was made is used without refusal: its bounds, density
// and span are the field's, and a zero one ticks to itself.
func TestAMadeClearCloudIsNotRefused(t *testing.T) {
	field := forwardingPuff(t, "field")
	cloud, err := medium.NewClearCloud(field)
	if err != nil {
		t.Fatal(err)
	}
	middle := forwardingOrigin()
	if cloud.Bounds() != field.Bounds() ||
		cloud.Density(middle) != field.Density(middle) {
		t.Error("the cloud is not its field")
	}
	var zero medium.ClearCloud
	if ticked := zero.Tick(forwardedTime); ticked != medium.Medium(zero) {
		t.Errorf("a zero cloud ticked to %v", ticked)
	}
}
