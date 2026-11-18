package pages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Encoder makes a film of numbered frames with its music. The
// coordinator's is ffmpeg; a test's is a generated mock.
type Encoder interface {
	Encode(ctx context.Context, film Film) error
}

// Film is one partial film to make: the first Count frames of the
// numbered frames in Frames, at Rate frames a second, with the film's
// music from Music cut to the same length, into Output.
type Film struct {
	Frames string
	Count  int
	Rate   float64
	Music  string
	Output string
}

// frameDigits is how many digits a frame's number has in its name.
const frameDigits = 6 // defined: slayer's frame-%06d.png

// framesFolder is the folder a film's numbered frames are in, when it is
// not the render folder itself.
const framesFolder = "frames" // defined: where slayer and searching put them

// largestRecord is the largest file read for a frame rate.
// picked: a receipt, screenplay or set is at most a few hundred
// kilobytes; a file this large is not one of them.
const largestRecord = 16 << 20

// growth is the share of new frames worth another partial film: a tenth.
// picked: each partial film encodes every frame again, so they come
// further apart as the film grows: about eighty for two thousand frames,
// about ten whole films' work in all.
const growth = 10

// stated is what a render's file may say of its frame rate.
type stated struct {
	Rate *float64 `json:"fps" yaml:"fps"`
}

// scored is what a render's file may say of its music: a score's audio
// file and digest, as heard's score.json has them, and the music's digest
// as heard's receipt.json has it.
type scored struct {
	Digest string `json:"music_sha256" yaml:"music_sha256"`
	Score  struct {
		Audio  string `json:"audio" yaml:"audio"`
		Digest string `json:"sha256" yaml:"sha256"`
	} `json:"score" yaml:"score"`
}

// screenplay is what a screenplay may say of the film's music, as
// heard's screenplay.json has it: the path of its audio file.
type screenplay struct {
	Music string `json:"music" yaml:"music"`
}

// partialFilm is what partial.json keeps of the partial film beside it.
type partialFilm struct {
	Frames  int     `json:"frames"`
	Rate    float64 `json:"rate"`
	Seconds float64 `json:"seconds"`
}

// filmView is what a film folder holds now: its numbered frames, the
// count from the first without a gap, the rate its files say, the
// partial film made so far, and a finished film once there is one.
type filmView struct {
	numbered bool
	frames   string
	count    int
	rate     float64
	music    string
	reason   string
	partial  partialFilm
	finished string
}

// frameNumber reads a frame's number from its name: six digits between
// "frame-" and ".png".
func frameNumber(name string) (int, bool) {
	digits, named := strings.CutPrefix(name, "frame-")
	digits, picture := strings.CutSuffix(digits, ".png")
	if !named || !picture || len(digits) != frameDigits ||
		strings.Trim(digits, "0123456789") != "" {
		return 0, false
	}
	number, err := strconv.Atoi(digits)
	return number, err == nil
}

// frameFolder is where a film's numbered frames are: its frames folder,
// when it has one, else the render folder.
func frameFolder(dir string) string {
	inner := filepath.Join(dir, framesFolder)
	if info, err := os.Stat(inner); err == nil && info.IsDir() {
		return inner
	}
	return dir
}

// contiguous counts the numbered frames from frame-000000.png up to the
// first gap, and says whether there are any numbered frames at all.
func contiguous(frames string) (int, bool, error) {
	entries, err := os.ReadDir(frames)
	have := map[int]bool{}
	for _, entry := range entries {
		number, numbered := frameNumber(entry.Name())
		if numbered && !entry.IsDir() {
			have[number] = true
		}
	}
	count := 0
	for have[count] {
		count++
	}
	return count, len(have) > 0, err
}

// decodeRecord reads a json or yaml file of the render's into what it
// may say, and says whether it could.
func decodeRecord(path string, into any) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, largestRecord+1))
	if err != nil || len(data) > largestRecord {
		return false
	}
	if strings.HasSuffix(path, ".json") {
		return json.Unmarshal(data, into) == nil
	}
	return yaml.Unmarshal(data, into) == nil
}

