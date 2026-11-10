package gpu

import (
	"time"

	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
)

type flamePart struct {
	fire       flame.Fire
	offset     maths.Vec
	at, period time.Duration
	glow       []maths.Light
}

type flameDescription struct {
	parts     []flamePart
	composite bool
}

// flameField recognizes paired fields without replacing custom textures.
func flameField(grid boxed) (flameDescription, bool) {
	density, composite, ok := flamePieces(grid.med)
	if !ok || grid.paint != nil {
		return flameDescription{}, false
	}
	emitting, shared, ok := glowingPieces(grid.shine)
	if !ok || shared != composite || len(density) != len(emitting) {
		return flameDescription{}, false
	}
	field := flameDescription{composite: composite}
	for i, body := range density {
		part, ok := pairedFlame(body, emitting[i])
		if !ok {
			return flameDescription{}, false
		}
		field.parts = append(field.parts, part)
	}
	return field, true
}

// flamePieces unwraps the supported single and composite fields.
func flamePieces(med medium.Medium) ([]flame.Piece, bool, bool) {
	switch body := med.(type) {
	case *flame.Body:
		if body != nil {
			return []flame.Piece{{Body: body}}, false, true
		}
	case *flame.Field:
		if body != nil {
			pieces := body.Pieces()
			return pieces, true, len(pieces) > 0 &&
				len(pieces) <= 64
		}
	}
	return nil, false, false
}

// glowingPieces finds the field identified by the emission texture.
func glowingPieces(shine texture.Texture) ([]flame.Piece, bool, bool) {
	if light, ok := shine.(interface{ Flame() *flame.Body }); ok &&
		light.Flame() != nil {
		return []flame.Piece{{Body: light.Flame()}}, false, true
	}
	if light, ok := shine.(interface{ Flames() *flame.Field }); ok &&
		light.Flames() != nil {
		pieces := light.Flames().Pieces()
		return pieces, true, len(pieces) > 0 && len(pieces) <= 64
	}
	return nil, false, false
}

// pairedFlame requires matching placement, model and sampling time.
func pairedFlame(density, emitting flame.Piece) (flamePart, bool) {
	if density.Body == nil || emitting.Body == nil ||
		density.At != emitting.At {
		return flamePart{}, false
	}
	fire, at, period, _ := density.Body.Samples()
	other, when, puff, glow := emitting.Body.Samples()
	if fire != other || at != when || period != puff || period <= 0 ||
		len(glow) != 256 {
		return flamePart{}, false
	}
	return flamePart{fire: fire, offset: density.At, at: at,
		period: period,
		glow:   glow}, true
}

// flameBake keeps one deferred grid and its reserved offsets in boxes.
type flameBake struct {
	grid           boxed
	density, shine int
}

// deferFlame reserves paired grids so later volumes retain their offsets.
func (encoded *scene) deferFlame(grid boxed) (int, int, bool) {
	if _, ok := flameField(grid); !ok {
		return -1, -1, false
	}
	first := len(encoded.boxes.bytes) / 4
	shine := first + grid.count()
	encoded.boxes.bytes = append(encoded.boxes.bytes, make([]byte,
		grid.count()*16)...)
	encoded.flames = append(encoded.flames, flameBake{grid: grid,
		density: first,
		shine:   shine})
	return first, shine, true
}
