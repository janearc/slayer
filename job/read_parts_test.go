package job

import (
	"fmt"
	"strings"
)

// titled is a film with titles: a text, a counted text, a frame and a
// bar, in that order.
const titled = film + `titles:
  theme: dusk.css
  rows: 24
  elements:
    name:
      kind: text
      words: Moonlake
      ink: ink
      ground: dim
      across: 0.5
      down: 0.9
      align: centre
      from: {value: 2, unit: s}
      until: {value: 6, unit: s}
    height:
      kind: text
      count: {from: 0, to: 100, start: 1, over: 4}
      format: "%.0f m"
      ink: ink
      across: 0.1
      down: 0.1
    border:
      kind: frame
      ink: border
      left: 0
      top: 0
      right: 1
      bottom: 1
    progress:
      kind: bar
      value: 0.25
      fill: ok
      empty: dim
      left: 0.1
      top: 0.95
      right: 0.9
      bottom: 0.97
`

// element is the titled job's element at index.
func element(job Job, index int) Title {
	words, _ := job.Titles()
	return words.Elements()[index]
}

// the indices of the titled job's elements.
const (
	named = iota
	counted
	bordered
	barred
)

// titleReads are the titles' keys.
var titleReads = []reads{
	{"titles.theme", titled, func(job Job) (any, bool) {
		words, _ := job.Titles()
		return base(words.Theme())
	}, "dusk.css"},
	{"titles.rows", titled, func(job Job) (any, bool) {
		words, _ := job.Titles()
		return words.Rows()
	}, "24"},
	{"titles.elements in order", titled, func(job Job) (any, bool) {
		words, has := job.Titles()
		var names []string
		for _, each := range words.Elements() {
			names = append(names, each.Name()+" "+each.Kind())
		}
		return names, has
	}, "[name text height text border frame progress bar]"},
	{"title words", titled, func(job Job) (any, bool) {
		words, _, counting, _ := element(job, named).Words()
		return words, !counting
	}, "Moonlake"},
	{"title at", titled, func(job Job) (any, bool) {
		across, down, align := element(job, named).At()
		return spoken(across, down, align), true
	}, "0.5 0.9 centre"},
	{"title roles", titled, func(job Job) (any, bool) {
		ink, ground, _, _ := element(job, named).Roles()
		return ink + " " + ground, true
	}, "ink dim"},
	{"title from", titled, func(job Job) (any, bool) {
		return element(job, named).From()
	}, "2s"},
	{"title until", titled, func(job Job) (any, bool) {
		return element(job, named).Until()
	}, "6s"},
	{"title count", titled, func(job Job) (any, bool) {
		_, format, counting, count := element(job, counted).Words()
		from, to, start, over := count.Values()
		return spoken(format, from, to, start, over), counting
	}, "%.0f m 0 100 1 4"},
	{"title align unsaid", titled, func(job Job) (any, bool) {
		_, _, align := element(job, counted).At()
		return align, true
	}, "left"},
	{"title box", titled, func(job Job) (any, bool) {
		left, top, right, bottom := element(job, bordered).Box()
		return spoken(left, top, right, bottom), true
	}, "0 0 1 1"},
	{"title value", titled, func(job Job) (any, bool) {
		value, has := element(job, barred).Value()
		from, to, _, _ := value.Values()
		return spoken(from, to), has
	}, "0.25 0.25"},
	{"title fill and empty", titled, func(job Job) (any, bool) {
		_, _, fill, empty := element(job, barred).Roles()
		return fill + " " + empty, true
	}, "ok dim"},
}

// soundReads are the sound's keys.
var soundReads = []reads{
	{"sound.tune", film + "sound: {tune: score.yaml, voicing: '@plain'}\n",
		func(job Job) (any, bool) {
			track, _ := job.Sound()
			score, voicing, has := track.Tune()
			return spoken(base(score, has)) + " " + voicing, has
		}, "score.yaml true @plain"},
	{"sound.voicing path", film + "sound: {tune: '@moon', voicing: v.yaml}\n",
		func(job Job) (any, bool) {
			track, _ := job.Sound()
			score, voicing, has := track.Tune()
			return score + " " + spoken(base(voicing, has)), has
		}, "@moon v.yaml true"},
	{"sound.file", film + "sound: {file: take.wav}\n",
		func(job Job) (any, bool) {
			track, _ := job.Sound()
			return base(track.File())
		}, "take.wav"},
	{"sound.from", film + "sound: {file: take.wav, from: {value: 12.5, " +
		"unit: s}}\n", func(job Job) (any, bool) {
		track, _ := job.Sound()
		return track.From()
	}, "12.5s"},
	{"sound.score", film + "sound: {file: take.wav, score: beats.yaml}\n",
		func(job Job) (any, bool) {
			track, _ := job.Sound()
			return base(track.Beats())
		}, "beats.yaml"},
}

// outputReads are the outputs' and the limits' keys.
var outputReads = []reads{
	{"outputs.picture", still + "outputs: {picture: still.png}\n",
		func(job Job) (any, bool) { return base(job.Outputs().Picture()) },
		"still.png"},
	{"outputs.raw", still + "outputs: {raw: still.pfm}\n",
		func(job Job) (any, bool) { return base(job.Outputs().Raw()) },
		"still.pfm"},
	{"outputs.frames", film + "outputs: {frames: frames}\n",
		func(job Job) (any, bool) { return base(job.Outputs().Frames()) },
		"frames"},
	{"outputs.movie", film + "outputs: {movie: film.mp4}\n",
		func(job Job) (any, bool) { return base(job.Outputs().Movie()) },
		"film.mp4"},
	{"outputs.note", film + "outputs: {note: the lake at dusk}\n",
		func(job Job) (any, bool) { return job.Outputs().Note() },
		"the lake at dusk"},
	{"limits.memory", still + "limits: {memory: {value: 16, unit: GiB}}\n",
		func(job Job) (any, bool) { return job.Limits().Memory() }, "16"},
	{"limits.time", still + "limits: {time: {value: 7200, unit: s}}\n",
		func(job Job) (any, bool) { return job.Limits().Time() }, "2h0m0s"},
}

// spoken is values as fmt writes them, a space between each.
func spoken(values ...any) string {
	return strings.TrimSpace(fmt.Sprintln(values...))
}