// statedRate reads the frame rate a json or yaml file states at its top.
func statedRate(path string) (float64, bool) {
	var said stated
	if !decodeRecord(path, &said) || said.Rate == nil || *said.Rate <= 0 ||
		math.IsInf(*said.Rate, 0) {
		return 0, false
	}
	return *said.Rate, true
}

// besideRecord reads a path a record names from the record's own folder
// when it is not absolute; nothing stays nothing.
func besideRecord(record, named string) string {
	if named == "" || filepath.IsAbs(named) {
		return named
	}
	return filepath.Join(filepath.Dir(record), named)
}

// agreed adds one file's word to the others', and says whether it agrees
// with them: a file that says nothing agrees with all.
func agreed(kept *string, from *string, said, path string) bool {
	if said == "" {
		return true
	}
	if *kept != "" && *kept != said {
		return false
	}
	*kept, *from = said, path
	return true
}

// musicOf is the film's music the render's files agree on: its audio file
// and, where one is given, its digest; or why there is none.
func musicOf(files []string) (string, string, string) {
	audio, digest, heard, hashed := "", "", "", ""
	for _, path := range files {
		var said scored
		var script screenplay
		if !decodeRecord(path, &said) {
			said = scored{}
		}
		if !decodeRecord(path, &script) {
			script = screenplay{}
		}
		for _, named := range []string{said.Score.Audio, script.Music} {
			if !agreed(&audio, &heard, besideRecord(path, named), path) {
				return "", "", "the render's files name two different music " +
					"files"
			}
		}
		if !agreed(&digest, &hashed, said.Digest, path) ||
			!agreed(&digest, &hashed, said.Score.Digest, path) {
			return "", "", "the render's files give two different music digests"
		}
	}
	if audio == "" {
		return "", "", "no file of the render's names its music (a score's " +
			"audio), and a partial film is never made without it"
	}
	return audio, digest, ""
}

// musicReason is why the music named cannot be used, or nothing: it must
// be a file that can be read, and match its digest where one is given.
func (p publisher) musicReason(audio, digest string) string {
	unreadable := "the film's music, " + audio + ", cannot be read"
	info, err := os.Stat(audio)
	if err != nil || !info.Mode().IsRegular() {
		return unreadable
	}
	key := fmt.Sprintf("%s %d %d", audio, info.Size(), info.ModTime().UnixNano())
	sum, known := p.digests.Load(key)
	if !known {
		read, err := digestOf(audio)
		if err != nil {
			return unreadable
		}
		sum, _ = p.digests.LoadOrStore(key, read)
	}
	if digest != "" && sum != digest {
		return "the film's music, " + audio + ", does not match the digest " +
			"the render's files give for it"
	}
	return ""
}

