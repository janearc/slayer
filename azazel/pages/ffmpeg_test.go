package pages

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"go.uber.org/mock/gomock"
)

// The generated frames of the real encode: small, of one colour, and of
// a height sixteen does not divide, so the padding shows.
const (
	frameWidth  = 160 // picked: small, and wide enough for VideoToolbox
	frameHeight = 90  // picked: not a multiple of 16
	framesMade  = 30  // defined: the ruling's thirty frames
	gapAt       = 20  // defined: the ruling's gap after twenty
	heightStep  = 16  // defined: the height QuickTime needs a multiple of
)

// The generated tone the film's music is: longer than the partial film,
// so the cut is the film's.
const (
	toneRate    = 48000 // picked: a common rate for music
	toneSeconds = 10    // picked: longer than twenty frames at six a second
	tonePitch   = 440   // picked: an A, any pitch would do
	toneLoud    = 8000  // picked: a quiet tone, well inside sixteen bits
	waveHeader  = 16    // defined: the size of a wave file's format chunk
	waveRest    = 36    // defined: the header's bytes after its size
	pcm         = 1     // defined: a wave file's plain samples
	mono        = 1     // defined: one channel
	sampleBytes = 2     // defined: sixteen bits a sample
	byteBits    = 8     // defined: bits in a byte
)

// sized is a generated picture of one grey, at the size given.
func sized(t *testing.T, width, height int) []byte {
	t.Helper()
	picture := image.NewGray(image.Rect(0, 0, width, height))
	for x := range width {
		for y := range height {
			picture.Set(x, y, color.Gray{Y: grey})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, picture); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// tone is a generated wave file of one sine tone, mono, sixteen bits.
func tone() []byte {
	samples := toneRate * toneSeconds
	var out bytes.Buffer
	write := func(values ...any) {
		for _, value := range values {
			binary.Write(&out, binary.LittleEndian, value)
		}
	}
	out.WriteString("RIFF")
	write(uint32(waveRest + samples*sampleBytes))
	out.WriteString("WAVEfmt ")
	write(uint32(waveHeader), uint16(pcm), uint16(mono), uint32(toneRate),
		uint32(toneRate*sampleBytes), uint16(sampleBytes),
		uint16(byteBits*sampleBytes))
	out.WriteString("data")
	write(uint32(samples * sampleBytes))
	for index := range samples {
		phase := 2 * math.Pi * tonePitch * float64(index) / toneRate
		write(int16(toneLoud * math.Sin(phase)))
	}
	return out.Bytes()
}

// digest is the sha256 of some bytes, in hexadecimal.
func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// gappedFilm writes thirty frames, numbered with a gap after the
// twentieth, and a receipt saying the rate and the music's digest, and
// returns its folder.
func gappedFilm(t *testing.T, rate, music string) (string, Folder) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "renders")
	folder := Folder{Shelf: animationsShelf, Name: filmName}
	frames := filepath.Join(folder.path(root), framesFolder)
	if err := os.MkdirAll(frames, folders); err != nil {
		t.Fatal(err)
	}
	picture := sized(t, frameWidth, frameHeight)
	for index := range framesMade {
		number := index
		if index >= gapAt {
			number++
		}
		name := filepath.Join(frames, fmt.Sprintf("frame-%06d.png", number))
		if err := os.WriteFile(name, picture, listing); err != nil {
			t.Fatal(err)
		}
	}
	receipt := filepath.Join(folder.path(root), "receipt.json")
	data := fmt.Sprintf(`{"fps": %s, "music_sha256": %q}`, rate, music)
	if err := os.WriteFile(receipt, []byte(data), listing); err != nil {
		t.Fatal(err)
	}
	return root, folder
}

// probedStream is what ffprobe says of one of a film's streams.
type probedStream struct {
	Kind      string `json:"codec_type"`
	Codec     string `json:"codec_name"`
	Tag       string `json:"codec_tag_string"`
	Primaries string `json:"color_primaries"`
	Transfer  string `json:"color_transfer"`
	Space     string `json:"color_space"`
	Height    int    `json:"height"`
	Frames    string `json:"nb_frames"`
	FrameRate string `json:"r_frame_rate"`
	Duration  string `json:"duration"`
}

