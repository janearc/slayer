package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/draw"
	"image/png"

	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"
)

var errVerifiedPNGReused = errors.New("omnipositor: published frame was reused before verification")
var errInvalidRenderedFrame = errors.New("omnipositor: invalid rendered frame")

// verifiedPNG is verifiedPNGWithHook with no hook before publishing.
func verifiedPNG(ctx context.Context, store *identity.Store, key identity.Key, redraw bool, render func() (*image.RGBA, error)) (*image.RGBA, error) {
	return verifiedPNGWithHook(ctx, store, key, redraw, nil, render)
}

// verifiedPNGWithHook returns the frame for key. If the store holds a verified
// PNG and redraw is false, it decodes and returns that. Otherwise it calls
// render, checks and encodes the image, calls beforePublish if there is one,
// and publishes it. A missing entry is always rendered; a conflicting or
// corrupt one only when redraw is set. It fails if the store reports the entry
// as already reused, and it checks ctx between steps.
func verifiedPNGWithHook(ctx context.Context, store *identity.Store, key identity.Key, redraw bool, beforePublish func() error, render func() (*image.RGBA, error)) (*image.RGBA, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := store.VerifyContext(ctx, key)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err == nil && !redraw {
		im, err := decodeRGBA(data)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return im, nil
	}
	if err != nil && !errors.Is(err, identity.ErrMissing) && !(redraw && (errors.Is(err, identity.ErrConflict) || errors.Is(err, identity.ErrCorrupt))) {
		return nil, err
	}
	im, err := render()
	if err != nil {
		return nil, err
	}
	if im == nil {
		return nil, errInvalidRenderedFrame
	}
	if err := validateRenderedRGBA(im); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, im); err != nil {
		return nil, err
	}
	if beforePublish != nil {
		if err := beforePublish(); err != nil {
			return nil, err
		}
	}
	reused, err := store.PublishContext(ctx, key, encoded.Bytes(), redraw)
	if err != nil {
		return nil, err
	}
	if reused {
		return nil, errVerifiedPNGReused
	}
	return im, nil
}

// decodeRGBA decodes PNG data and copies it into a new RGBA image that has the
// bounds of the decoded one.
func decodeRGBA(data []byte) (*image.RGBA, error) {
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	im := image.NewRGBA(decoded.Bounds())
	draw.Draw(im, im.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	return im, nil
}

// validateRenderedRGBA refuses a frame with empty bounds, a width or height
// too large for an int, or a stride or pixel slice too short to hold every row
// of four-byte pixels. It returns errInvalidRenderedFrame for each.
func validateRenderedRGBA(im *image.RGBA) error {
	bounds := im.Bounds()
	if bounds.Empty() {
		return errInvalidRenderedFrame
	}
	maxInt := int(^uint(0) >> 1)
	dx, ok := spanWithinInt(bounds.Min.X, bounds.Max.X, maxInt)
	if !ok {
		return errInvalidRenderedFrame
	}
	dy, ok := spanWithinInt(bounds.Min.Y, bounds.Max.Y, maxInt)
	if !ok {
		return errInvalidRenderedFrame
	}
	if im.Stride <= 0 {
		return errInvalidRenderedFrame
	}
	if dx > maxInt/4 {
		return errInvalidRenderedFrame
	}
	row := dx * 4
	if im.Stride < row || len(im.Pix) < row {
		return errInvalidRenderedFrame
	}
	if dy > 1 && im.Stride > (len(im.Pix)-row)/(dy-1) {
		return errInvalidRenderedFrame
	}
	return nil
}

// spanWithinInt returns max minus min and true, or false if max is below min
// or the difference would not fit in an int of largest value maxInt.
func spanWithinInt(min, max, maxInt int) (int, bool) {
	if max < min {
		return 0, false
	}
	if min < 0 && max > maxInt+min {
		return 0, false
	}
	return max - min, true
}
