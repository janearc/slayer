package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// the numbers the gargoyles of a tower are placed by, which are picked
const (
	// setbackShare is how much narrower the next segment of a shaft must
	// be than the top of the one under it, as a share, for the top to be
	// a ledge that gargoyles stand under. picked: a visible step.
	setbackShare = 0.98
)

// faceOut is the unit direction in plan, as a corner, that is out of the
// outline from the edge running from first to second. The sign of the
// outline's area, which says how it is wound, decides which side is out,
// so an outline that is not convex has the right side for each edge.
func faceOut(first, second outlineCorner, area float64) outlineCorner {
	run := outlineCorner{second.X - first.X, second.Z - first.Z}
	length := math.Hypot(run.X, run.Z)
	out := outlineCorner{run.Z / length, -run.X / length}
	if area < 0 {
		return outlineCorner{-out.X, -out.Z}
	}
	return out
}

// standingOut is the place of a gargoyle at a point of a plan, at a
// height, on a face whose unit direction out of the tower is out: the
// frame a cornice's gargoyle has, with its face in the plane z is nought
// and out of the wall along z, which is x along the wall and y up.
func standingOut(at outlineCorner, height float64, out outlineCorner) maths.Subspace {
	along := maths.V(out.Z, 0, -out.X)
	return maths.NewSubspace(maths.V(at.X, height, at.Z),
		maths.Orientation{X: along, Y: maths.YAxis,
			Z: maths.V(out.X, 0, out.Z)}, 0)
}

// ledge is an outline that gargoyles stand under, with its height: the
// top of a segment of a shaft that has the next one set back from it, or
// the top of the last, as a cornice. The outline is a part of the plan at
// the top of the segment, in the tower's plan.
type ledge struct {
	outline []outlineCorner
	height  float64
}

// ledges are the ledges of a tower, in order from the foot of each shaft.
func (tower *Tower) ledges() ([]ledge, error) {
	var all []ledge
	storey := tower.storeyHeight()
	for _, shaft := range tower.Shafts {
		height := tower.baseHeight()
		for index, segment := range shaft.Segments {
			height += float64(segment.Storeys) * storey
			last := index == len(shaft.Segments)-1
			if !last && shaft.Segments[index+1].Bottom.radius() >=
				segment.Top.radius()*setbackShare {
				continue
			}
			_, tops, err := shaft.layer(index)
			if err != nil {
				return nil, err
			}
			for _, top := range tops {
				all = append(all, ledge{top, height})
			}
		}
	}
	return all, nil
}

// GargoyleMounts are the gargoyles of a tower, in a stone of a name or
// the stone the seed draws if the name is empty. Each stands on a corbel
// under a ledge: at every convex corner of the plan that turns by at
// least tower-corner-turn, facing out along the diagonal, and along each
// face every tower-pitch, clear of its ends by tower-edge-margin.
//
// The creatures are those of gargoyle-creatures.yaml in the order it writes
// them, going round again, and each place is a frame with its face in
// the plane z is nought and out of the tower along z.
func (tower *Tower) GargoyleMounts(stone string) ([]Mount, error) {
	drawing, err := newDrawer(gargoyleCulture, tower.seed)
	if err != nil {
		return nil, err
	}
	chosen, err := drawing.stoneOf(stone)
	if err != nil {
		return nil, err
	}
	creatures, err := GargoyleCreatures()
	if err != nil {
		return nil, err
	}
	ledges, err := tower.ledges()
	if err != nil {
		return nil, err
	}
	at := mountSite{drop: drawing.fixed("tower-gargoyle-drop"),
		pitch:  drawing.fixed("tower-pitch"),
		margin: drawing.fixed("tower-edge-margin"),
		turn:   drawing.fixed("tower-corner-turn"),
		proud:  drawing.fixed("tower-boss-proud")}
	if drawing.trouble != nil {
		return nil, drawing.trouble
	}
	var mounts []Mount
	for _, each := range ledges {
		for _, place := range at.along(each) {
			mounts = append(mounts, Mount{Kind: GargoyleKind(
				creatures[len(mounts)%len(creatures)], chosen.Name),
				Place: place})
		}
	}
	return mounts, nil
}

// mountSite is how a ledge's gargoyles are placed: how far below the
// ledge, how far apart, how far from a corner, the least turn at a corner,
// and how far out of the face.
type mountSite struct {
	drop, pitch, margin, turn, proud float64
}

// along is the places of the gargoyles under a ledge, the corners of its
// outline first and then its faces.
func (site mountSite) along(each ledge) []maths.Subspace {
	area := signedArea(each.outline)
	height := each.height - site.drop
	var places []maths.Subspace
	count := len(each.outline)
	for index, corner := range each.outline {
		before := each.outline[(index+count-1)%count]
		after := each.outline[(index+1)%count]
		places = append(places, site.cornerAt(before, corner, after, area,
			height)...)
		places = append(places, site.faceAt(corner, after, area, height)...)
	}
	return places
}

// cornerAt is the place of the gargoyle at a corner, if the outline turns
// there by enough and the corner is convex, facing out along the diagonal.
func (site mountSite) cornerAt(before, corner, after outlineCorner, area,
	height float64) []maths.Subspace {
	into, out := faceOut(before, corner, area), faceOut(corner, after, area)
	across := into.X*out.Z - into.Z*out.X
	if area < 0 {
		across = -across
	}
	turned := math.Acos(math.Max(-1, math.Min(1, into.X*out.X+into.Z*out.Z))) *
		halfTurn / math.Pi
	if !(across > 0) || turned < site.turn {
		return nil
	}
	diagonal := outlineCorner{into.X + out.X, into.Z + out.Z}
	length := math.Hypot(diagonal.X, diagonal.Z)
	diagonal = outlineCorner{diagonal.X / length, diagonal.Z / length}
	at := outlineCorner{corner.X + diagonal.X*site.proud,
		corner.Z + diagonal.Z*site.proud}
	return []maths.Subspace{standingOut(at, height, diagonal)}
}

// faceAt is the places of the gargoyles along a face from one corner to
// the next, a pitch apart, centred and clear of each end by the margin.
func (site mountSite) faceAt(first, second outlineCorner, area,
	height float64) []maths.Subspace {
	length := math.Hypot(second.X-first.X, second.Z-first.Z)
	room := length - 2*site.margin
	if !(room >= 0) || !(site.pitch > 0) {
		return nil
	}
	count := int(room/site.pitch) + 1
	start := site.margin + (room-float64(count-1)*site.pitch)/2
	out := faceOut(first, second, area)
	var places []maths.Subspace
	for step := range count {
		share := (start + float64(step)*site.pitch) / length
		at := outlineCorner{
			first.X + (second.X-first.X)*share + out.X*site.proud,
			first.Z + (second.Z-first.Z)*share + out.Z*site.proud}
		places = append(places, standingOut(at, height, out))
	}
	return places
}
