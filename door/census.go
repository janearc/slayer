package door

// The census: assess FILE --census PATH stages the job by draw's own
// runner, as draw stages it, counts its first frame's world and writes
// what it counted to PATH as JSON, with the receipt that says what was
// counted: the job's bytes, its scene and camera, the time and the
// build. It draws nothing. A film's census is its first frame's; --at
// moves a still.
//
// The faces are those staged in the scene, not those visible or hit
// from the camera; the file says so in its counts field.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/janearc/slayer/hillside"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// censusFlag names the file assess writes the census to.
const censusFlag = "census"

// censusForm is the census file's form, which changes when its fields
// change meaning.
const censusForm = "slayer-census-v1"

// censusSays is the census file's one sentence on what it counts.
const censusSays = "Faces staged in the scene at the frame's time, " +
	"before any test of what the camera sees: not faces visible or hit " +
	"from the camera. A mesh group's faces count every instance; stored " +
	"faces count each distinct mesh once. Analytic shapes and volumes " +
	"are listed by kind and never given faces."

// unknownCommit is what the census says of a build that carries no
// commit, such as a test's.
const unknownCommit = "unknown"

// censusFrame is the frame a census counts: the first.
const censusFrame = 0

// counter is a source that counts its staged world: a hillside session.
type counter interface {
	Census(index int) (hillside.Census, error)
}

// errUncounted is the refusal of a census of frames already rendered.
var errUncounted = errors.New("this scene is frames already rendered, " +
	"pictures with no staged geometry to count")

// Census is the session's census of a frame.
func (made sessionSource) Census(index int) (hillside.Census, error) {
	return made.session.Census(index)
}

// censusFile is the census as written.
type censusFile struct {
	Form    string       `json:"form"`
	Counts  string       `json:"counts"`
	Says    string       `json:"says"`
	Receipt counted      `json:"receipt"`
	Camera  *eyeWritten  `json:"camera"`
	Unseen  string       `json:"camera_unsaid,omitempty"`
	World   thing.Census `json:"geometry"`
}

// counted is a census's receipt, what was counted: the job file and its
// sha256 and key, its scene and camera as the job resolved them, the set
// file's sha256 where the scene is a set, the frame and its scene time,
// and the build.
type counted struct {
	Job       string          `json:"job"`
	JobSHA256 string          `json:"job_sha256"`
	JobKey    string          `json:"job_key"`
	Scene     json.RawMessage `json:"scene"`
	SetSHA256 string          `json:"set_sha256,omitempty"`
	JobCamera json.RawMessage `json:"job_camera,omitempty"`
	Frame     int             `json:"frame"`
	Frames    int             `json:"frames"`
	At        float64         `json:"at_seconds"`
	Build     string          `json:"build"`
	Commit    string          `json:"commit"`
	Modified  string          `json:"commit_modified"`
}

// eyeWritten is the staged camera as written: where it stands, the way
// it looks, which way is up, its lens and its angle across in degrees.
type eyeWritten struct {
	Position [3]float64 `json:"position"`
	Look     [3]float64 `json:"look"`
	Up       [3]float64 `json:"up"`
	Lens     string     `json:"lens"`
	Wide     float64    `json:"wide_degrees,omitempty"`
}

// censused opens what draw would draw with, writes the census of the
// source to path, and lets the source go.
func censused(run runner, path string) error {
	tools, err := opening(run)
	if err != nil {
		return err
	}
	return errors.Join(writeCensus(run, tools.source, path),
		tools.source.Close())
}

// writeCensus counts the source's first frame and writes the census.
func writeCensus(run runner, opened source, path string) error {
	counting, counts := opened.(counter)
	if !counts {
		return errUncounted
	}
	census, err := counting.Census(censusFrame)
	if err != nil {
		return err
	}
	made, err := censusOf(run, census)
	if err != nil {
		return err
	}
	text, err := json.MarshalIndent(made, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), folderMode); err != nil {
		return err
	}
	return os.WriteFile(path, append(text, '\n'), fileMode)
}

// censusOf is the census file for what was counted, with its receipt.
func censusOf(run runner, census hillside.Census) (censusFile, error) {
	given, err := os.ReadFile(run.given)
	if err != nil {
		return censusFile{}, err
	}
	var canonical map[string]json.RawMessage
	if err := json.Unmarshal(run.wanted.Canonical(), &canonical); err != nil {
		return censusFile{}, err
	}
	commit, modified := commitOf()
	made := censusFile{Form: censusForm, Counts: thing.CensusCounts,
		Says: censusSays, Unseen: census.Unseen, World: census.World,
		Camera: eyeWrittenOf(census.Eye), Receipt: counted{
			Job: run.given, JobSHA256: digest(given),
			JobKey: run.wanted.Key(), Scene: canonical["scene"],
			JobCamera: canonical["camera"], Frame: censusFrame,
			Frames: run.wanted.Clock().Frames(),
			At:     census.At.Seconds(), Build: run.env.build.drawnBy(),
			Commit: commit, Modified: modified}}
	made.Receipt.SetSHA256, err = setDigest(run)
	return made, err
}

// setDigest is the sha256 of the job's set file, or nothing where the
// scene is not a set.
func setDigest(run runner) (string, error) {
	path, isSet := run.wanted.Scene().Set()
	if !isSet {
		return "", nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digest(raw), nil
}

// digest is the sha256 of bytes, in hex.
func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// commitOf is the commit the running build was made from and whether
// its tree was modified, as Go embeds them; unknown where it embeds
// none.
func commitOf() (commit, modified string) {
	commit, modified = unknownCommit, unknownCommit
	info, has := debug.ReadBuildInfo()
	if !has {
		return commit, modified
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			commit = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}
	return commit, modified
}

// eyeWrittenOf is the staged camera as written; nil where none is said.
func eyeWrittenOf(eye *hillside.Eye) *eyeWritten {
	if eye == nil {
		return nil
	}
	return &eyeWritten{Position: triple(eye.Position),
		Look: triple(eye.Look), Up: triple(eye.Up), Lens: eye.Lens,
		Wide: eye.Wide}
}

// triple is a vector as three numbers, x, y and z.
func triple(vector maths.Vec) [3]float64 {
	return [3]float64{vector.X, vector.Y, vector.Z}
}
