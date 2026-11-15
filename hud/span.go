package hud

import "time"

// Span is the stretch of film time an element is drawn in: from its
// start, which it is drawn at, until its end, which it is not. A span
// with no start begins with the film and one with no end lasts to its
// last frame. The zero Span is both: an element with it is always drawn.
//
// An element outside its span lays out no cells at all, so the picture
// under it is left as it was and no role of its is asked of the theme.
type Span struct {
	from, until       time.Duration
	hasFrom, hasUntil bool
}

// Between is the span from one time until a later one. A span that ends
// at or before its start holds no time and is refused by the caller that
// built it; here it is the empty span, which holds nothing.
func Between(from, until time.Duration) Span {
	return Span{from: from, until: until, hasFrom: true, hasUntil: true}
}

// From is the span that starts at a time and never ends.
func From(from time.Duration) Span {
	return Span{from: from, hasFrom: true}
}

// Until is the span that has always begun and ends at a time.
func Until(until time.Duration) Span {
	return Span{until: until, hasUntil: true}
}

// Holds is whether an element with this span is drawn at the time.
func (made Span) Holds(at time.Duration) bool {
	if made.hasFrom && at < made.from {
		return false
	}
	return !made.hasUntil || at < made.until
}
