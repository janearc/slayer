//go:build draw

package hillside

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/scene/thing"
)

// exposureLens is a job's film with the tests' aperture and focus in it.
var exposureLens = fmt.Sprintf("film: {width: %d, height: %d, rays: %d, "+
	"aperture: %g, focus: %g}\n", namedWidth, namedHeight, namedRays,
	exposureAperture, exposureFocus)

// lensJob is a still of the swarm through the tests' lens, with more.
func lensJob(more string) string {
	return "scene: {named: swarm}\n" + exposureLens + more
}

// asAsked is what a session's frame is asked with, as its own job says.
func asAsked(t *testing.T, session *Session) asked {
	t.Helper()
	given, err := session.asking(Ask{})
	if err != nil {
		t.Fatal(err)
	}
	return given
}

// a job's aperture and focus make a thin lens for the frame: it sees the
// frame's own angle across the frame's own shape, opens as wide as the
// job says and focuses as far off, and is the lens the frame is seen
// through
func TestAJobsApertureMakesAThinLensOfTheFramesAngle(t *testing.T) {
	given := asAsked(t, opening(t, jobFrom(t, lensJob(""))))
	if err := (sequencePlan{}).exposing(&given, nothing); err != nil {
		t.Fatal(err)
	}
	glass := given.optics.opened
	if glass == nil {
		t.Fatal("a job with an aperture made no lens")
	}
	reach, focus, across, up := glass.Opening()
	for _, each := range []struct {
		name      string
		got, want float64
	}{
		{"the angle across", glass.Wide(), given.wide * maths.Degree},
		{"the aperture", reach * shareHalves, exposureAperture},
		{"the focus", focus, exposureFocus},
		{"the frame's shape", across / up,
			float64(namedWidth) / float64(namedHeight)},
	} {
		if !(math.Abs(each.got-each.want) <= exposureClose*each.want) {
			t.Errorf("%s is %v, want %v", each.name, each.got, each.want)
		}
	}
	if seen, thin := lensFor(given).(lens.Thin); !thin || seen != *glass {
		t.Errorf("the frame is seen through %T, not the job's lens",
			lensFor(given))
	}
	want := film.Pinhole{Wide: given.wide * maths.Degree}
	if sight := sightFor(given); sight != film.Sensor(want) {
		t.Errorf("a hud's labels are placed through %v, not the pinhole "+
			"of the lens's angle", sight)
	}
}

// a job that says no aperture is seen through a pinhole, and its world
// is told nothing of a ray's angle
func TestAJobWithNoApertureIsAPinhole(t *testing.T) {
	given := asAsked(t, opening(t, jobFrom(t, namedJob("named: swarm",
		""))))
	if err := (sequencePlan{}).exposing(&given, nothing); err != nil {
		t.Fatal(err)
	}
	if _, pinhole := lensFor(given).(film.Pinhole); !pinhole ||
		given.optics.opened != nil || sightFor(given) != lensFor(given) {
		t.Errorf("a job with no aperture is seen through %T",
			lensFor(given))
	}
	world := thing.NewWorld()
	given.footprint(world)
	if world.Pixel != nothing {
		t.Errorf("a pinhole's world is told a ray sees %v", world.Pixel)
	}
}

// a world seen through the job's lens is told the angle one ray of the
// picture sees, which rays from across the aperture do not say
func TestAJobsLensTellsTheWorldWhatARaySees(t *testing.T) {
	given := asAsked(t, opening(t, jobFrom(t, lensJob(""))))
	if err := (sequencePlan{}).exposing(&given, nothing); err != nil {
		t.Fatal(err)
	}
	world := thing.NewWorld()
	given.footprint(world)
	want := given.optics.opened.Pixel(namedWidth * namedRays)
	if world.Pixel != want || !(want > nothing) {
		t.Errorf("the world is told a ray sees %v, want %v", world.Pixel,
			want)
	}
}

// a job's shutter gives each glimpse a share of the open interval, the
// glimpse its middle; a job with no shutter has no share
func TestAGlimpsesShareIsTheOpenIntervalOverTheGlimpses(t *testing.T) {
	session := shutterSmoke(t, shutterFilm(shutterHalf, shutterGlimpses))
	open := time.Second / shutterRate * shutterHalf / shutterWhole
	if want := open / shutterGlimpses; session.share != want ||
		session.offsets[nothing] != want/shareHalves {
		t.Errorf("a glimpse's share is %v and the first is %v in; want "+
			"%v and half of it", session.share, session.offsets[nothing],
			want)
	}
	plain := shutterSmoke(t, fmt.Sprintf("fps: %d\nduration: {value: 1, "+
		"unit: s}\n", shutterRate))
	if plain.share != nothing {
		t.Errorf("a film with no shutter has a share of %v", plain.share)
	}
}
