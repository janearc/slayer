package cheeptune

import (
	"encoding/json"
	"testing"
)

// What rendering costs, since a bench that re-renders while somebody paints
// is only usable if this stays small.
//
// Not a gate: a number that fails on a busy machine teaches people to ignore
// the gate. Run it when the machine is quiet, and compare like with like.
//
//	go test -run X -bench . -benchmem

func BenchmarkRenderFourBars(b *testing.B) {
	s, v := benchPair(b, 4)
	b.ResetTimer()
	for b.Loop() {
		if _, err := Render(s, v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderEightBars(b *testing.B) {
	s, v := benchPair(b, 8)
	b.ResetTimer()
	for b.Loop() {
		if _, err := Render(s, v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOpenCrypt(b *testing.B) {
	// the whole way in, documents and all, which is what a game pays at
	// startup.
	for b.Loop() {
		if _, err := Open("@crypt", "@crypt-dmg"); err != nil {
			b.Fatal(err)
		}
	}
}

// benchPair is a score of n bars and the voicing it was written for.
func benchPair(b *testing.B, bars int) (*Score, *Voicing) {
	b.Helper()
	v, err := ReadVoicing("@crypt-dmg")
	if err != nil {
		b.Fatal(err)
	}
	s, err := ReadScore("@crypt")
	if err != nil {
		b.Fatal(err)
	}
	for i := range s.Parts {
		if len(s.Parts[i].Bars) > bars {
			s.Parts[i].Bars = s.Parts[i].Bars[:bars]
		}
	}
	// back through the document, so the events match the bars kept
	doc, err := json.Marshal(s)
	if err != nil {
		b.Fatal(err)
	}
	trimmed, err := ParseScore(doc)
	if err != nil {
		b.Fatal(err)
	}
	return trimmed, v
}

// benchRepeats is how many times over glass-hour is played to make the
// studio's long piece: five times fourteen bars at 66 a minute is four
// minutes and a quarter. Picked to pass the four minutes the studio is
// asked to render in a few seconds.
const benchRepeats = 5

// BenchmarkStudioFourMinutes is the studio rendering a little over four
// minutes through each of its built-in voicings. Measured on 2026-11-12
// on the shared laptop, three cores allowed, other work running: between
// one and two and a half seconds each.
func BenchmarkStudioFourMinutes(b *testing.B) {
	s, err := ReadScore("@glass-hour")
	if err != nil {
		b.Fatal(err)
	}
	for i := range s.Parts {
		once := s.Parts[i].Bars
		for range benchRepeats - 1 {
			s.Parts[i].Bars = append(s.Parts[i].Bars, once...)
		}
	}
	doc, err := json.Marshal(s)
	if err != nil {
		b.Fatal(err)
	}
	long, err := ParseScore(doc)
	if err != nil {
		b.Fatal(err)
	}
	_, voicings := Builtin()
	for _, name := range voicings {
		v, err := ReadVoicing(name)
		if err != nil {
			b.Fatal(err)
		}
		if v.Machine != "studio" {
			continue
		}
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				if _, err := Render(long, v); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
