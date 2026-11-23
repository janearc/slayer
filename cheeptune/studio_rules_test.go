package cheeptune

import (
	"math"
	"strings"
	"testing"
)

// The good channels the refusals below are near misses of.
const (
	goodFM = `"fm": {"arrangement": "chain", "operators": [
		{"ratio": 1, "level": 1}, {"ratio": 2, "level": 0.1}]}`
	goodAnalog = `"analog": {"oscillators": [{"shape": "saw"}]}`
	goodPluck  = `"pluck": {"brightness": 0.5, "damping": 0.3,
		"position": 0.2, "decay": 3, "release": 1}`
	goodModal = `"modal": {"set": "bar", "hardness": 0.5, "decay": 2}`
	goodDrums = `"drums": {"k": {"tone": 50, "decay": 0.3}}`
)

// studioChannel is one studio channel of a kind playing a role.
func studioChannelOf(kind, role, fields string) string {
	return `{"kind": "` + kind + `", "role": "` + role + `", ` + fields +
		`}`
}

func TestEveryGoodStudioChannelIsRead(t *testing.T) {
	for kind, fields := range map[string]string{"fm": goodFM,
		"analog": goodAnalog, "pluck": goodPluck, "modal": goodModal} {
		channel := studioChannelOf(kind, "lead", fields)
		_, err := ParseVoicing(studioVoicing(channel, ""))
		if err != nil {
			t.Errorf("a plain %s channel: %v", kind, err)
		}
	}
	drumChannel := studioChannelOf("drum", "perc", goodDrums)
	_, err := ParseVoicing(studioVoicing(drumChannel, ""))
	if err != nil {
		t.Errorf("a plain drum channel: %v", err)
	}
}

// nearMiss is a channel that breaks one rule and the words it is
// refused with.
type nearMiss struct{ doc, fragment string }

