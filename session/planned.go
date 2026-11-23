package session

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// A plan is what a picture needs of its set, decided from the set's own
// numbers and the camera before anything is made: each thing and each
// stand the set names, how big it is, how many pixels it covers, and
// whether it is built, averaged or dropped. See doc/plan-before-staging.md.
// This first plan is printed and not yet used: staging still makes
// everything.

const (
	// planPixel is the span, in pixels, at which a thing is built rather
	// than averaged. picked: a thing under a pixel cannot be told apart.
	planPixel = 1.0
	// planSamples is how many places across and down a stand's region is
	// sampled to share its members between near, far and out of view.
	// picked: enough that a stand's shares are whole percents.
	planSamples = 64
)

// The verdicts a plan gives.
const (
	verdictBuild   = "build"
	verdictAverage = "average"
	verdictDrop    = "drop"
	verdictOpen    = "open" // a stand some of whose members are built
)

// planLine is one thing or stand of the set, as the plan sees it.
type planLine struct {
	name, kind      string
	count           int
	size            maths.Vec
	away            float64
	pixels          float64
	built, averaged int
	dropped         int
	verdict, why    string
	asked           time.Duration
}

// Planned is the plan of the job's first frame, as text: a line for each
// thing and each stand of its set, then the totals. It reads the set and
// its terrain and makes nothing else.
func Planned(wanted job.Job) (string, error) {
	given, err := jobAsked(wanted)
	if err != nil {
		return "", err
	}
	started := time.Now()
	read, err := given.readSet()
	if err != nil {
		return "", err
	}
	if _, has := read.Ground(); !has || read.InSpace() {
		return "", fmt.Errorf("%s: a plan is made only for a set on "+
			"terrain, so far", given.set)
	}
	given, _ = fantasied(read, given)
	stage := &staging{given: given, world: thing.NewWorld(), have: stock{},
		places: map[string]maths.Vec{}, radii: map[string]float64{},
		recipes: read}
	if err := stage.terrain(read); err != nil {
		return "", err
	}
	ground := time.Since(started)
	things, err := stage.plannedThings(read)
	if err != nil {
		return "", err
	}
	eye, err := stage.seen()
	if err != nil {
		return "", err
	}
	lines := make([]planLine, 0, len(things)+len(read.Stands()))
	for _, one := range things {
		lines = append(lines, one.judged(eye))
	}
	for i, each := range read.Stands() {
		line, err := stage.plannedStand(each, i, eye)
		if err != nil {
			return "", fmt.Errorf("%s: %w", each.Name(), err)
		}
		lines = append(lines, line)
	}
	return planText(given, eye, lines, ground, time.Since(started)), nil
}

// plannedThing is a thing of the set placed and sized, not yet judged.
type plannedThing struct {
	line planLine
	box  maths.Box
}

// plannedThings places each thing the set names where staging would,
// and sizes it from its kind's extent, so the camera can stand by them.
func (stage *staging) plannedThings(read set.Set) ([]plannedThing,
	error) {
	var all []plannedThing
	for i, each := range read.Things() {
		place, absolute := stage.absolutePosition(each)
		if !absolute {
			east, north := each.At()
			var err error
			if place, err = stage.at(east, north); err != nil {
				return nil, fmt.Errorf("%s: %w", each.Name(), err)
			}
			place.Y += each.Height()
		}
		stage.places[each.Name()] = place
		turn := stage.placementTurn(each.Kind(), each.Name(), each.Seed(),
			uint64(i))
		at := maths.Subspace{Origin: place,
			Orientation: maths.About(maths.YAxis, turn)}
		box, asked, err := stage.extentAt(each.Kind(), each.Recipe(),
			each.Seed(), at)
		line := planLine{name: each.Name(), kind: each.Kind(), count: 1,
			asked: asked}
		if err != nil {
			line.verdict, line.why = verdictBuild,
				"its kind cannot say how big it is: "+err.Error()
			box = maths.Box{Min: place, Max: place}
		}
		line.size = box.Max.Sub(box.Min)
		all = append(all, plannedThing{line: line, box: box})
	}
	return all, nil
}

