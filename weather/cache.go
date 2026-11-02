package weather

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// version is the solver's, in every cache key: a change to what the
// solver computes changes it, so no weather made by an older one is
// read as this one's.
const version = "weather 4: barotropic, arakawa, ssp-rk3, ∇⁸, ∇⁴ bands, " +
	"age renewed a five-hundredth a storm, upwelling 20 by the row's mean"

// key is what names the weather in the cache: everything that shapes
// it, the gpu's float32 apart from the cpu's float64, hashed.
func (made *Weather) key(onGPU bool) string {
	grid, stir := made.grid, made.stirring
	said := fmt.Sprintf("%s|%d %d %v %v|%v %v %v|%v %v %v|%d|%v|%v %v %v %v",
		version, grid.across, grid.rows, grid.limit, grid.radius,
		made.spin, made.drag, made.relax, stir.rate, stir.radius,
		stir.strength, made.seed, onGPU, smoothing, courant, longest,
		filtered)
	sum := sha256.Sum256([]byte(said))
	return hex.EncodeToString(sum[:12])
}

// magic opens every checkpoint, so a file that is not one is refused.
const magic = "slayer weather checkpoint\n"

// Reach carries the weather to a moment, so many seconds since it
// began, or the first step past it, by way of the cache in a directory,
// and is how many steps it took.
//
// it starts from the latest checkpoint of the same weather at or before
// that moment, if there is one, and goes on from there, leaving a
// checkpoint every so many seconds and at the moment itself.
//
// so the next shot of this weather, or a later frame of it, starts where
// this one stopped.
func (made *Weather) Reach(dir string, moment, every float64,
	onGPU bool) (int, error) {
	shelf := filepath.Join(dir, made.key(onGPU))
	if err := os.MkdirAll(shelf, 0o755); err != nil {
		return 0, err
	}
	if reached, err := os.ReadFile(note(shelf, moment)); err == nil {
		return 0, made.load(filepath.Join(shelf, string(reached)))
	}
	if err := made.resume(shelf, moment); err != nil {
		return 0, err
	}
	taken := 0
	for made.clock < moment {
		next := math.Min(moment, (math.Floor(made.clock/every)+1)*every)
		steps, err := made.runTo(next, onGPU)
		taken += steps
		if err != nil {
			return taken, err
		}
		if err := made.save(shelf); err != nil {
			return taken, err
		}
	}
	return taken, os.WriteFile(note(shelf, moment), []byte(name(made.clock)),
		0o644)
}

// note is where a shelf says which checkpoint answers a moment: the
// first step past it, which a checkpoint's own name, its clock, does
// not say.
func note(shelf string, moment float64) string {
	return filepath.Join(shelf, fmt.Sprintf("%016.0f.at", moment))
}

// runTo runs on the gpu or the cpu as asked.
func (made *Weather) runTo(until float64, onGPU bool) (int, error) {
	if onGPU {
		return made.RunOnGPU(until)
	}
	return made.Run(until), nil
}

// name is a checkpoint's file name for a moment, so they sort by it.
func name(clock float64) string {
	return fmt.Sprintf("%016.0f.wx", clock)
}

// resume loads the latest checkpoint at or before a moment from a
// shelf, if the weather has not already gone further; with none there,
// the weather is left as it is.
func (made *Weather) resume(shelf string, moment float64) error {
	entries, err := os.ReadDir(shelf)
	if err != nil {
		return err
	}
	var moments []float64
	for _, entry := range entries {
		clock, err := strconv.ParseFloat(
			strings.TrimSuffix(entry.Name(), ".wx"), 64)
		if err == nil && strings.HasSuffix(entry.Name(), ".wx") &&
			clock <= moment && clock > made.clock {
			moments = append(moments, clock)
		}
	}
	if len(moments) == 0 {
		return nil
	}
	return made.load(filepath.Join(shelf, name(slices.Max(moments))))
}

// save writes the weather as it is to a checkpoint on a shelf, by way of
// a file beside it renamed into place, so a run stopped halfway leaves
// no half a checkpoint.
func (made *Weather) save(shelf string) error {
	final := filepath.Join(shelf, name(made.clock))
	partial, err := os.CreateTemp(shelf, ".partial-*")
	if err != nil {
		return err
	}
	defer os.Remove(partial.Name())
	head := append([]byte(magic), make([]byte, 32)...)
	binary.LittleEndian.PutUint64(head[len(magic):],
		math.Float64bits(made.clock))
	binary.LittleEndian.PutUint64(head[len(magic)+8:], uint64(made.drawn))
	binary.LittleEndian.PutUint64(head[len(magic)+16:],
		uint64(made.grid.cells()))
	binary.LittleEndian.PutUint64(head[len(magic)+24:],
		math.Float64bits(made.next))
	body := make([]byte, 0, 8*len(made.fields())*made.grid.cells())
	for _, field := range made.fields() {
		for _, value := range field {
			body = binary.LittleEndian.AppendUint64(body,
				math.Float64bits(value))
		}
	}
	if _, err := partial.Write(append(head, body...)); err != nil {
		partial.Close()
		return err
	}
	if err := partial.Close(); err != nil {
		return err
	}
	return os.Rename(partial.Name(), final)
}

// load reads a checkpoint into the weather, refused if it is not one or
// holds another grid's cells.
func (made *Weather) load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	head := make([]byte, len(magic)+32)
	if _, err := io.ReadFull(file, head); err != nil {
		return err
	}
	if string(head[:len(magic)]) != magic {
		return errors.New(path + " is not a weather checkpoint")
	}
	at := head[len(magic):]
	cells := int(binary.LittleEndian.Uint64(at[16:]))
	if cells != made.grid.cells() {
		return fmt.Errorf("%s holds %d cells, not %d", path, cells,
			made.grid.cells())
	}
	kept := made.fields()
	body := make([]byte, 8*len(kept)*cells)
	if _, err := io.ReadFull(file, body); err != nil {
		return err
	}
	for f, field := range kept {
		for k := range cells {
			field[k] = math.Float64frombits(
				binary.LittleEndian.Uint64(body[8*(f*cells+k):]))
		}
	}
	made.clock = math.Float64frombits(binary.LittleEndian.Uint64(at))
	made.drawn = int64(binary.LittleEndian.Uint64(at[8:]))
	made.next = math.Float64frombits(binary.LittleEndian.Uint64(at[24:]))
	return nil
}