// studioNearMisses is a near miss for every rule a studio channel keeps.
func studioNearMisses() []nearMiss {
	fm := func(inner string) string {
		return studioChannelOf("fm", "lead", `"fm": {`+inner+`}`)
	}
	analog := func(inner string) string {
		return studioChannelOf("analog", "lead", `"analog": {`+inner+`}`)
	}
	drums := func(inner string) string {
		return studioChannelOf("drum", "perc", `"drums": {`+inner+`}`)
	}
	ops := `"operators": [{"ratio": 1, "level": 1},
		{"ratio": 2, "level": 0.1}]`
	return []nearMiss{
		{`{"kind": "flute", "role": "lead"}`,
			"unknown kind \"flute\"; the studio's kinds are analog, " +
				"drum, fm, modal, pluck"},
		{studioChannelOf("fm", "lead", goodFM+`, "duty": 0.5`),
			"duty is only for the gameboy machine"},
		{studioChannelOf("fm", "lead", goodFM+`, "pan": "left"`),
			"pan is only for the gameboy machine; a studio channel " +
				"has place"},
		{studioChannelOf("fm", "lead", goodFM+`, "place": 2`),
			"place 2 is outside -1 to 1"},
		{studioChannelOf("fm", "lead", goodFM+`, "send": {"reverb": 2}`),
			"send reverb 2 is outside 0 to 1"},
		{studioChannelOf("fm", "lead", goodFM+`, "send": {"reverb": 0.5}`),
			"sends to a reverb and the voicing has none"},
		{studioChannelOf("fm", "lead", goodFM+`, "send": {"delay": 0.5}`),
			"sends to a delay and the voicing has none"},
		{studioChannelOf("fm", "lead", goodFM+`, "voices": 40`),
			"voices 40 is outside 1 to 16"},
		{studioChannelOf("fm", "lead", goodFM+`, "detune": 150`),
			"detune 150 is outside -100 to 100"},
		{studioChannelOf("fm", "lead", goodFM+`, `+goodPluck),
			"pluck is only for a channel of kind pluck"},
		{studioChannelOf("fm", "lead", `"volume": 1`),
			"a channel of kind fm needs fm"},
		{studioChannelOf("fm", "lead", goodFM+`, "envelope": {"decay": 1}`),
			"envelope is not for a fm channel; each operator has its own"},
		{studioChannelOf("pluck", "lead", goodPluck+
			`, "vibrato": {"rate": 5, "depth": 5}`),
			"vibrato is not for a pluck channel"},
		{studioChannelOf("analog", "lead", goodAnalog+
			`, "vibrato": {"rate": 50, "depth": 5}`),
			"vibrato rate 50 is outside 0.05 to 20"},
		{studioChannelOf("analog", "lead", goodAnalog+
			`, "envelope": {"sustain": 2}`),
			"envelope sustain 2 is outside 0 to 1"},
		{studioChannelOf("drum", "perc", goodDrums+`, "arp": 3`),
			"arp, detune and voices are for pitched channels"},
		{fm(`"arrangement": "tree", ` + ops),
			`fm arrangement "tree"; arrangements are chain, fan, pairs, sum`},
		{fm(`"arrangement": "pairs", ` + ops),
			"fm arrangement pairs takes [4] operators and this has 2"},
		{fm(`"arrangement": "chain", "operators": [{"ratio": 99,
			"level": 1}, {"ratio": 1, "level": 3}]`),
			"fm operator 1 ratio 99 is outside 0.125 to 32"},
		{fm(`"arrangement": "chain", "feedback": 2, ` + ops),
			"fm feedback 2 is outside 0 to 1"},
		{analog(`"oscillators": []`), "analog has 0 oscillators"},
		{analog(`"oscillators": [{"shape": "sine"}]`),
			`shape "sine"; shapes are saw, square, triangle`},
		{analog(`"oscillators": [{"shape": "saw", "width": 0.3}]`),
			"width is only for a square"},
		{analog(`"oscillators": [{"shape": "square", "width": 0.99,
			"level": 2, "octave": 9}]`),
			"analog oscillator 1 width 0.99 is outside 0.05 to 0.95"},
		{analog(`"oscillators": [{"shape": "saw"}], "drive": 2`),
			"analog drive 2 is outside 0 to 1"},
		{analog(`"oscillators": [{"shape": "saw"}],
			"filter": {"cutoff": 5}`),
			"filter cutoff 5 is outside 20 to 20000"},
		{analog(`"oscillators": [{"shape": "saw"}],
			"filter": {"cutoff": 500, "amount": 2}`),
			"filter amount needs a filter envelope to move by"},
		{studioChannelOf("pluck", "lead", `"pluck": {"brightness": 2,
			"decay": 3, "release": 1}`),
			"pluck brightness 2 is outside 0 to 1"},
		{studioChannelOf("pluck", "lead", `"pluck": {"position": 0.8,
			"decay": 3, "release": 1}`),
			"pluck position 0.8 is outside 0 to 0.5"},
		{studioChannelOf("pluck", "lead", `"pluck": {"decay": 0,
			"release": 1}`), "pluck decay 0 is outside 0.01 to 30"},
		{studioChannelOf("modal", "lead", `"modal": {"set": "anvil",
			"decay": 2}`),
			`modal set "anvil"; sets are bar, bowl, marimba, plate, tube`},
		{studioChannelOf("modal", "lead", `"modal": {"hardness": 0.5}`),
			"modal needs a set (bar, bowl, marimba, plate, tube) or " +
				"its own modes"},
		{studioChannelOf("modal", "lead", `"modal": {"set": "bar",
			"decay": 2, "modes": [{"ratio": 1, "decay": 1, "level": 1}]}`),
			"modal has a set and its own modes; give one"},
		{studioChannelOf("modal", "lead", `"modal": {"decay": 2,
			"modes": [{"ratio": 99, "decay": 1, "level": 1}]}`),
			"modal decay is for a named set"},
		{studioChannelOf("modal", "lead", `"modal": {
			"modes": [{"ratio": 99, "decay": 1, "level": 1}]}`),
			"modal mode 1 ratio 99 is outside 0.1 to 40"},
		{studioChannelOf("modal", "lead", `"modal": {"modes": [`+
			strings.Repeat(`{"ratio": 1, "decay": 1, "level": 1},`, 12)+
			`{"ratio": 1, "decay": 1, "level": 1}]}`),
			"modal has 13 modes and the most is 12"},
		{drums(``), "a drum channel with no drums plays nothing"},
		{drums(`"x": {"tone": 50, "decay": 0.3}`),
			"drum x: a studio drum is one of k, s, h, o, t, b"},
		{drums(`"k": {"click": 0.5}`),
			"drum k has neither a tone nor noise and is silent"},
		{drums(`"k": {"tone": 5, "decay": 0.3}`),
			"drum k tone 5 is outside 20 to 2000"},
		{drums(`"k": {"tone": 50, "decay": 0.3, "sweep": 12}`),
			"drum k fall 0 is outside 0.001 to 8"},
		{drums(`"k": {"noise": 0.5, "noise_decay": 0.1}`),
			"drum k colour 0 is outside 20 to 16000"},
		{drums(`"k": {"tone": 50, "decay": 0.3, "volume": 2}`),
			"drum k volume 2 is outside 0 to 1"},
	}
}