// extentAt is the box a kind or recipe would make from a seed, standing
// at a place, and how long the kind took to say.
func (stage *staging) extentAt(kind, recipe string, seed uint64,
	at maths.Subspace) (maths.Box, time.Duration, error) {
	started := time.Now()
	row, err := stage.construction(kind, recipe)
	if err != nil {
		return maths.Box{}, time.Since(started), err
	}
	sown := manufacture.Sow(placement.Seed(stage.given.seed, int(seed)))
	local, err := manufacture.ExtentOf(*row, sown)
	if err != nil {
		return maths.Box{}, time.Since(started), err
	}
	return placedBox(local, at), time.Since(started), nil
}

// placedBox is a local box stood at a place: the box that holds its
// eight corners turned and moved.
func placedBox(local maths.Box, at maths.Subspace) maths.Box {
	var placed maths.Box
	for corner := range 8 {
		point := maths.V(either(corner&1, local.Min.X, local.Max.X),
			either(corner&2, local.Min.Y, local.Max.Y),
			either(corner&4, local.Min.Z, local.Max.Z))
		world := at.Out(point)
		placed = joined(placed, maths.Box{Min: world, Max: world},
			corner == 0)
	}
	return placed
}

// either is the first number where a bit is clear, the second where set.
func either(bit int, clear, set float64) float64 {
	if bit == 0 {
		return clear
	}
	return set
}

// judged is a thing's verdict from how the eye sees its box.
func (one plannedThing) judged(eye strategies.Eye) planLine {
	line := one.line
	seen := eye.Cover(one.box)
	middle := one.box.Min.Add(one.box.Max).Scale(0.5)
	line.away = middle.Sub(eye.Place.Origin).Len()
	line.pixels = seen.Pixels
	if line.verdict != "" {
		line.built = 1
		return line
	}
	switch {
	case !seen.Visible:
		line.verdict, line.why, line.dropped = verdictDrop,
			"out of view", 1
	case seen.Pixels < planPixel:
		line.verdict, line.why, line.averaged = verdictAverage,
			"under a pixel", 1
	default:
		line.verdict, line.why, line.built = verdictBuild,
			fmt.Sprintf("%.0f pixels", seen.Pixels), 1
	}
	return line
}

// plannedStand judges a stand without placing its members: its region is
// sampled, each sample taken as where a member could stand, and its count
// shared between the samples that are out of view, under a pixel and
// over one. A member's size is the largest of its designs.
func (stage *staging) plannedStand(each set.Stand, index int,
	eye strategies.Eye) (planLine, error) {
	count, seed := each.Count()
	line := planLine{name: each.Name(), kind: each.Kind(), count: count}
	member, asked, err := stage.memberExtent(each, seed)
	line.asked = asked
	if err != nil {
		line.verdict, line.why, line.built = verdictBuild,
			"its kind cannot say how big it is: "+err.Error(), count
		return line, nil
	}
	line.size = member.Max.Sub(member.Min)
	reach := line.size.Len()
	region, err := stage.standRegion(each)
	if err != nil {
		return line, err
	}
	above, below := each.Band()
	var out, under, over, nearest = 0, 0, 0, math.Inf(1)
	for row := range planSamples {
		for column := range planSamples {
			east := region.Min.X + (float64(column)+0.5)/planSamples*
				(region.Max.X-region.Min.X)
			z := region.Min.Z + (float64(row)+0.5)/planSamples*
				(region.Max.Z-region.Min.Z)
			place, err := stage.at(east+stage.west, stage.north-z)
			if err != nil {
				continue
			}
			if height := place.Y + stage.lowest; !(height > above &&
				height < below) {
				continue
			}
			middle := place.Add(maths.V(0, line.size.Y/2, 0))
			away := middle.Sub(eye.Place.Origin).Len()
			nearest = math.Min(nearest, away)
			switch {
			case !eye.SeesBall(middle, reach/2):
				out++
			case eye.Span(middle, reach) < planPixel:
				under++
			default:
				over++
			}
		}
	}
	sampled := out + under + over
	if sampled == 0 {
		line.verdict, line.why, line.dropped = verdictDrop,
			"no ground in its band", count
		return line, nil
	}
	line.away = nearest
	line.pixels = eye.Span(eye.Place.Origin.Add(maths.V(0, 0,
		-nearest)), reach)
	line.dropped = count * out / sampled
	line.built = count * over / sampled
	line.averaged = count - line.dropped - line.built
	switch {
	case line.built > 0:
		line.verdict = verdictOpen
	case line.averaged > 0:
		line.verdict = verdictAverage
	default:
		line.verdict = verdictDrop
	}
	line.why = fmt.Sprintf("%d%% out of view, %d%% under a pixel, "+
		"%d%% over", 100*out/sampled, 100*under/sampled, 100*over/sampled)
	return line, nil
}

