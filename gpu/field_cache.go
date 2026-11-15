//go:build darwin

package gpu

import (
	"fmt"
	"sync"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

type fieldSeedKey struct {
	eye, space                 maths.Subspace
	mass, spin, reach, eyeTime float64
	velocity                   maths.Vec
	pinhole                    film.Pinhole
	lensType, lensKey          string
	width, height              int
}

type fieldSeedCache struct {
	mu    sync.Mutex
	ready bool
	key   fieldSeedKey
	rays  []byte
}

// seedKey accepts built-in pinholes and lenses with explicit ray identities.
func seedKey(input fieldSeedInput) (fieldSeedKey, bool) {
	source := input.source
	key := fieldSeedKey{eye: input.eye, space: input.space,
		mass: source.Mass, spin: source.Spin, reach: source.Reach,
		eyeTime: source.EyeTime, velocity: source.Eye,
		width: input.width, height: input.height}
	switch lens := input.lens.(type) {
	case nil:
	case film.Pinhole:
		key.pinhole = lens
	case film.IdentifiedSensor:
		key.lensKey = lens.RayKey()
		if key.lensKey == "" {
			return key, false
		}
		key.lensType = fmt.Sprintf("%T", lens)
	default:
		return key, false
	}
	return key, true
}

// cachedFieldSeeds keeps one immutable tape. Concurrent callers may retain an
// older tape while another call replaces the cache; its bytes are not reused.
func (renderer *Renderer) cachedFieldSeeds(encoded *scene, input fieldSeedInput) bool {
	key, known := seedKey(input)
	if !known {
		return fieldSeeds(encoded, input)
	}
	cache := &renderer.seeds
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.ready && cache.key == key {
		encoded.rays.bytes = cache.rays
		return true
	}
	fieldSeeds(encoded, input)
	cache.key, cache.rays, cache.ready = key, encoded.rays.bytes, true
	return false
}
