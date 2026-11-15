package gpumist

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/janearc/slayer/metal"
)

type Sample struct {
	X    int64
	Y    int64
	Seed uint64
}

type Result struct {
	Raw        uint64
	Normalized float32
}

type Device struct {
	mu      sync.Mutex
	session *metal.Session
	input   metal.Buffer
	output  metal.Buffer
}

const (
	inputWords  = 6
	outputWords = 3
	maxSamples  = 1 << 15
)

// New opens a Metal session and allocates the input and output buffers, sized
// for maxSamples samples. The caller must Close the Device it returns.
func New() (*Device, error) {
	session, err := metal.OpenSession(source(), "lattice_hash")
	if err != nil {
		return nil, fmt.Errorf("gpumist: open Metal: %w", err)
	}
	made := &Device{session: session}
	if made.input, err = session.Buffer(maxSamples * inputWords); err != nil {
		session.Close()
		return nil, fmt.Errorf("gpumist: allocate input buffer: %w", err)
	}
	if made.output, err = session.Buffer(maxSamples * outputWords); err != nil {
		session.Close()
		return nil, fmt.Errorf("gpumist: allocate output buffer: %w", err)
	}
	return made, nil
}

// Close ends the session and drops the buffers. It does nothing on a nil or
// an already closed Device.
func (d *Device) Close() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.session != nil {
		d.session.Close()
		d.session = nil
		d.input = metal.Buffer{}
		d.output = metal.Buffer{}
	}
}

// Hash runs the lattice hash on the GPU for each sample and returns the
// results in the same order. Raw is the 64-bit hash; Normalized is the hash
// shifted right by 11 and scaled by 2^-53. It refuses a nil or closed Device
// and more than maxSamples samples; no samples gives an empty result.
func (d *Device) Hash(samples []Sample) ([]Result, error) {
	if d == nil {
		return nil, errors.New("gpumist: device is closed")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.session == nil {
		return nil, errors.New("gpumist: device is closed")
	}
	if len(samples) == 0 {
		return []Result{}, nil
	}
	if len(samples) > maxSamples {
		return nil, fmt.Errorf("gpumist: too many samples: %d > %d", len(samples), maxSamples)
	}
	packSamples(d.input.Numbers(), samples)
	if err := d.session.Dispatch("lattice_hash", len(samples), 1, nil, d.input, d.output); err != nil {
		return nil, fmt.Errorf("gpumist: dispatch lattice_hash: %w", err)
	}
	if err := d.session.Commit(); err != nil {
		return nil, fmt.Errorf("gpumist: commit lattice_hash: %w", err)
	}
	out := make([]Result, len(samples))
	words := d.output.Numbers()
	for i := range out {
		lo := math.Float32bits(words[outputWords*i])
		hi := math.Float32bits(words[outputWords*i+1])
		out[i].Raw = uint64(lo) | uint64(hi)<<32
		out[i].Normalized = words[outputWords*i+2]
	}
	return out, nil
}

// packSamples writes each sample into words as six float32 slots holding the
// raw bits of X, Y and Seed, each as a low half then a high half. words must
// have room for six slots per sample.
func packSamples(words []float32, samples []Sample) {
	for i, sample := range samples {
		base := i * inputWords
		words[base] = math.Float32frombits(uint32(sample.X))
		words[base+1] = math.Float32frombits(uint32(uint64(sample.X) >> 32))
		words[base+2] = math.Float32frombits(uint32(sample.Y))
		words[base+3] = math.Float32frombits(uint32(uint64(sample.Y) >> 32))
		words[base+4] = math.Float32frombits(uint32(sample.Seed))
		words[base+5] = math.Float32frombits(uint32(sample.Seed >> 32))
	}
}

// source returns the Metal source for the lattice_hash kernel, with the four
// 64-bit constants, the shift and the scale written in ahead of it.
func source() string {
	var b strings.Builder
	for _, constant := range []struct {
		name  string
		value uint64
	}{
		{"latticeX", 11400714819323198485},
		{"latticeY", 14029467366897019727},
		{"latticeSeed", 1609587929392839161},
		{"latticeMix", 18397679294719823053},
	} {
		fmt.Fprintf(&b, "constant ulong %s = %dul;\n", constant.name, constant.value)
	}
	fmt.Fprintf(&b, "constant uint latticeShift = %du;\n", 33)
	fmt.Fprintf(&b, "constant float latticeScale = %.17ef;\n", math.Ldexp(1, -53))
	b.WriteString("kernel void lattice_hash(device const uint *in [[buffer(0)]], device uint *out [[buffer(1)]], uint gid [[thread_position_in_grid]]) { uint i = gid * 6u; ulong x = ((ulong)in[i + 1u] << 32) | (ulong)in[i]; ulong y = ((ulong)in[i + 3u] << 32) | (ulong)in[i + 2u]; ulong seed = ((ulong)in[i + 5u] << 32) | (ulong)in[i + 4u]; ulong h = x * latticeX ^ y * latticeY ^ seed * latticeSeed; h ^= h >> latticeShift; h *= latticeMix; h ^= h >> latticeShift; out[gid * 3u] = (uint)h; out[gid * 3u + 1u] = (uint)(h >> 32); float normalized = float(h >> 11u) * latticeScale; out[gid * 3u + 2u] = as_type<uint>(normalized); }\n")
	return b.String()
}