func TestEveryRuleAStudioChannelKeepsRefusesItsNearMiss(t *testing.T) {
	for _, c := range studioNearMisses() {
		_, err := ParseVoicing(studioVoicing(c.doc, ""))
		if err == nil || !strings.Contains(err.Error(), c.fragment) {
			t.Errorf("%s:\nwant %q\ngot  %v", c.doc, c.fragment, err)
		}
	}
}

func TestTheStudiosMixKeepsItsRanges(t *testing.T) {
	channel := studioChannelOf("fm", "lead", goodFM)
	for extra, fragment := range map[string]string{
		`, "reverb": {"size": 2, "decay": 2}`: "reverb size 2 is outside",
		`, "reverb": {"size": 1, "decay": 99}`: "reverb decay 99 is " +
			"outside 0.1 to 20",
		`, "delay": {"beats": "x", "feedback": 0.5}`: "delay beats:",
		`, "delay": {"beats": "1", "feedback": 1}`: "delay feedback 1 " +
			"is outside 0 to 0.95",
	} {
		_, err := ParseVoicing(studioVoicing(channel, extra))
		wantErr(t, err, fragment)
	}
	_, err := ParseVoicing([]byte(`{"name": "v", "machine": "studio",
		"master": 9, "channels": [` + channel + `]}`))
	wantErr(t, err, "master 9 is outside 0.01 to 4")
	many := strings.Repeat(channel+",", 16) + channel
	_, err = ParseVoicing(studioVoicing(many, ""))
	wantErr(t, err, "the studio has 16 channels and the voicing has 17")
}

func TestTheGameBoyRefusesTheStudiosFieldsByName(t *testing.T) {
	for field, fragment := range map[string]string{
		`"place": 0.5`:             "place is only for the studio machine",
		`"send": {"reverb": 1}`:    "send is only for the studio machine",
		`"voices": 4`:              "voices is only for the studio machine",
		goodFM:                     "fm is only for the studio machine",
		goodAnalog:                 "analog is only for the studio machine",
		goodPluck:                  "pluck is only for the studio machine",
		goodModal:                  "modal is only for the studio machine",
		goodDrums:                  "drums is only for the studio machine",
		`"duty": 0.5, "duty": 0.5`: "",
	} {
		_, err := ParseVoicing(voicing(`{"kind": "pulse", "role": "lead",
			"duty": 0.5, `+field+`}`, ""))
		if fragment == "" {
			if err != nil {
				t.Errorf("a plain pulse: %v", err)
			}
			continue
		}
		wantErr(t, err, fragment)
	}
	for _, field := range []string{`"master": 0.5`,
		`"reverb": {"size": 1, "decay": 1}`,
		`"delay": {"beats": "1"}`} {
		_, err := ParseVoicing([]byte(`{"name": "v", "machine": "gameboy",
			` + field + `, "channels": [` + lead + `]}`))
		wantErr(t, err, "is only for the studio machine; the gameboy "+
			"has no mix to set")
	}
}

func TestTheStudioRefusesWhatAChannelCannotSound(t *testing.T) {
	drum := studioChannelOf("drum", "perc", goodDrums)
	fm := studioChannelOf("fm", "lead", goodFM)
	bind := func(part, channels string) error {
		s, err := ParseScore(slow(part))
		if err != nil {
			t.Fatal(err)
		}
		v, err := ParseVoicing(studioVoicing(channels, ""))
		if err != nil {
			t.Fatal(err)
		}
		_, err = bindVoicing(s, v)
		return err
	}
	wantErr(t, bind(`{"role": "perc", "bars": ["s:4"]}`, drum),
		`hit "s" is not one of k`)
	wantErr(t, bind(`{"role": "lead", "bars": ["c9:4"]}`, fm),
		"c9 is outside the studio's range c0 to c8")
	// the near miss: c8 is in.
	if err := bind(`{"role": "lead", "bars": ["c8:4"]}`, fm); err != nil {
		t.Errorf("c8 on the studio: %v", err)
	}
	// a role may be played by two studio channels, and both sound.
	if err := bind(`{"role": "lead", "bars": ["c4:4"]}`,
		fm+","+studioChannelOf("pluck", "lead", goodPluck)); err != nil {
		t.Errorf("two channels on one role: %v", err)
	}
	// a drum channel bound to notes and a pitched one bound to hits
	// are refused when the plan is made. only perc holds hits, so the
	// score puts each where it does not belong by way of the voicing.
	s, _ := ParseScore(slow(`{"role": "perc", "bars": ["k:4"]}`))
	fmChannel := studioChannelOf("fm", "perc", goodFM)
	v, _ := ParseVoicing(studioVoicing(fmChannel, ""))
	_, err := bindVoicing(s, v)
	wantErr(t, err, `hit "k" on a pitched channel`)
	s, _ = ParseScore(slow(`{"role": "lead", "bars": ["c4:4"]}`))
	leadDrumChannel := studioChannelOf("drum", "lead", goodDrums)
	v, _ = ParseVoicing(studioVoicing(leadDrumChannel, ""))
	_, err = bindVoicing(s, v)
	wantErr(t, err, "a note on a drum channel; it plays hits")
}

