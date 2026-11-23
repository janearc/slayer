package cheeptune

import "testing"

func voicing(channels string, drop string) []byte {
	return []byte(`{"name": "v", "machine": "gameboy", "channels": [` +
		channels + `], "drop": [` + drop + `]}`)
}

const lead = `{"kind": "pulse", "role": "lead", "duty": 0.5}`

func TestTheMachineHasTwoPulsesAndNoMore(t *testing.T) {
	_, err := ParseVoicing(voicing(lead+`,
		{"kind": "pulse", "role": "bass", "duty": 0.5},
		{"kind": "pulse", "role": "pad", "duty": 0.5}`, ""))
	wantErr(t, err, "channel 3 (pulse): the gameboy has 2 pulse channels")
}

func TestOnlyTheHardwaresDutiesAreAllowed(t *testing.T) {
	_, err := ParseVoicing(voicing(
		`{"kind": "pulse", "role": "lead", "duty": 0.3}`, ""))
	wantErr(t, err, "duty 0.3; the hardware has 0.125, 0.25, 0.5, 0.75")
}

func TestAFieldOnTheWrongKindIsRefused(t *testing.T) {
	_, err := ParseVoicing(voicing(
		`{"kind": "wave", "role": "bass", "table": "triangle", "duty": 0.5}`,
		""))
	wantErr(t, err, "duty is only for a pulse")
	_, err = ParseVoicing(voicing(`{"kind": "noise", "role": "perc",
		"hits": {"k": {"shift": 1, "divisor": 1, "decay": 0.1}},
		"vibrato": {"rate": 5, "depth": 10}}`, ""))
	wantErr(t, err, "envelope, vibrato, arp and detune are for pitched")
}

func TestAnEchoNeedsItsDelayAndGain(t *testing.T) {
	_, err := ParseVoicing(voicing(lead+`,
		{"kind": "pulse", "echo": "lead", "duty": 0.5}`, ""))
	wantErr(t, err, "an echo needs a delay", "an echo needs a gain")
}

func TestAChannelPlaysOrEchoesNotBoth(t *testing.T) {
	_, err := ParseVoicing(voicing(`{"kind": "pulse", "role": "lead",
		"echo": "lead", "delay": "1", "gain": 0.5, "duty": 0.5}`, ""))
	wantErr(t, err, "plays lead and echoes lead; a channel does one")
}

func TestAWaveTableIsANameOrThirtyTwoHexDigits(t *testing.T) {
	v, err := ParseVoicing(voicing(`{"kind": "wave", "role": "bass",
		"table": "0123456789abcdeffedcba9876543210"}`, ""))
	if err != nil {
		t.Fatal(err)
	}
	if got := v.Channels[0].table[15]; got != 15 {
		t.Errorf("step 16 is %d, want 15", got)
	}
	_, err = ParseVoicing(voicing(
		`{"kind": "wave", "role": "bass", "table": "sine"}`, ""))
	wantErr(t, err, `table "sine" is not triangle, saw, square`)
}

func TestAnArpIsTicksOrBeats(t *testing.T) {
	v, err := ParseVoicing(voicing(`{"kind": "pulse", "role": "pad",
		"duty": 0.5, "arp": "1/8"}, {"kind": "pulse", "role": "lead",
		"duty": 0.5, "arp": 2}`, ""))
	if err != nil {
		t.Fatal(err)
	}
	if a := v.Channels[0].Arp; a.beats != (frac{1, 8}) {
		t.Errorf("arp \"1/8\" read as %s", a)
	}
	if a := v.Channels[1].Arp; a.ticks != 2 {
		t.Errorf("arp 2 read as %s", a)
	}
}

// the pair, not either document alone.

func bind(t *testing.T, sc, vc []byte) error {
	t.Helper()
	s, err := ParseScore(sc)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ParseVoicing(vc)
	if err != nil {
		t.Fatal(err)
	}
	_, err = bindVoicing(s, v)
	return err
}

func TestARoleNobodyPlaysMustBeDropped(t *testing.T) {
	sc := score(4, false, `{"role": "lead", "bars": ["c4:4"]},
		{"role": "pad", "bars": ["(c4 e4):4"]}`)
	unbound := bind(t, sc, voicing(lead, ""))
	wantErr(t, unbound,
		"pad: no channel plays it and it is not dropped")
	if err := bind(t, sc, voicing(lead, `"pad"`)); err != nil {
		t.Errorf("pad dropped, still refused: %v", err)
	}
}

func TestPitchesAndHitsGoWhereTheyCanSound(t *testing.T) {
	noise := `{"kind": "noise", "role": "perc",
		"hits": {"k": {"shift": 1, "divisor": 1, "decay": 0.1}}}`
	noteBound := bind(t,
		score(4, false, `{"role": "perc", "bars": ["c4:4"]}`),
		voicing(noise, ""))
	wantErr(t, noteBound,
		"a note on the noise channel")
	pitchedBound := bind(t,
		score(4, false, `{"role": "lead", "bars": ["k:4"]}`),
		voicing(lead, ""))
	wantErr(t, pitchedBound,
		`hit "k" on a pitched channel`)
	unknownBound := bind(t,
		score(4, false, `{"role": "perc", "bars": ["s:4"]}`),
		voicing(noise, ""))
	wantErr(t, unknownBound,
		`hit "s" is not one of k`)
}

func TestANoteOutsideTheRegisterIsRefused(t *testing.T) {
	outside := bind(t,
		score(4, false, `{"role": "lead", "bars": ["b1:4"]}`),
		voicing(lead, ""))
	wantErr(t, outside,
		"b1 is outside the pulse range c2 to c8")
}

func TestAnEchoOfADroppedRoleIsRefused(t *testing.T) {
	echoed := bind(t,
		score(4, false, `{"role": "lead", "bars": ["c4:4"]},
			{"role": "pad", "bars": ["c4:4"]}`),
		voicing(lead+`, {"kind": "pulse", "echo": "pad", "delay": "1",
			"gain": 0.5, "duty": 0.5}`, `"pad"`))
	wantErr(t, echoed,
		"echoes pad, which is dropped")
}
