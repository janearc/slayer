//go:build draw

package cheeptune

import (
	"bytes"
	"testing"
)

// noisy is a Game Boy noise channel with a low hit and a high one.
const noisy = `{"kind": "noise", "role": "perc", "hits": {
	"k": {"shift": 7, "divisor": 3, "decay": 0.2},
	"h": {"shift": 1, "divisor": 0, "decay": 0.2}}}`

func TestTheStudioSoundsEveryHitOfAStack(t *testing.T) {
	both := render(t, perc("(k h) -:3"), studioVoicing(kit, ""))
	s := newSpectrum(seconds(both, 0.06, 0.1), both.Rate)
	low, high := power(s, 30, 120), power(s, 9000, 15000)
	alone := render(t, perc("k -:3"), studioVoicing(kit, ""))
	s = newSpectrum(seconds(alone, 0.06, 0.1), alone.Rate)
	if kick := power(s, 30, 120); low < kick-1 {
		t.Errorf("the stack's kick is %.1f dB and alone it is %.1f", low,
			kick)
	}
	// the anti-test: the kick alone has no hat over it, and the stack
	// has.
	if hat := power(s, 9000, 15000); high < hat+20 {
		t.Errorf("the stack's top is %.1f dB and the kick's alone %.1f",
			high, hat)
	}
}

func TestTheGameBoyPlaysTheFirstHitWrittenInAStack(t *testing.T) {
	v := voicing(noisy, "")
	stack := render(t, perc("(k h) -:3"), v)
	first := render(t, perc("k -:3"), v)
	if !bytes.Equal(pcmBytes(stack.Left), pcmBytes(first.Left)) {
		t.Errorf("(k h) on the Game Boy is not k")
	}
	other := render(t, perc("(h k) -:3"), v)
	second := render(t, perc("h -:3"), v)
	if !bytes.Equal(pcmBytes(other.Left), pcmBytes(second.Left)) {
		t.Errorf("(h k) on the Game Boy is not h")
	}
	// the anti-test: the two hits are different sounds, so the
	// comparison can tell which was played.
	if bytes.Equal(pcmBytes(first.Left), pcmBytes(second.Left)) {
		t.Errorf("k and h are the same sound")
	}
	// and a hit the channel lacks is refused though it would not be
	// the one played.
	s, _ := ParseScore(perc("(k t) -:3"))
	gb, _ := ParseVoicing(v)
	_, err := Bind(s, gb)
	wantErr(t, err, `hit "t" is not one of h, k`)
}