func TestAFullChannelTakesTheOldestVoiceAlreadyLetGo(t *testing.T) {
	ring := func(strike) float64 { return 10 }
	// three voices: the first let go at one second, the second and
	// third still held when a fourth and fifth arrive.
	sts := []strike{
		{on: 0, gate: 1, stolen: math.Inf(1)},
		{on: 0.5, gate: 9, stolen: math.Inf(1)},
		{on: 1, gate: 9, stolen: math.Inf(1)},
		{on: 2, gate: 1, stolen: math.Inf(1)},
		{on: 3, gate: 1, stolen: math.Inf(1)},
	}
	steal(sts, 3, ring)
	// at two seconds the first is the only one let go: it is taken.
	if sts[0].stolen != 2 {
		t.Errorf("the released voice was taken at %v", sts[0].stolen)
	}
	// at three seconds none of the three is let go (the fourth is held
	// until three, which counts as let go): the fourth goes.
	if sts[3].stolen != 1 || !math.IsInf(sts[1].stolen, 1) {
		t.Errorf("at 3 s: %v %v %v", sts[1].stolen, sts[2].stolen,
			sts[3].stolen)
	}
	// with nothing let go the oldest goes.
	held := []strike{
		{on: 0, gate: 9, stolen: math.Inf(1)},
		{on: 1, gate: 9, stolen: math.Inf(1)},
		{on: 2, gate: 9, stolen: math.Inf(1)},
	}
	steal(held, 2, ring)
	if held[0].stolen != 2 || !math.IsInf(held[1].stolen, 1) {
		t.Errorf("the oldest was not taken: %v", held)
	}
	// the anti-test: with room for all, nothing is taken.
	free := []strike{{on: 0, gate: 9, stolen: math.Inf(1)},
		{on: 1, gate: 9, stolen: math.Inf(1)}}
	steal(free, 2, ring)
	if !math.IsInf(free[0].stolen, 1) {
		t.Errorf("a voice was taken with room to spare")
	}
}

func TestAVoicingSaysWhichOfItsChannelsPlayHits(t *testing.T) {
	fmChannels := fmPair("2")
	drumChannel := studioChannelOf("drum", "perc", goodDrums)
	v, err := ParseVoicing(studioVoicing(fmChannels+","+drumChannel, ""))
	if err != nil {
		t.Fatal(err)
	}
	if v.Percussive(0) || !v.Percussive(1) || v.Percussive(7) ||
		v.Percussive(-1) {
		t.Errorf("percussive: %v %v", v.Percussive(0), v.Percussive(1))
	}
	g, _ := ParseVoicing(voicing(lead, ""))
	if g.Percussive(0) {
		t.Errorf("a pulse is not percussive")
	}
	if got := strings.Join((gameboy{}).kinds(), " "); got !=
		"pulse wave noise" {
		t.Errorf("the gameboy's kinds are %s", got)
	}
	if (&Voicing{Machine: "sid"}).Percussive(0) {
		t.Errorf("an unknown machine has percussive channels")
	}
}

func TestEveryLibraryTuneFitsEveryStudioVoicing(t *testing.T) {
	_, voicings := Builtin()
	for _, name := range voicings {
		v, err := ReadVoicing(name)
		if err != nil {
			t.Fatal(err)
		}
		if v.Machine != "studio" {
			continue
		}
		for _, tune := range Library() {
			s, err := ReadScore("@" + tune)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := bindVoicing(s, v); err != nil {
				t.Errorf("%s through %s: %v", tune, name, err)
			}
		}
	}
}
