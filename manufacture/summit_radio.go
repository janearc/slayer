package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/scene/shape"
)

// parabolaDiameterFactor is defined by z=r*r/(4*f) with diameter D=2*r.
const parabolaDiameterFactor = 16

// focalRatio uses the SMA's measured ratio and picked ratios for other dishes.
func (row summitInstrument) focalRatio() float64 {
	if row.FocalRatio != "" {
		return research.Value(row.FocalRatio)
	}
	return summitValue("radio-focal-ratio")
}

// radio makes a backed paraboloid with a feed and four separate supports.
func (row summitInstrument) radio(assembly *instrumentAssembly) {
	aperture, pose := row.aperture(), row.frame()
	focus := row.focalRatio() * aperture
	sag := aperture / (parabolaDiameterFactor * row.focalRatio())
	primary := radioDish{radius: aperture * spaceHalf, depth: sag,
		thickness: aperture * summitValue("radio-shell-thickness")}
	assembly.add(primary, pose, spaceMirror)
	length := focus * summitValue("radio-feed-seat")
	radius := aperture * summitValue("radio-feed-radius")
	flipped := tiltedFrame(maths.YAxis.Scale(length), math.Pi).Within(pose)
	secondary := mirrorDish{radius: radius,
		depth: aperture * summitValue("radio-feed-sag")}
	assembly.add(secondary, flipped, spaceSecondary)
	row.feedSupports(assembly, pose, sag, length, radius)
	row.receiverCabin(assembly)
}

// feedSupports keeps each attachment distinct at both ends of the quadrupod.
func (row summitInstrument) feedSupports(assembly *instrumentAssembly,
	pose maths.Subspace, sag, length, radius float64,
) {
	aperture := row.aperture()
	count := int(summitValue("radio-feed-arms"))
	for index := range count {
		angle := fullTurn * float64(index) / float64(count)
		sine, cosine := math.Sincos(angle)
		direction := maths.V(cosine, spaceZero, sine)
		bottom := direction.Scale(aperture * summitValue("radio-arm-foot"))
		bottom.Y = sag * summitValue("radio-arm-height")
		top := direction.Scale(radius * summitValue("radio-arm-tip"))
		top.Y = length - aperture*summitValue("radio-feed-sag")
		from, to := pose.Out(bottom), pose.Out(top)
		assembly.rod(from, to, aperture*summitValue("radio-arm-radius"),
			spaceStructure)
	}
}

// receiverCabin suggests the electronics carried below the dish.
func (row summitInstrument) receiverCabin(assembly *instrumentAssembly) {
	aperture := row.aperture()
	width := aperture * summitValue("radio-cabin-width")
	depth := aperture * summitValue("radio-cabin-depth")
	height := aperture * summitValue("radio-cabin-height")
	corner := maths.V(-width*spaceHalf,
		row.Seat-height-aperture*summitValue("radio-cabin-gap"),
		-depth*spaceHalf)
	assembly.box(width, height, depth, corner, leaf)
}

// array keeps antenna prototypes shared, rather than flattening every copy.
func (row summitInstrument) array(detail string) (good, error) {
	if row.ArrayCount != "" && len(row.Array) != int(research.Value(row.ArrayCount)) {
		//ninja:vow feint value API; Things rejects it
		return good{}, fmt.Errorf("manufacture: %s antenna inventory differs from %s",
			row.Kind, row.ArrayCount)
	}
	prototype := row
	prototype.Array = nil
	seed := sow(uint64(spaceOne))
	one, err := prototype.madeAt(seed, detail)
	if err != nil {
		//ninja:vow feint value API; Things rejects it
		return good{}, err
	}
	groups, err := row.arrayGroups(one)
	if err != nil {
		//ninja:vow feint value API; Things rejects it
		return good{}, err
	}
	bounds := maths.SegmentBox(row.Minimum, row.Maximum, spaceZero)
	return good{dressed: one.dressed, groups: groups, extent: &bounds}, nil
}

// arrayGroups uses one flat mesh per material and a checked local hierarchy.
func (row summitInstrument) arrayGroups(one good) (map[slot]*shape.MeshGroup, error) {
	groups := map[slot]*shape.MeshGroup{}
	for material, skin := range one.skins() {
		polygon, err := skin.meshed()
		if err != nil {
			return nil, err
		}
		members := make([]shape.MeshMember, len(row.Array))
		for index, at := range row.Array {
			members[index].Mesh = polygon
			members[index].Space = maths.NewSubspace(at, maths.Identity, spaceOne)
		}
		group, err := shape.NewMeshGroup(members)
		if err != nil {
			return nil, err
		}
		groups[material] = group
	}
	return groups, nil
}
