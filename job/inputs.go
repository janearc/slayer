package job

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// The names the canonical form gives each file a job reads, under its
// inputs. A built-in tune or voicing (@name) is not a file: the build
// that draws the job is what holds it.
const (
	inputSet     = "set"
	inputFrames  = "frames"
	inputTheme   = "theme"
	inputTune    = "tune"
	inputVoicing = "voicing"
	inputSound   = "sound"
	inputBeats   = "beats"
	inputPlay    = "play"
)

// soundInputs are the inputs that change only the sound, which the
// picture's key leaves out.
var soundInputs = []string{inputTune, inputVoicing, inputSound,
	inputBeats}

// The digests of what is not a file that can be read.
const (
	absentDigest     = "absent"
	unreadableDigest = "unreadable"
)

// digested is the files a job reads, by name, and the sha256 of each,
// worked out once, when first asked for, since a folder of frames can
// be large.
type digested struct {
	once  sync.Once
	paths map[string]string
	sums  map[string]string
	fixed map[string]string
}

// inputs is every file the document reads, by its name in the canonical
// form, each absolute.
func (wire document) inputs() map[string]string {
	paths := map[string]string{inputSet: wire.Scene.Set,
		inputFrames: wire.Scene.Frames, inputPlay: wire.Scene.Play}
	if wire.Titles != nil {
		paths[inputTheme] = wire.Titles.Theme
	}
	if wire.Sound != nil {
		paths[inputTune] = wire.Sound.Tune
		paths[inputVoicing] = wire.Sound.Voicing
		paths[inputSound] = wire.Sound.File
		paths[inputBeats] = wire.Sound.Score
	}
	for name, path := range paths {
		if path == "" || path[:one] == builtIn {
			delete(paths, name)
		}
	}
	return paths
}

// of is the digest of every input, by name; a fresh map each call.
func (made *digested) of() map[string]string {
	if made == nil {
		return map[string]string{}
	}
	made.once.Do(func() {
		made.sums = map[string]string{}
		for name, path := range made.paths {
			if fixed, has := made.fixed[name]; has {
				made.sums[name] = fixed
			} else {
				made.sums[name] = digestOf(path)
			}
		}
	})
	copied := make(map[string]string, len(made.sums))
	for name, sum := range made.sums {
		copied[name] = sum
	}
	return copied
}

// digestOf is the sha256 of a file's bytes, or of a folder's files: each
// file directly inside it by name and the sha256 of its bytes, in name
// order. A path that is not there is absent; one that cannot be read is
// unreadable. Neither is refused here: the drawer refuses it by name.
func digestOf(path string) string {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return absentDigest
	}
	if err != nil {
		return unreadableDigest
	}
	if !info.IsDir() {
		return fileDigest(path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return unreadableDigest
	}
	folder := sha256.New()
	for _, entry := range entries {
		named := filepath.Join(path, entry.Name())
		// a frame may be a link to a file kept elsewhere: what it points
		// at is what is drawn from, so it is followed
		if info, err := os.Stat(named); err == nil && info.Mode().IsRegular() {
			_, _ = io.WriteString(folder, entry.Name()+"\x00"+
				fileDigest(named)+"\n")
		}
	}
	return hex.EncodeToString(folder.Sum(nil))
}

// fileDigest is the sha256 of a file's bytes, read as a stream.
func fileDigest(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return unreadableDigest
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return unreadableDigest
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// formed is the canonical form with the digest of every input under
// inputs, and without the sound and its inputs where sound is false.
// The marshalling cannot fail: the form was JSON a moment ago, which
// built checked.
func (job Job) formed(sound bool) []byte {
	var tree map[string]any
	_ = json.Unmarshal(job.canonical, &tree)
	if tree == nil {
		tree = map[string]any{}
	}
	sums := job.inputs.of()
	// a scene played from music moves with the tune: its sound is then
	// part of the picture, and stays in the picture's key
	if !sound && job.wire.Scene.Play == "" {
		delete(tree, "sound")
		for _, name := range soundInputs {
			delete(sums, name)
		}
	}
	if len(sums) > nothing {
		tree["inputs"] = sums
	}
	raw, _ := json.Marshal(tree)
	return raw
}

// Inputs is every file the job reads, by its name in the canonical
// form, each absolute: its set or its frames folder, its titles' theme,
// and its tune, voicing, sound file and beats. A built-in is not a file
// and is not among them. The map is a fresh copy each call.
func (job Job) Inputs() map[string]string {
	if job.inputs == nil {
		return map[string]string{}
	}
	copied := make(map[string]string, len(job.inputs.paths))
	for name, path := range job.inputs.paths {
		copied[name] = path
	}
	return copied
}

// developing is the parts of the canonical form that only develop the
// light, and the inputs they read: the light's key leaves them out.
var developing = []string{"print", "stock", "titles"}

// LightKey is the sha256 of the canonical form without the sound, the
// print, the stock and the titles and the files they read, in hex: two
// jobs with the same light key draw the same light, however each
// develops it.
func (job Job) LightKey() string {
	var tree map[string]any
	_ = json.Unmarshal(job.formed(false), &tree)
	for _, name := range developing {
		delete(tree, name)
	}
	if sums, has := tree["inputs"].(map[string]any); has {
		delete(sums, inputTheme)
		if len(sums) == nothing {
			delete(tree, "inputs")
		}
	}
	raw, _ := json.Marshal(tree)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// PictureKey is the sha256 of the canonical form without the sound and
// the files it reads, in hex: two jobs with the same picture key draw the
// same frames, whatever their sound.
func (job Job) PictureKey() string {
	sum := sha256.Sum256(job.formed(false))
	return hex.EncodeToString(sum[:])
}
