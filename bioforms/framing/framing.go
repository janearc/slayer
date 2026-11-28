// Package framing suggests and tracks perspective views of posed geometry.
package framing

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// Part is a posed part's local bound and its placement within the root.
type Part struct {
	Bounds    maths.Box
	Placement maths.Subspace
}

// Pose gives the parts to frame and the root's placement in scene space.
// The root's local +z is front, +x is the profile side and +y is up.
type Pose struct {
	Root  maths.Subspace
	Parts []Part
}

// Source supplies posed geometry at the renderer's scene time.
type Source func(time.Duration) (Pose, error)

// View describes a level camera angle relative to the subject's front.
type View string

const (
	Front        View = "front"
	ThreeQuarter View = "three-quarter"
	Profile      View = "profile"
)

// Settings controls perspective framing. Aspect is the picture width over
// height including pixel aspect, and is required. Zero selects the defaults
// for Wide (radians), Fill (fraction of frame), and Clearance (scene units).
// WorldAligned uses scene axes for the camera heading and up.
type Settings struct {
	Aspect, Wide, Fill, Clearance float64
	WorldAligned                  bool
}

// Request defines what the tracker follows and how to frame it. An empty
// View selects ThreeQuarter.
type Request struct {
	Source   Source
	Settings Settings
	View     View
}

type tracker struct {
	source   Source
	settings Settings
	view     View
	across   float64
	down     float64
}

// Shot is a checked pinhole camera framing one sampled pose.
type Shot struct {
	eye    maths.Subspace
	target maths.Vec
	up     maths.Vec
	wide   float64
	view   View
}

// Eye is the camera placement in scene space.
func (shot Shot) Eye() maths.Subspace { return shot.eye }

// Target is the world bounds centre the camera looks at.
func (shot Shot) Target() maths.Vec { return shot.target }

// Wide is the horizontal field of view in radians.
func (shot Shot) Wide() float64 { return shot.wide }

// View is the suggested angle used for the shot.
func (shot Shot) View() View { return shot.view }

// New prepares a tracker that can sample independently at any scene time.
func New(request Request) (*tracker, error) {
	if request.Source == nil {
		return nil, fmt.Errorf("framing: need a pose source")
	}
	view := request.View
	if view == "" {
		view = ThreeQuarter
	}
	switch view {
	case Front, ThreeQuarter, Profile:
	default:
		return nil, fmt.Errorf("framing: unknown view %q", view)
	}
	settings := request.Settings.defaults()
	if !finite(settings.Aspect) || settings.Aspect <= 0 ||
		!(settings.Wide > 0 && settings.Wide < math.Pi) ||
		!(settings.Fill > 0 && settings.Fill <= 1) ||
		!finite(settings.Clearance) || settings.Clearance <= 0 {
		return nil, fmt.Errorf("framing: need positive aspect and clearance, an angle in (0, pi), and fill in (0, 1]")
	}
	across := math.Tan(settings.Wide / 2)
	down := across / settings.Aspect
	if !finite(down) || down*settings.Fill <= 0 || across*settings.Fill <= 0 {
		return nil, fmt.Errorf("framing: projection exceeds the numeric range")
	}
	return &tracker{source: request.Source, settings: settings, view: view,
		across: across, down: down}, nil
}

func (settings Settings) defaults() Settings {
	if settings.Wide == 0 {
		settings.Wide = .60
	}
	if settings.Fill == 0 {
		settings.Fill = 1 / 1.13
	}
	if settings.Clearance == 0 {
		settings.Clearance = .01
	}
	return settings
}

// At fits the requested view to the subject at scene time at.
func (tracking *tracker) At(at time.Duration) (Shot, error) {
	bounds, err := tracking.sample(at)
	if err != nil {
		return Shot{}, err
	}
	shot, err := tracking.fit(bounds, tracking.view)
	if err != nil {
		return Shot{}, fmt.Errorf("framing at %s: %w", at, err)
	}
	return shot, nil
}

// Suggestions fits three named views to one sample of the subject at at.
func (tracking *tracker) Suggestions(at time.Duration) ([]Shot, error) {
	bounds, err := tracking.sample(at)
	if err != nil {
		return nil, err
	}
	views := [...]View{Front, ThreeQuarter, Profile}
	shots := make([]Shot, 0, len(views))
	for _, view := range views {
		shot, err := tracking.fit(bounds, view)
		if err != nil {
			return nil, fmt.Errorf("framing at %s: %w", at, err)
		}
		shots = append(shots, shot)
	}
	return shots, nil
}

func (tracking *tracker) sample(at time.Duration) (posedBounds, error) {
	if tracking == nil || tracking.source == nil {
		return posedBounds{}, fmt.Errorf("framing: no tracker")
	}
	pose, err := tracking.source(at)
	if err != nil {
		return posedBounds{}, fmt.Errorf("framing pose at %s: %w", at, err)
	}
	bounds, err := pose.bounds()
	if err != nil {
		return posedBounds{}, fmt.Errorf("framing pose at %s: %w", at, err)
	}
	if tracking.settings.WorldAligned {
		bounds.basis = maths.Identity
	}
	return bounds, nil
}

func (view View) back() maths.Vec {
	switch view {
	case Front:
		return maths.ZAxis
	case Profile:
		return maths.XAxis
	default:
		return maths.XAxis.Add(maths.ZAxis).Scale(math.Sqrt(.5))
	}
}

func (tracking *tracker) fit(bounds posedBounds, view View) (Shot, error) {
	up := bounds.basis.Out(maths.YAxis)
	back := bounds.basis.Out(view.back())
	basis, err := maths.View(back, maths.Vec{}, up)
	if err != nil {
		return Shot{}, fmt.Errorf("%s camera direction: %w", view, err)
	}
	across := tracking.across * tracking.settings.Fill
	down := tracking.down * tracking.settings.Fill
	distance := tracking.settings.Clearance
	for _, corner := range bounds.corners {
		point := basis.Orientation.In(corner.Sub(bounds.target))
		// Perspective divides both image axes by distance minus the
		// corner's displacement toward the eye.
		fit := max(math.Abs(point.X)/across, math.Abs(point.Y)/down)
		distance = max(distance, point.Z+max(fit, tracking.settings.Clearance))
	}
	distance = math.Nextafter(distance, math.Inf(1))
	origin := bounds.target.Add(basis.Orientation.Z.Scale(distance))
	eye, err := maths.View(origin, bounds.target, up)
	if err != nil {
		return Shot{}, fmt.Errorf("%s camera position: %w", view, err)
	}
	for _, corner := range bounds.corners {
		point := eye.In(corner)
		depth := -point.Z
		if !(depth > 0 && math.Abs(point.X)/depth <= tracking.across &&
			math.Abs(point.Y)/depth <= tracking.down) {
			return Shot{}, fmt.Errorf("%s camera cannot fit the bounds at this numeric scale", view)
		}
	}
	return Shot{eye: *eye, target: bounds.target, up: up,
		wide: tracking.settings.Wide, view: view}, nil
}
