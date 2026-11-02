//go:build darwin

package weather

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"

	"github.com/janearc/slayer/metal"
)

//go:embed weather.metal
var kernels string

// a device is the weather's state on the gpu, in float32 on a sphere of
// radius one: each field, its stage and its change; the stream function
// and its wavenumbers; and the laplacians the fields are smoothed by.
type device struct {
	session                    *metal.Session
	now, stage, change         [fields]metal.Buffer
	psi, modes, ahead, means   metal.Buffer
	thrice, twice, once, older metal.Buffer
}

// onDevice opens a session with the weather's kernels and its buffers.
func (made *Weather) onDevice() (*device, error) {
	if made.grid.across > 2048 {
		return nil, fmt.Errorf("the gpu holds rows of 2048 cells at most, "+
			"not %d", made.grid.across)
	}
	session, err := metal.OpenSession(kernels, "laplacian", "tendency",
		"combine", "forward", "back", "sweep", "filter", "means")
	if err != nil {
		return nil, err
	}
	cells := made.grid.cells()
	on := &device{session: session}
	wanted := []*metal.Buffer{&on.psi, &on.ahead, &on.thrice, &on.twice,
		&on.once, &on.older, &on.means}
	for f := range fields {
		wanted = append(wanted, &on.now[f], &on.stage[f], &on.change[f])
	}
	for _, into := range wanted {
		if *into, err = session.Buffer(cells); err != nil {
			session.Close()
			return nil, err
		}
	}
	if on.modes, err = session.Buffer(2 * cells); err != nil {
		session.Close()
		return nil, err
	}
	return on, nil
}

// sky is the kernels' sky struct for a step and its weights, packed as
// weather.metal lays it out.
func (made *Weather) sky(step, keep, weigh float64) []byte {
	grid := made.grid
	r := grid.radius
	packed := binary.LittleEndian.AppendUint32(nil, uint32(grid.across))
	packed = binary.LittleEndian.AppendUint32(packed, uint32(grid.rows))
	packed = binary.LittleEndian.AppendUint32(packed,
		uint32(bits.TrailingZeros(uint(grid.across))))
	packed = binary.LittleEndian.AppendUint32(packed, 0)
	for _, value := range []float64{grid.limit, grid.rise(), grid.run(),
		made.spin, made.drag, made.relax,
		made.smoothed(4) / math.Pow(r, 8), made.smoothed(2) / math.Pow(r, 4),
		step, keep, weigh, math.Cos(filtered), upwelling, fresh} {
		packed = binary.LittleEndian.AppendUint32(packed,
			math.Float32bits(float32(value)))
	}
	return packed
}

// laplacian adds a laplacian of one buffer into another to the batch.
func (on *device) laplacian(made *Weather, params []byte, from,
	into metal.Buffer) error {
	return on.session.Dispatch("laplacian", made.grid.across,
		made.grid.rows, params, from, into)
}

// stream adds the stream function of a vorticity to the batch: each row
// forward, the sweep across the rows, each row back.
func (on *device) stream(made *Weather, params []byte,
	vorticity metal.Buffer) error {
	grid, s := made.grid, on.session
	if err := s.Grouped("forward", grid.across/2, grid.rows, params,
		vorticity, on.modes); err != nil {
		return err
	}
	if err := s.Dispatch("sweep", grid.across, 1, params, on.modes,
		on.ahead); err != nil {
		return err
	}
	return s.Grouped("back", grid.across/2, grid.rows, params, on.modes,
		on.psi)
}

// stageOf adds to the batch one stage of the step: the change of the
// fields in from, and keep of the step's start and weigh of from
// carried on by it, written into into.
func (on *device) stageOf(made *Weather, from, into [fields]metal.Buffer,
	step, keep, weigh float64) error {
	grid := made.grid
	params := made.sky(step, keep, weigh)
	if err := on.stream(made, params, from[0]); err != nil {
		return err
	}
	for _, pair := range [][2]metal.Buffer{{from[0], on.thrice},
		{on.thrice, on.twice}, {on.twice, on.thrice},
		{from[1], on.once}, {from[2], on.older}} {
		if err := on.laplacian(made, params, pair[0], pair[1]); err != nil {
			return err
		}
	}
	if err := on.session.Grouped("means", grid.across/2, grid.rows, params,
		from[0], on.means); err != nil {
		return err
	}
	if err := on.session.Dispatch("tendency", grid.across, grid.rows,
		params, on.psi, from[0], from[1], from[2], on.thrice, on.once,
		on.older, on.means, on.change[0], on.change[1],
		on.change[2]); err != nil {
		return err
	}
	for f := range fields {
		if err := on.session.Dispatch("combine", grid.cells(), 1, params,
			on.now[f], from[f], on.change[f], into[f]); err != nil {
			return err
		}
	}
	return nil
}

// stepOn is Weather.step on the gpu: the three stages of shu and osher,
// the last written into the state, and the filter, as one batch.
func (on *device) stepOn(made *Weather, seconds float64) error {
	stages := []struct {
		from, into  [fields]metal.Buffer
		keep, weigh float64
	}{{on.now, on.stage, 0, 1},
		{on.stage, on.stage, 0.75, 0.25},
		{on.stage, on.now, 1.0 / 3, 2.0 / 3}}
	for _, one := range stages {
		err := on.stageOf(made, one.from, one.into, seconds, one.keep,
			one.weigh)
		if err != nil {
			return err
		}
	}
	params := made.sky(seconds, 0, 0)
	for f := range fields {
		err := on.session.Grouped("filter", made.grid.across/2,
			made.grid.rows, params, on.now[f])
		if err != nil {
			return err
		}
	}
	return on.session.Commit()
}

// RunOnGPU is Run on the gpu: the weather carried on until so many
// seconds have passed since it began, in float32, and how many steps it
// took. its storms are stamped and its next step measured on the cpu,
// in the memory the two share, between the gpu's steps.
func (made *Weather) RunOnGPU(until float64) (int, error) {
	on, err := made.onDevice()
	if err != nil {
		return 0, err
	}
	defer on.session.Close()
	on.load(made)
	zeta, age := on.now[0].Numbers(), on.now[2].Numbers()
	add := func(k int, spin, core float64) {
		zeta[k] += float32(spin)
		age[k] *= float32(1 - core)
	}
	if made.next == 0 {
		made.next = made.stepFor()
	}
	taken := 0
	for made.clock < until {
		step := made.next
		if err := on.stepOn(made, step); err != nil {
			return taken, err
		}
		made.stir(step, add)
		made.clock += step
		taken++
		made.next = on.stepFor(made)
	}
	on.store(made)
	return taken, nil
}

// load copies the weather's fields onto the device.
func (on *device) load(made *Weather) {
	for f, field := range made.fields() {
		numbers := on.now[f].Numbers()
		for k, value := range field {
			numbers[k] = float32(value)
		}
	}
}

// store copies the device's fields back into the weather.
func (on *device) store(made *Weather) {
	for f, field := range made.fields() {
		for k, value := range on.now[f].Numbers() {
			field[k] = float64(value)
		}
	}
}

// stepFor is Weather.stepFor from the stream function the gpu last
// found, on a sphere of radius one, so its slopes are scaled by the
// radius back to metres a second.
func (on *device) stepFor(made *Weather) float64 {
	psi := make([]float64, made.grid.cells())
	for k, value := range on.psi.Numbers() {
		psi[k] = float64(value)
	}
	fastest := made.fastestIn(made.grid.walled(psi), made.grid.radius)
	if fastest == 0 {
		return longest
	}
	return math.Min(longest, courant*made.cell()/fastest)
}
