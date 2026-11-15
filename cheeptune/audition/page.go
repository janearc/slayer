package audition

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
)

// Page writes index.html for the tunes: the list listened to and voted
// on. Each row is the film's player; the tune's mood, feeling, intent
// and what to listen for as prose; its parts as a timeline with what each
// feels like, the player seeking to a part when its name is pressed; and
// two large buttons, keep and toss.
//
// The page asks /votes for the record. When there is no server (it was
// opened as a file) it keeps votes in the browser's local storage, says so,
// and offers one button that copies the votes as text. No build step and
// nothing from the network.

// row is one tune as the template reads it.
type row struct {
	Name, Mood, Feeling, Intent, Listen, Length, Tempo string
	Parts                                              []partRow
}

// partRow is one part as the template reads it.
type partRow struct {
	Name, Feels, At, Seconds, Share string
}

// group is a heading and the rows under it.
type group struct {
	Name string
	Rows []row
}

// Page is the page's bytes for the tunes, grouped by their sheets' groups
// in groupOrder. It reads a tune's name and sheet and nothing else, so a
// tune that was read and not rendered has its row like any other.
func Page(tunes []*Tune) ([]byte, error) {
	var buf bytes.Buffer
	data := struct {
		Count  int
		Groups []group
		Style  template.CSS
		Script template.JS
	}{len(tunes), groups(tunes), template.CSS(pageStyle),
		template.JS(pageScript)}
	if err := pageTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// groupOrder is the order the page lists groups in: the instruments
// first, then glass-hour as acid, then what was written by hand for the
// studio (zappy's theme, the rest of the glass-hour set), then the one
// tune heard through the studio's first sounds, then the moods in the
// order they were asked for on 2026-11-12; a group not named here follows them, in the order
// it first appears.
var groupOrder = []string{"instruments", "glass hour, acid", "zappy",
	"glass hour", "glass hour, new sounds",
	"sombre and slow", "tense and sparse",
	"bright and driving", "playful", "triumphant", "eerie",
	"sad, turning hopeful", "slow start, fast finish"}

// rank is where a group stands in the page's order.
func rank(name string) int {
	for i, g := range groupOrder {
		if g == name {
			return i
		}
	}
	return len(groupOrder)
}

// groups collects the tunes under their sheets' groups, in order.
func groups(tunes []*Tune) []group {
	sorted := append([]*Tune(nil), tunes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	var out []group
	index := map[string]int{}
	for _, t := range sorted {
		name := t.Sheet.Group
		at, ok := index[name]
		if !ok {
			at = len(out)
			index[name] = at
			out = append(out, group{Name: name})
		}
		out[at].Rows = append(out[at].Rows, rowOf(t))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return rank(out[i].Name) < rank(out[j].Name)
	})
	return out
}

// rowOf is the template's view of one tune.
func rowOf(t *Tune) row {
	r := row{Name: t.Name, Mood: t.Sheet.Mood, Feeling: t.Sheet.Feeling,
		Intent: t.Sheet.Intent, Listen: t.Sheet.ListenFor,
		Length: clock(t.Sheet.Seconds),
		Tempo:  fmt.Sprintf("%v a minute", t.Sheet.Tempo)}
	for _, span := range t.Sheet.Spans() {
		share := (span.End - span.Start).Seconds() / t.Sheet.Seconds * 100
		r.Parts = append(r.Parts, partRow{Name: span.Name,
			Feels: span.Feels, At: clock(span.Start.Seconds()),
			Seconds: fmt.Sprintf("%.2f", span.Start.Seconds()),
			Share:   fmt.Sprintf("%.2f", share)})
	}
	return r
}

// clock is seconds as minutes and seconds, 0:14.
func clock(seconds float64) string {
	whole := int(seconds + 0.5)
	return fmt.Sprintf("%d:%02d", whole/60, whole%60)
}

var pageTemplate = template.Must(template.New("page").Parse(pageHTML))
