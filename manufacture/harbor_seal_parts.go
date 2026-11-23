package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// SealLook is how a seal is dressed over its skin. Wet runs from dry to
// soaked and Nostrils from open to shut, each in [0, 1]. A phocid's
// nostrils are shut at rest and open only to breathe. Seed varies the
// pelage's spots.
type SealLook struct {
	Wet, Nostrils float64
	Seed          uint64
}

// check refuses a look whose wet or nostril share is not a number from
// nought to one.
func (look SealLook) check() error {
	for _, share := range [...]float64{look.Wet, look.Nostrils} {
		if math.IsNaN(share) || share < 0 || share > 1 {
			return fmt.Errorf("seal look: share outside [0, 1]")
		}
	}
	return nil
}

// sealAnchor is a landmark on the rest skin carried to the pose, with the
// posed skin's outward, forward, upward and leftward directions there.
type sealAnchor struct{ point, normal, forward, up, across maths.Vec }

// sealAnchorStep is how far along a direction a rest point is moved to
// find where that direction goes in the pose, in metres.
const sealAnchorStep = .01

// sealProbe is how far outside the skin, in metres, a landmark's ray
// starts.
const sealProbe = .5

// anchor finds where a ray first meets the rest skin and carries that
// landmark, and the directions at it, to the pose.
func (rig *SealRig) anchor(pose SealRigPose, origin, direction maths.Vec) (sealAnchor, error) {
	rest, normal, found := rig.restHit(origin, direction)
	if !found {
		return sealAnchor{}, fmt.Errorf("harbor seal: no skin at landmark %v", origin)
	}
	at := rig.HeadPoint(pose, rest)
	toward := func(direction maths.Vec) maths.Vec {
		moved := rig.HeadPoint(pose, rest.Add(direction.Scale(sealAnchorStep)))
		return moved.Sub(at).Unit()
	}
	return sealAnchor{at, toward(normal), toward(maths.V(1, 0, 0)),
		toward(maths.Up), toward(maths.V(0, 0, 1))}, nil
}

// sealMolds are the five materials the small anatomy is made in.
type sealMolds struct{ cornea, rim, dark, hair, horn sealMold }

// Parts makes what stands on or in the skin as meshes in the posed
// space: corneas and their lids, ear openings and nostrils with their
// lips, the umbilicus, every whisker as its own strand, and the claws.
// Each part's root lies inside the skin it grows from.
func (rig *SealRig) Parts(pose SealRigPose, look SealLook) ([]thing.Thing, error) {
	if err := pose.check(); err != nil {
		return nil, err
	}
	if err := look.check(); err != nil {
		return nil, err
	}
	var molds sealMolds
	for _, side := range [...]float64{-1, 1} {
		for _, build := range []func(SealRigPose, float64, SealLook, *sealMolds) error{
			rig.eye, rig.ear, rig.nostril, rig.whiskers, rig.brow,
		} {
			if err := build(pose, side, look, &molds); err != nil {
				return nil, err
			}
		}
	}
	if err := rig.umbilicus(pose, &molds); err != nil {
		return nil, err
	}
	rig.claws(rig.posed(pose), &molds)
	return molds.things(look)
}

// things makes each mold a thing in its material.
func (molds *sealMolds) things(look SealLook) ([]thing.Thing, error) {
	skin := thing.Material{Name: "seal lip", Colour: maths.RGB(.075, .062, .052),
		Gloss: .16 + .2*look.Wet, Satin: .3}
	entries := []struct {
		mold     *sealMold
		material thing.Material
	}{
		{&molds.cornea, thing.Material{Name: "harbor seal eye",
			Colour: maths.RGB(.010, .008, .007), Gloss: .55, Satin: .28}},
		{&molds.rim, skin},
		{&molds.dark, thing.Material{Name: "seal recess",
			Colour: maths.RGB(.006, .005, .005), Gloss: .10, Satin: .1}},
		{&molds.hair, thing.Material{Name: "seal whisker",
			Colour: maths.RGB(.44, .37, .28), Gloss: .21, Satin: .4}},
		{&molds.horn, thing.Material{Name: "seal claw",
			Colour: maths.RGB(.085, .072, .058), Gloss: .30, Satin: .25}},
	}
	var parts []thing.Thing
	for _, entry := range entries {
		made, err := entry.mold.mesh()
		if err != nil {
			return nil, fmt.Errorf("harbor seal %s: %w", entry.material.Name, err)
		}
		if made != nil {
			material := entry.material
			parts = append(parts, thing.Thing{Shape: made, Material: &material,
				Motion: thing.Still(maths.Local())})
		}
	}
	return parts, nil
}

// The eye. It sits where a ray from the side meets the head at this
// height, faces forward and outward, and is ringed by a lid.
const (
	sealEyeX, sealEyeY = .73, .110 // chosen: where the head is widest at the brow
	sealEyeForward     = .55       // chosen: share of the forward axis in its gaze
	sealCorneaRound    = .0195     // chosen: the cornea's radius in metres
	sealCorneaDepth    = .0085     // chosen: its depth, from the sunk centre out
	sealCorneaSunk     = .0050     // chosen: how far its centre lies under the skin
	sealLidRound       = .0235     // chosen: the lid ring's radius
	sealLidTube        = .0028     // chosen: the lid ring's thickness
)

