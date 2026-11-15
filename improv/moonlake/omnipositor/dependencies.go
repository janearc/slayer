package omnipositor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
)

var ErrUnboundExecutionSnapshot = errors.New("identity: authoritative immutable execution snapshot unavailable: compositor images, request water, lighting configuration, and parsed show are not bound")

type FireSnapshot struct {
	frames [3]lights.FireFrames
	inputs map[string]identity.HashedInput
	hits   []string
}

// Frames returns the snapshot's fire frames with the pixel data copied, so
// the caller cannot change the snapshot.
func (s *FireSnapshot) Frames() [3]lights.FireFrames {
	var out [3]lights.FireFrames
	for sequence, frames := range s.frames {
		out[sequence] = lights.FireFrames{Count: frames.Count, Images: make(map[int]lights.Image, len(frames.Images))}
		for index, image := range frames.Images {
			image.Light = append([]float64(nil), image.Light...)
			out[sequence].Images[index] = image
		}
	}
	return out
}

// CacheHits returns a copy of the names of the fire frames that came from
// the cache when the snapshot was made.
func (s *FireSnapshot) CacheHits() []string { return append([]string(nil), s.hits...) }

// prepareFire loads the fire frames that request needs, within maximum frames
// and budget bytes, and records a hash for the count and for each frame so
// they can be part of an identity. It refuses if another fire request is
// running. The pixel data is not copied.
func (a *LoadedAssets) prepareFire(ctx context.Context, request LightRequest, maximum int, budget int64) (*FireSnapshot, error) {
	if !a.fireRequestMu.TryLock() {
		return nil, fmt.Errorf("%w: asset fire request already active", lights.ErrFireResources)
	}
	defer a.fireRequestMu.Unlock()
	var hits []string
	frames, err := a.loadFireBudgetTracked(ctx, request, false, maximum, budget, &hits)
	if err != nil {
		return nil, err
	}
	snapshot := &FireSnapshot{frames: frames, inputs: make(map[string]identity.HashedInput), hits: hits}
	a.mu.Lock()
	defer a.mu.Unlock()
	for sequence, frames := range frames {
		sequenceName := [3]string{"steady", "swell", "burst"}[sequence]
		snapshot.inputs["fire."+sequenceName+".count"] = identity.HashedInput{SHA256: sourceDigest([]byte(fmt.Sprint(frames.Count)))}
		for index := range frames.Images {
			name := fmt.Sprintf("fire.%s[%d]", sequenceName, index)
			asset := a.identities[name]
			snapshot.inputs[name] = identity.HashedInput{SHA256: asset.SHA256, Choice: fmt.Sprintf("%d/%d", index, frames.Count)}
		}
	}
	sort.Strings(snapshot.hits)
	return snapshot, nil
}

type FrameIdentity struct{}

type FrameDependencies struct {
	Fire *FireSnapshot
}

// sourceDigest is the SHA-256 of b, as lower case hex.
func sourceDigest(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

// NewFrameIdentity does nothing yet: it ignores its arguments and always
// returns nil and ErrUnboundExecutionSnapshot.
func NewFrameIdentity(c Config, l lights.Config, a *LoadedAssets, binarySHA256 string, scoreBytes, cueBytes []byte, backend, device string) (*FrameIdentity, error) {
	return nil, ErrUnboundExecutionSnapshot
}

// Prepare does nothing yet: it always returns empty dependencies and
// ErrUnboundExecutionSnapshot.
func (b *FrameIdentity) Prepare(ctx context.Context, request LightRequest) (FrameDependencies, error) {
	return FrameDependencies{}, ErrUnboundExecutionSnapshot
}
