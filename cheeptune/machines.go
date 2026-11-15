package cheeptune

import "sort"

// The machine seam.
//
// A machine is one answer to four questions about a voicing's channels:
// which kinds there are, whether a channel written for it keeps its rules,
// whether a channel can sound an event at all, and how a plan becomes
// samples. A voicing names its machine, and everything that used to ask
// "is this a pulse" asks the machine.
//
// A third machine is a new file holding a type with these methods, and one
// line in the machines table below. Nothing in the reader, the binder or
// the renderer names a machine.
//
//	gameboy  gameboy.go, machine.go: four channels, simulated at the
//	         register, a tick at a time. Its output is pinned by hash.
//	studio   studio.go and the engine files beside it: up to sixteen
//	         channels, five engines, real chords, a reverb and a delay.
type machine interface {
	// kinds are the channel kinds the machine has, as a voicing writes
	// them.
	kinds() []string
	// check is every rule the machine keeps on a voicing's channels and
	// on the voicing's own fields. It says how many channels play each
	// role, so the voicing can check what is dropped.
	check(v *Voicing) (errs []error, plays map[string]int)
	// fits is whether a channel can sound an event.
	fits(c *Channel, e event) error
	// percussive is whether a kind plays hits and not notes.
	percussive(kind string) bool
	// render runs the machine over a plan.
	render(pl *Plan) (*Piece, error)
}

// machines is every machine there is, by the name a voicing gives it.
var machines = map[string]machine{
	"gameboy": gameboy{},
	"studio":  studio{},
}

// Machines are the names a voicing's machine may take, in order.
var Machines = machineNames()

// machineNames is the table's keys, sorted.
func machineNames() []string {
	names := make([]string, 0, len(machines))
	for name := range machines {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Percussive is whether the channel plays hits and not notes, on whatever
// machine the voicing names. A picture of a piece asks it to know which
// channels go in the hit lanes.
func (v *Voicing) Percussive(channel int) bool {
	m, ok := machines[v.Machine]
	if !ok || channel < 0 || channel >= len(v.Channels) {
		return false
	}
	return m.percussive(v.Channels[channel].Kind)
}