// memberExtent is the box that holds every design a stand's members are
// made from, in their own frame, and how long the kind took to say.
func (stage *staging) memberExtent(each set.Stand, seed uint64) (maths.Box,
	time.Duration, error) {
	var total time.Duration
	var all maths.Box
	for design := range standVariety {
		pick := standPick(each, seed, design)
		box, asked, err := stage.extentAt(each.Kind(), each.Recipe(), pick,
			maths.Subspace{Orientation: maths.Identity})
		total += asked
		if err != nil {
			return maths.Box{}, total, err
		}
		all = joined(all, box, design == 0)
	}
	return all, total, nil
}

// joined is two boxes together, or the second alone where first says so.
func joined(have, more maths.Box, first bool) maths.Box {
	if first {
		return more
	}
	return maths.Box{
		Min: maths.V(math.Min(have.Min.X, more.Min.X),
			math.Min(have.Min.Y, more.Min.Y), math.Min(have.Min.Z, more.Min.Z)),
		Max: maths.V(math.Max(have.Max.X, more.Max.X),
			math.Max(have.Max.Y, more.Max.Y), math.Max(have.Max.Z, more.Max.Z))}
}

// standRegion is where a stand's members can stand, in the ground's
// frame: its area where it says one, the ring round what it stands
// within, or else the whole ground. It may be larger than where they
// stand; it is never smaller.
func (stage *staging) standRegion(each set.Stand) (maths.Box, error) {
	across, down, step := stage.ground.Size()
	whole := maths.Box{Max: maths.V(float64(across-1)*step, 0,
		float64(down-1)*step)}
	if west, south, east, north, has := each.Area(); has {
		return maths.Box{Min: maths.V(west-stage.west, 0, stage.north-north),
			Max: maths.V(east-stage.west, 0, stage.north-south)}, nil
	}
	of, _, _, _, far, within := each.Within()
	if !within {
		return whole, nil
	}
	seen, known := stage.places[of]
	if !known {
		return maths.Box{}, fmt.Errorf("within sight of %s, which stands "+
			"nowhere on the ground", of)
	}
	return maths.Box{Min: maths.V(seen.X-far, 0, seen.Z-far),
		Max: maths.V(seen.X+far, 0, seen.Z+far)}, nil
}

// planText is the plan as a person reads it: the picture, a line for
// each thing and stand, and what it comes to.
func planText(given asked, eye strategies.Eye, lines []planLine,
	ground, whole time.Duration) string {
	var text strings.Builder
	fmt.Fprintf(&text, "plan of %s at %d by %d, made in %s (the terrain "+
		"%s); nothing else was made\n", given.set, eye.Across, eye.Down,
		whole.Round(time.Millisecond), ground.Round(time.Millisecond))
	fmt.Fprintf(&text, "%-28s %-22s %7s %8s %8s %9s %7s %7s %7s  %s\n",
		"name", "kind", "count", "size m", "away m", "pixels", "built",
		"avg", "drop", "verdict, why (asked in)")
	var built, averaged, dropped, count int
	for _, line := range lines {
		built, averaged, dropped = built+line.built,
			averaged+line.averaged, dropped+line.dropped
		count += line.count
		fmt.Fprintf(&text, "%-28s %-22s %7d %8.1f %8.0f %9.2f %7d %7d %7d  "+
			"%s, %s (%s)\n", clipped(line.name, 28), clipped(line.kind, 22),
			line.count, line.size.Len(), line.away, line.pixels, line.built,
			line.averaged, line.dropped, line.verdict, line.why,
			line.asked.Round(time.Microsecond))
	}
	fmt.Fprintf(&text, "total: %d things asked for; %d built, %d averaged, "+
		"%d dropped\n", count, built, averaged, dropped)
	return text.String()
}

// clipped is a name cut to a column's width.
func clipped(name string, width int) string {
	if len(name) <= width {
		return name
	}
	return name[:width-1] + "~"
}