// digestOf is a file's sha256, in hexadecimal.
func digestOf(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// recordFiles is every file whose frame rate counts: the json and yaml
// files at the top of the render folder, and the job's own inputs.
// A folder that cannot be read has no records here; view says why.
func recordFiles(dir string, inputs []string) []string {
	entries, _ := os.ReadDir(dir)
	files := slices.Clone(inputs)
	for _, entry := range entries {
		name := entry.Name()
		record := strings.HasSuffix(name, ".json") ||
			strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
		if record && name != partialRecord && !entry.IsDir() {
			files = append(files, filepath.Join(dir, name))
		}
	}
	return files
}

// rateOf is the frame rate the render's files agree on, or why there is
// none: azazel never guesses one.
func rateOf(files []string) (float64, string) {
	rate, from := 0.0, ""
	for _, path := range files {
		said, found := statedRate(path)
		if !found {
			continue
		}
		if from != "" && said != rate {
			return 0, fmt.Sprintf("the render's files disagree on its frame "+
				"rate: %s says %v and %s says %v", filepath.Base(from), rate,
				filepath.Base(path), said)
		}
		rate, from = said, path
	}
	if from == "" {
		return 0, "no file of the render's says its frame rate (a top-level " +
			"fps in a json or yaml file beside the pictures, or in the job's " +
			"inputs), and azazel never guesses one"
	}
	return rate, ""
}

// finishedFilm is a film at the top of the render folder other than the
// partial one: the finished film, once it is there. A folder that cannot
// be read has none here; view says why.
func finishedFilm(dir string) string {
	entries, _ := os.ReadDir(dir)
	var films []string
	for _, entry := range entries {
		name := entry.Name()
		if isFilm(name) && name != partialName && !entry.IsDir() {
			films = append(films, name)
		}
	}
	if len(films) == 0 {
		return ""
	}
	return slices.MinFunc(films, collate)
}

// readPartial is what partial.json says of partial.mp4, or nothing when
// either is missing or the record cannot be read.
func readPartial(dir string) partialFilm {
	var record partialFilm
	data, err := os.ReadFile(filepath.Join(dir, partialRecord))
	if err != nil || !regularFile(filepath.Join(dir, partialName)) {
		return partialFilm{}
	}
	if json.Unmarshal(data, &record) != nil || record.Frames <= 0 {
		return partialFilm{}
	}
	return record
}

// view reads what a render folder's film holds now, making nothing. A
// folder of stills, or of no numbered frames, has no film.
func (p publisher) view(folder Folder, inputs []string) (filmView, error) {
	if folder.Shelf != animationsShelf {
		return filmView{}, nil
	}
	dir := folder.path(p.root)
	look := filmView{frames: frameFolder(dir), finished: finishedFilm(dir),
		partial: readPartial(dir)}
	count, numbered, err := contiguous(look.frames)
	look.count, look.numbered = count, numbered
	if !numbered || look.finished != "" {
		return look, err
	}
	records := recordFiles(dir, inputs)
	look.rate, look.reason = rateOf(records)
	if look.reason == "" {
		look.reason = p.musicFor(&look, records)
	}
	if look.reason == "" && count == 0 {
		look.reason = "the frames do not begin at frame-000000.png yet"
	}
	return look, err
}

// musicFor finds the film's music and says why it cannot be used, or
// nothing: a partial film is never made without it.
func (p publisher) musicFor(look *filmView, records []string) string {
	audio, digest, reason := musicOf(records)
	if reason == "" {
		reason = p.musicReason(audio, digest)
	}
	look.music = audio
	return reason
}

// due says whether the partial film should be made again now: at a job's
// end, when there is any new frame; while it runs, when a second of film
// or a tenth more frames has come, whichever is more.
func (look filmView) due(ending bool) bool {
	if !look.numbered || look.finished != "" || look.reason != "" {
		return false
	}
	last := look.partial.Frames
	if look.partial.Rate != look.rate {
		last = 0
	}
	if ending {
		return look.count > last
	}
	step := max(int(math.Ceil(look.rate)), last/growth)
	return look.count >= last+step
}

// encode makes the partial film beside partial.mp4 and renames it over,
// then records what it holds in partial.json.
func (p publisher) encode(ctx context.Context, dir string,
	look filmView) error {
	output := filepath.Join(dir, partialName)
	temporary, err := placeBeside(output)
	if err != nil {
		return err
	}
	err = p.encoder.Encode(ctx, Film{Frames: look.frames, Count: look.count,
		Rate: look.rate, Music: look.music, Output: temporary})
	if err == nil {
		err = os.Rename(temporary, output)
	}
	if err != nil {
		return errors.Join(err, os.Remove(temporary))
	}
	record := partialFilm{Frames: look.count, Rate: look.rate,
		Seconds: float64(look.count) / look.rate}
	data, err := json.Marshal(record)
	return errors.Join(err, writeBeside(filepath.Join(dir, partialRecord),
		data, written))
}

// lines is what the folder page says of the film.
func (look filmView) lines() []string {
	switch {
	case !look.numbered:
		return nil
	case look.finished != "":
		film := escape(look.finished)
		return []string{`The finished film: <a href="` + film + `">` + film +
			`</a>.`}
	case look.partial.Frames > 0:
		return []string{fmt.Sprintf(`The film so far, <a href="%s">%s</a>: `+
			`the first %d frames without a gap, %s seconds at %v frames a `+
			`second.`, partialName, partialName, look.partial.Frames,
			strconv.FormatFloat(look.partial.Seconds, 'f', 1, 64),
			look.partial.Rate)}
	case look.reason != "":
		return []string{"No film so far: " + escape(look.reason) + "."}
	}
	return []string{"The film so far is made once a second of frames is " +
		"in, from frame-000000.png without a gap."}
}