// probe asks ffprobe about a film's streams, and gives them by kind.
func probe(t *testing.T, ffprobe, film string) map[string][]probedStream {
	t.Helper()
	told, err := exec.CommandContext(t.Context(), ffprobe, "-v", "error",
		"-show_entries", "stream=codec_type,codec_name,codec_tag_string,"+
			"color_primaries,color_transfer,color_space,height,nb_frames,"+
			"r_frame_rate,duration", "-of", "json", film).Output()
	var said struct {
		Streams []probedStream `json:"streams"`
	}
	if err == nil {
		err = json.Unmarshal(told, &said)
	}
	if err != nil {
		t.Fatal("ffprobe could not read the film:", err, string(told))
	}
	kinds := map[string][]probedStream{}
	for _, stream := range said.Streams {
		kinds[stream.Kind] = append(kinds[stream.Kind], stream)
	}
	return kinds
}

// length reads a stream's duration.
func length(t *testing.T, text string) float64 {
	t.Helper()
	seconds, err := strconv.ParseFloat(text, 64)
	if err != nil {
		t.Fatal(err)
	}
	return seconds
}

// TestFfmpegMakesTheFilmQuickTimePlays runs the real encode over thirty
// frames with a gap after twenty and a generated tone, and checks the
// film with ffprobe: picture and sound.
func TestWorkingMovieDoesNotSuppressPartial(t *testing.T) {
	music := tone()
	root, folder := gappedFilm(t, "6", digest(music))
	scoreMusic(t, folder.path(root), music, digest(music))
	working := filepath.Join(folder.path(root), ".film.mp4-123.mp4")
	if err := os.WriteFile(working, []byte("incomplete"), 0600); err != nil {
		t.Fatal(err)
	}
	p := publisherOf(Settings{Renders: root})
	look, err := p.view(folder, nil)
	if err != nil || look.finished != "" || !look.due(false) {
		t.Fatal("working file hid partial", look, err)
	}
	found, err := scanFolder(folder.path(root))
	if err != nil || len(found.films) != 0 {
		t.Fatal("working movie was published", found, err)
	}
}

func TestFfmpegMakesTheFilmQuickTimePlays(t *testing.T) {
	encoder := FoundFfmpeg()
	if encoder.Path == "" || encoder.Probe == "" {
		t.Skip("ffmpeg or ffprobe is not on the path; the real encode " +
			"was not tested")
	}
	music := tone()
	root, folder := gappedFilm(t, "6", digest(music))
	scoreMusic(t, folder.path(root), music, digest(music))
	made := publisherOf(Settings{Renders: root, Encoder: encoder})
	look := viewOf(t, root, folder, nil)
	if err := made.encode(t.Context(), folder.path(root), look); err != nil {
		t.Fatal(err)
	}
	kinds := probe(t, encoder.Probe, filepath.Join(folder.path(root),
		partialName))
	if len(kinds["video"]) != 1 || len(kinds["audio"]) != 1 {
		t.Fatalf("the film's streams: %+v", kinds)
	}
	picture, sound := kinds["video"][0], kinds["audio"][0]
	if picture.Codec != "hevc" || picture.Tag != "hvc1" ||
		picture.Primaries != "bt709" || picture.Transfer != "bt709" ||
		picture.Space != "bt709" || picture.Height%heightStep != 0 ||
		picture.Frames != "20" || picture.FrameRate != "6/1" ||
		sound.Codec != "aac" || math.Abs(length(t, sound.Duration)-
		length(t, picture.Duration)) > 1.0/filmRate {
		t.Fatalf("the film is not what QuickTime plays: %+v", kinds)
	}
	refuseSilentCopy(t, encoder, filepath.Join(folder.path(root),
		partialName))
}

// refuseSilentCopy copies a film without its sound, and checks that the
// check after muxing refuses the copy.
func refuseSilentCopy(t *testing.T, encoder Ffmpeg, film string) {
	t.Helper()
	silent := filepath.Join(t.TempDir(), "silent.mp4")
	told, err := exec.CommandContext(t.Context(), encoder.Path, "-v",
		"error", "-i", film, "-an", "-c", "copy", silent).CombinedOutput()
	if err != nil {
		t.Fatal(err, string(told))
	}
	if checkSound(t.Context(), encoder.Probe, Film{Rate: filmRate,
		Output: silent}) == nil {
		t.Fatal("a silent film passed the check after muxing")
	}
}

