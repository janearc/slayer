package main

import (
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// said runs cueshow on testdata's score and sheet at a moment and is
// what it says, key by value.
func said(t *testing.T, seconds string) map[string]string {
	t.Helper()
	args, stdout := os.Args, os.Stdout
	defer func() { os.Args, os.Stdout = args, stdout }()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Args = []string{"cueshow", "testdata/score.yaml", "testdata/cues.yaml",
		seconds, "24"}
	os.Stdout = write
	main()
	write.Close()
	out, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	lines := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		if key, value, found := strings.Cut(line, "="); found {
			lines[key] = value
		}
	}
	return lines
}

// number is a said value as a number, or the test stops.
func number(t *testing.T, lines map[string]string, key string) float64 {
	t.Helper()
	value, err := finiteNumber(lines[key])
	if err != nil {
		t.Fatalf("%s=%q: %v", key, lines[key], err)
	}
	return value
}

// cueshow folds the sheet up to a moment: what was cut is there, what
// fades is part way, what comes later is not yet, and the grid says the
// bar; the fires burst from their cue's beat, and the moon rises, then
// beats.
func TestCueshowFoldsTheSheetToAMoment(t *testing.T) {
	before := said(t, "0.5")
	if before["bar"] != "intro" || number(t, before, "rig") != 0 {
		t.Errorf("before the grid: bar %s, rig %s", before["bar"], before["rig"])
	}
	if master := number(t, said(t, "2"), "master"); !(math.Abs(master-0.5) <= 1e-3) {
		t.Errorf("half way through the master's fade it is %v", master)
	}
	onFirst := said(t, "2.2")
	if onFirst["bar"] != "1.1" || number(t, onFirst, "rig") != 0.5 ||
		onFirst["beams"] != "4" || onFirst["colour"] != "0.080,1.000,0.120" {
		t.Errorf("in bar one: %v", onFirst)
	}
	if rig := number(t, said(t, "4.5"), "rig"); !(math.Abs(rig-0.5-0.5*0.5) <= 1e-3) {
		t.Errorf("half way through the rig's fade to 1 it is %v", rig)
	}
	lit := said(t, "4.75")
	if number(t, lit, "fire") != 1 || lit["fire_since"] != "0.2500" {
		t.Errorf("a quarter second after the fires' cue: fire %s, since %s",
			lit["fire"], lit["fire_since"])
	}
	if since := said(t, "7")["fire_since"]; since != "0.5000,2.5000" {
		t.Errorf("two bursts on, the fires burst %s ago", since)
	}
	if alt := number(t, said(t, "3"), "moon_alt"); !(math.Abs(alt-2) <= 1e-3) {
		t.Errorf("risen, the moon stands at %v", alt)
	}
	struck := number(t, said(t, "6"), "moon")
	between := number(t, said(t, "6.25"), "moon")
	if !(math.Abs(struck-1) <= 1e-3) || !(between <= 0.5) {
		t.Errorf("pulsing, the moon is %v on the beat and %v between",
			struck, between)
	}
}

// offset, the right heads play the pulse a sixteenth behind the left:
// back and forth; in unison they strike together.
func TestOffsetBanksGoBackAndForth(t *testing.T) {
	apart := said(t, "8.0")
	if apart["banks"] != "offset" || apart["figure"] != "scan" {
		t.Fatalf("at 8 s: banks %s, figure %s", apart["banks"], apart["figure"])
	}
	left, right := number(t, apart, "gate_left"), number(t, apart, "gate_right")
	if !(math.Abs(left-1) <= 1e-3) || !(right <= 0.5) {
		t.Errorf("on a hit, offset: left %v, right %v", left, right)
	}
	late := said(t, "8.125")
	if !(number(t, late, "gate_right") >= 0.99) {
		t.Errorf("a sixteenth on, the right strikes %v", number(t, late, "gate_right"))
	}
	together := said(t, "4.0")
	if together["banks"] != "unison" ||
		number(t, together, "gate_left") != number(t, together, "gate_right") {
		t.Errorf("in unison: %v", together)
	}
}

// TestFiniteNumberRefusesNaNAndInfinity feeds finiteNumber text that
// strconv.ParseFloat reads as a number but that is not a finite one, and
// text that is a finite number.
func TestFiniteNumberRefusesNaNAndInfinity(t *testing.T) {
	for _, text := range []string{"NaN", "nan", "Inf", "+Inf", "-Inf", "infinity"} {
		value, err := finiteNumber(text)
		if err == nil {
			t.Errorf("%q read as %v, want an error", text, value)
			continue
		}
		if !strings.Contains(err.Error(), strconv.Quote(text)) {
			t.Errorf("%q: error %q does not name the text", text, err)
		}
	}
	for text, want := range map[string]float64{"1.5": 1.5, "-0.25": -0.25, "12": 12} {
		got, err := finiteNumber(text)
		if err != nil || got != want {
			t.Errorf("%q read as %v with error %v, want %v", text, got, err, want)
		}
	}
}

// TestFiniteNumberRefusesWhatIsNotFinite checks a number is read as it
// is written, and that a word, NaN and both infinities are refused, the
// last three by name.
func TestFiniteNumberRefusesWhatIsNotFinite(t *testing.T) {
	if got, err := finiteNumber("-2.5e1"); err != nil || got != -25 {
		t.Errorf("-2.5e1 read as %v (%v)", got, err)
	}
	if _, err := finiteNumber("soon"); err == nil {
		t.Error("a word was read as a number")
	}
	for _, text := range []string{"NaN", "Inf", "-Inf"} {
		_, err := finiteNumber(text)
		if err == nil || !strings.Contains(err.Error(), text) {
			t.Errorf("%s gave %v, not a refusal naming it", text, err)
		}
	}
}

// TestCueshowSaysItsUsageForTheWrongNumberOfWords checks that three and
// six words print the usage line and nothing of the show.
func TestCueshowSaysItsUsageForTheWrongNumberOfWords(t *testing.T) {
	for _, words := range [][]string{
		{"cueshow", "testdata/score.yaml", "testdata/cues.yaml"},
		{"cueshow", "testdata/score.yaml", "testdata/cues.yaml", "1", "24", "extra"},
	} {
		args, stdout := os.Args, os.Stdout
		read, write, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Args, os.Stdout = words, write
		main()
		os.Args, os.Stdout = args, stdout
		write.Close()
		out, err := io.ReadAll(read)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(out); got != "cueshow score.yaml cues.yaml seconds [fps]\n" {
			t.Errorf("%d words said %q", len(words), got)
		}
	}
}