// eye sets the eye of one side into the skin at its anchor on the head;
// side is the sign of z.
func (rig *SealRig) eye(pose SealRigPose, side float64, _ SealLook, molds *sealMolds) error {
	anchor, err := rig.anchor(pose,
		maths.V(sealEyeX, sealEyeY, side*sealProbe), maths.V(0, 0, -side))
	if err != nil {
		return err
	}
	gaze := anchor.normal.Add(anchor.forward.Scale(sealEyeForward)).Unit()
	turn := sealFrame(gaze, anchor.up)
	molds.cornea.ellipsoid(anchor.point.Sub(gaze.Scale(sealCorneaSunk)), turn,
		maths.V(sealCorneaRound, sealCorneaRound, sealCorneaDepth), 20)
	molds.rim.lip(anchor.point.Sub(gaze.Scale(sealLidTube/2)), turn,
		sealLidRound, sealLidRound, sealLidTube)
	return nil
}

// The ear is an opening with a raised lip and no pinna. All chosen.
const (
	sealEarX, sealEarY = .655, .112
	sealEarLong        = .0105
	sealEarTall        = .0075
	sealEarTube        = .0020
)

// ear sets the ear of one side at its anchor on the head; side is the
// sign of z.
func (rig *SealRig) ear(pose SealRigPose, side float64, _ SealLook, molds *sealMolds) error {
	anchor, err := rig.anchor(pose,
		maths.V(sealEarX, sealEarY, side*sealProbe), maths.V(0, 0, -side))
	if err != nil {
		return err
	}
	turn := sealFrame(anchor.normal, anchor.forward)
	molds.rim.lip(anchor.point.Sub(anchor.normal.Scale(sealEarTube/4)), turn,
		sealEarLong, sealEarTall, sealEarTube)
	molds.dark.ellipsoid(anchor.point.Sub(anchor.normal.Scale(sealEarTube)), turn,
		maths.V(sealEarLong-sealEarTube/2, sealEarTall-sealEarTube/2, sealEarTube), 10)
	return nil
}

// The nostrils are two slits that meet below, a V seen from the front.
// A slit's lips close it; its length is fixed. All chosen.
const (
	sealNoseProbe      = 1.1   // x, in metres, from which a ray finds the nose
	sealNoseY          = .067  // height of the nostrils
	sealNoseApart      = .022  // each nostril's distance from the midline
	sealNostrilSlant   = .70   // radians above level the outer end sits
	sealNostrilLong    = .0085 // half the slit's length
	sealNostrilTube    = .0008 // the lip's thickness
	sealNostrilOpening = .0020 // the lip's added half-width when fully open
	// sealNostrilShutGap is the least the lip's half-width stands above
	// its thickness, a tenth of a millimetre: at the thickness the ring's
	// inner side is a line and its faces lie on one another.
	sealNostrilShutGap = .0001
)

// nostril sets the nostril of one side at the nose, as the look's share of
// nostrils says; side is the sign of z.
func (rig *SealRig) nostril(pose SealRigPose, side float64, look SealLook, molds *sealMolds) error {
	anchor, err := rig.anchor(pose,
		maths.V(sealNoseProbe, sealNoseY, side*sealNoseApart), maths.V(-1, 0, 0))
	if err != nil {
		return err
	}
	slit := anchor.up.Scale(math.Sin(sealNostrilSlant)).
		Add(anchor.across.Scale(side * math.Cos(sealNostrilSlant)))
	turn := sealFrame(anchor.normal, slit)
	half := math.Max(sealNostrilTube+sealNostrilOpening*(1-look.Nostrils),
		sealNostrilTube+sealNostrilShutGap)
	molds.rim.lip(anchor.point.Sub(anchor.normal.Scale(sealNostrilTube/2)), turn,
		sealNostrilLong, half, sealNostrilTube)
	molds.dark.ellipsoid(anchor.point.Sub(anchor.normal.Scale(sealNostrilTube*1.6)), turn,
		maths.V(sealNostrilLong-sealNostrilTube/2, half-sealNostrilTube/2+sealNostrilTube/4,
			sealNostrilTube*1.5), 10)
	return nil
}

// The umbilicus is a small scar on the belly, a raised ring round a pit.
// All chosen.
const (
	sealNavelX    = -.09
	sealNavelRing = .0085
	sealNavelTube = .0022
	sealNavelPit  = .0050
)

// umbilicus sets the navel on the belly at its anchor, facing down.
func (rig *SealRig) umbilicus(pose SealRigPose, molds *sealMolds) error {
	anchor, err := rig.anchor(pose, maths.V(sealNavelX, -sealProbe, 0), maths.Up)
	if err != nil {
		return err
	}
	turn := sealFrame(anchor.normal, anchor.forward)
	molds.rim.lip(anchor.point.Sub(anchor.normal.Scale(sealNavelTube/2)), turn,
		sealNavelRing, sealNavelRing, sealNavelTube)
	molds.dark.ellipsoid(anchor.point.Sub(anchor.normal.Scale(sealNavelTube)), turn,
		maths.V(sealNavelPit, sealNavelPit, sealNavelTube), 10)
	return nil
}
