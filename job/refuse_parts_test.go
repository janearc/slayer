package job

import "strings"

// retitled is the titled job with before replaced by after, once.
func retitled(before, after string) string {
	return strings.Replace(titled, before, after, one)
}

// titleMisses are near misses of the titles' keys.
var titleMisses = []miss{
	{"titles.theme empty", retitled("theme: dusk.css", "theme: ''"),
		[]string{"job.yaml:6:", "/titles/theme"}},
	{"titles.rows bound", retitled("rows: 24", "rows: 1"),
		[]string{"job.yaml:7:", "/titles/rows"}},
	{"title kind", retitled("kind: frame", "kind: label"),
		[]string{"job.yaml:", "/titles/elements/border"}},
	{"title text without ink", retitled("      ink: ink\n      ground",
		"      ground"), []string{"job.yaml:", "/titles/elements/name"}},
	{"title misspelt", retitled("words: Moonlake", "wrods: Moonlake"),
		[]string{"job.yaml:", "wrods"}},
	{"title count without format", retitled("      format: \"%.0f m\"\n",
		""), []string{"job.yaml:", "format"}},
	{"title across bound", retitled("across: 0.5", "across: 1.5"),
		[]string{"job.yaml:14:", "/titles/elements/name/across"}},
	{"title from unit", retitled("from: {value: 2, unit: s}",
		"from: {value: 2, unit: ms}"),
		[]string{"job.yaml:17:", "/titles/elements/name/from/unit"}},
	{"title until not after from", retitled("until: {value: 6",
		"until: {value: 2"), []string{"job.yaml:18:", "title name",
		"until is after from"}},
}

// soundMisses are near misses of the sound's keys.
var soundMisses = []miss{
	{"sound.tune without voicing", film + "sound: {tune: s.yaml}\n",
		[]string{"job.yaml:5:", "voicing"}},
	{"sound.voicing without tune", film + "sound: {voicing: v.yaml}\n",
		[]string{"job.yaml:5:", "tune"}},
	{"sound tune and file", film + "sound: {tune: s.yaml, voicing: " +
		"v.yaml, file: a.wav}\n", []string{"job.yaml:5:", "tune or a file"}},
	{"sound.from without file", film + "sound: {tune: s.yaml, voicing: " +
		"v.yaml, from: {value: 1, unit: s}}\n",
		[]string{"job.yaml:5:", "file"}},
	{"sound neither", film + "sound: {}\n",
		[]string{"job.yaml:5:", "says a tune"}},
	{"sound.file empty", film + "sound: {file: ''}\n",
		[]string{"job.yaml:5:", "/sound/file"}},
	{"sound.score type", film + "sound: {file: a.wav, score: [s]}\n",
		[]string{"job.yaml:5:", "/sound/score"}},
	{"sound on a still", still + "sound: {file: a.wav}\n",
		[]string{"job.yaml:3:", "a still has none"}},
}

// outputMisses are near misses of the outputs', limits' and the frames
// scene's keys.
var outputMisses = []miss{
	{"outputs.picture empty", still + "outputs: {picture: ''}\n",
		[]string{"job.yaml:3:", "/outputs/picture"}},
	{"outputs misspelt", still + "outputs: {pictures: a.png}\n",
		[]string{"job.yaml:3:", "pictures"}},
	{"outputs.note type", still + "outputs: {note: 3}\n",
		[]string{"job.yaml:3:", "/outputs/note"}},
	{"outputs.note with a tab", still + "outputs: {note: \"a\\tb\"}\n",
		[]string{"job.yaml:3:", "/outputs/note"}},
	{"a still with movie", still + "outputs: {movie: a.mp4}\n",
		[]string{"job.yaml:3:", "outputs.movie is a film's"}},
	{"a still with frames", still + "outputs: {picture: a.png, " +
		"frames: f}\n", []string{"job.yaml:3:", "outputs.frames is a film's"}},
	{"a film with picture", film + "outputs: {picture: a.png}\n",
		[]string{"job.yaml:5:", "outputs.picture is a still's"}},
	{"a film with raw", film + "outputs: {movie: a.mp4, raw: a.pfm}\n",
		[]string{"job.yaml:5:", "outputs.raw is a still's"}},
	{"limits.memory bound", still + "limits: {memory: {value: 0, unit: " +
		"GiB}}\n", []string{"job.yaml:3:", "/limits/memory"}},
	{"limits.time unit", still + "limits: {time: {value: 2, unit: h}}\n",
		[]string{"job.yaml:3:", "/limits/time/unit"}},
	{"frames with camera", heading + "scene: {frames: d}\n" +
		"camera: {name: rim}\n", []string{"job.yaml:3:", "nothing is traced",
		"camera"}},
	{"frames with shutter", heading + "scene: {frames: d}\nfps: 24\n" +
		"duration: {value: 1, unit: s}\nshutter: {angle: {value: 180, " +
		"unit: deg}, glimpses: 2}\n", []string{"job.yaml:5:",
		"nothing is traced", "shutter"}},
}
