package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// GroundRig keeps the established factory geometry in mechanical groups.
// Foundations, service buildings and piers stay fixed. The upper enclosure
// and fork turn in azimuth; the optics turn about their primary pivot.
// This is a scenery rig, not a swept-clearance engineering model.
type GroundRig struct {
	Fixed, Azimuth, Optical, Enclosure []thing.Thing
	Pivot                              maths.Vec
	Altitude                           float64
	Launchers                          []maths.Subspace
}

// MakeGroundRig separates parts before material flattening loses their roles.
// Only the ordinary alt-az ground families have this acquisition rig.
func MakeGroundRig(name, detail string) (*GroundRig, error) {
	return makeGroundRig(name, detail, false)
}

// MakeGroundRigWithLaunchers adds authored guide tubes and their exit poses
// to the optical assembly. The unchanged catalogue rig remains available
// through MakeGroundRig; survey cameras receive no fictional guide hardware.
func MakeGroundRigWithLaunchers(name, detail string) (*GroundRig, error) {
	return makeGroundRig(name, detail, true)
}

// makeGroundRig builds the ground rig named name at the detail given, quick
// or demo. It fails for a name that is not a ground observatory, or whose
// row is equatorial. With withLaunchers it also adds the film launchers,
// unless the row is a survey.
func makeGroundRig(name, detail string, withLaunchers bool) (*GroundRig, error) {
	maker, ok := groundRows()[name]
	row, ground := maker.(groundObservatory)
	if !ok || !ground || row.equatorial {
		return nil, fmt.Errorf("no acquisition rig for %q", name)
	}
	if detail != "quick" && detail != "demo" {
		return nil, fmt.Errorf("rig detail must be quick or demo")
	}
	assembly := &instrumentAssembly{fine: cut{around: int(groundValue("sides-" + detail))}, object: good{dressed: groundLooks()}}
	row.enclosure(assembly)
	row.enclosureDetails(assembly)
	enclosure := len(assembly.object.parts)
	row.forkMount(assembly)
	mount := len(assembly.object.parts)
	pose := row.opticalFrame()
	row.optics(assembly, pose)
	row.opticsDetails(assembly, pose)
	var launchers []maths.Subspace
	if withLaunchers && row.survey == nil {
		// Film launchers sit on brackets rooted in the primary truss rim.
		// Their exits and optical bodies share this same construction frame.
		rim := row.aperture * groundValue("truss-rim")
		length := launcherValue("length")
		radius := launcherValue("radius")
		for _, sign := range []float64{-1, 1} {
			base := maths.V(sign*(rim+radius*2), 0, 0)
			launch := maths.NewSubspace(base, maths.Identity, 1).Within(pose)
			assembly.add(groundDrum(radius, radius-launcherValue("wall"), length), launch, leaf)
			for _, along := range []float64{.15, .65} {
				root := pose.Out(maths.V(sign*(rim-radius), along*radius, along*radius))
				foot := launch.Out(maths.V(0, length*along, 0))
				assembly.rod(root, foot, radius*.35, spaceStructure)
				collar := maths.NewSubspace(maths.V(0, length*along, 0), maths.Identity, 1).Within(launch)
				assembly.add(groundDrum(radius*1.3, radius, radius*.6), collar, spaceStructure)
			}
			launch.Origin = launch.Out(maths.V(0, length, 0))
			launchers = append(launchers, launch)
		}
	}
	if assembly.trouble != nil {
		return nil, assembly.trouble
	}
	rig := &GroundRig{Pivot: pose.Origin, Altitude: groundValue("altitude-deg"), Launchers: launchers}
	// forkMount's first box is the fixed pedestal, followed by the fork
	// and the cross-bearing. Keep it with the house, never with the yaw.
	groups := make([][]part, 4)
	for i, piece := range assembly.object.parts {
		role := 0
		if i > enclosure && i < mount {
			role = 1
		}
		if i >= mount {
			role = 2
		}
		if i < enclosure && piece.rigTurn {
			role = 3
		}
		groups[role] = append(groups[role], piece)
	}
	for i, parts := range groups {
		group := good{parts: parts, dressed: assembly.object.dressed}
		things, err := group.Things(maths.Local())
		if err != nil {
			return nil, err
		}
		*([]*[]thing.Thing{&rig.Fixed, &rig.Azimuth, &rig.Optical, &rig.Enclosure}[i]) = things
	}
	return rig, nil
}

// turnSince marks a mechanically rotating shell before material flattening.
func (assembly *instrumentAssembly) turnSince(first int) {
	for i := first; i < len(assembly.object.parts); i++ {
		assembly.object.parts[i].rigTurn = true
	}
}