// TestMusicSourcesMustAgree: a screenplay naming other music than the
// score refuses the film; the same music from both is taken.
func TestMusicSourcesMustAgree(t *testing.T) {
	dir := t.TempDir()
	audio := scoreMusic(t, dir, []byte("music"), "")
	play := filepath.Join(dir, "screenplay.json")
	for _, named := range []string{"/elsewhere/music.wav", audio} {
		data := fmt.Sprintf(`{"music": %q}`, named)
		if err := os.WriteFile(play, []byte(data), listing); err != nil {
			t.Fatal(err)
		}
		found, _, reason := musicOf([]string{filepath.Join(dir, "score.json"),
			play})
		if (named == audio) != (reason == "" && found == audio) {
			t.Errorf("%s: %q %q", named, found, reason)
		}
	}
}

// publishWithoutAFilm publishes a gapped film's folder with an encoder
// that must not be asked, and returns its page.
func publishWithoutAFilm(t *testing.T, root string, folder Folder) string {
	t.Helper()
	for _, shelf := range []string{framesShelf, animationsShelf} {
		if err := os.MkdirAll(filepath.Join(root, shelf), folders); err != nil {
			t.Fatal(err)
		}
	}
	encoder := NewMockEncoder(gomock.NewController(t))
	settings := settingsFor(root, encoder, &clock{at: fixtureNoon})
	if _, err := Publish(t.Context(), settings, folder, 0); err != nil {
		t.Fatal(err)
	}
	if regularFile(filepath.Join(folder.path(root), partialName)) {
		t.Fatal("a partial film was made")
	}
	return read(t, filepath.Join(folder.path(root), pageName))
}

// TestDisagreeingRatesMakeNoFilm is the anti-test: with the receipt and
// a set file at odds on the rate, there is no film and the page says why.
func TestDisagreeingRatesMakeNoFilm(t *testing.T) {
	music := tone()
	root, folder := gappedFilm(t, "6", digest(music))
	scoreMusic(t, folder.path(root), music, "")
	set := filepath.Join(folder.path(root), "set.yaml")
	if err := os.WriteFile(set, []byte("fps: 24\n"), listing); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(publishWithoutAFilm(t, root, folder)),
		[]byte("disagree")) {
		t.Fatal("the page did not say the rates disagree")
	}
}

// TestNoMusicMakesNoFilm: a render that names no music gets no film, and
// the page says why; a silent film is never made.
func TestNoMusicMakesNoFilm(t *testing.T) {
	root, folder := gappedFilm(t, "6", "")
	page := publishWithoutAFilm(t, root, folder)
	if !bytes.Contains([]byte(page), []byte("names its music")) {
		t.Fatal("the page did not say there is no music:", page)
	}
}

// TestMusicNotMatchingItsDigestMakesNoFilm: music that is not what the
// receipt's digest says gets no film.
func TestMusicNotMatchingItsDigestMakesNoFilm(t *testing.T) {
	root, folder := gappedFilm(t, "6", digest([]byte("other music")))
	scoreMusic(t, folder.path(root), tone(), "")
	page := publishWithoutAFilm(t, root, folder)
	if !bytes.Contains([]byte(page), []byte("does not match")) {
		t.Fatal("the page did not say the music does not match:", page)
	}
}

// TestUnreadableMusicMakesNoFilm: music that cannot be read gets no film.
func TestUnreadableMusicMakesNoFilm(t *testing.T) {
	root, folder := gappedFilm(t, "6", "")
	audio := scoreMusic(t, folder.path(root), tone(), "")
	if err := os.Chmod(audio, 0); err != nil { // defined: no one may read it
		t.Fatal(err)
	}
	page := publishWithoutAFilm(t, root, folder)
	if !bytes.Contains([]byte(page), []byte("cannot be read")) {
		t.Fatal("the page did not say the music cannot be read:", page)
	}
}
