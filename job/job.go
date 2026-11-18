// Package job reads and checks a job: the document that says how to
// render a scene. README.md beside this file is its design and its
// contract. It draws nothing and imports no renderer.
package job

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"

	yaml "gopkg.in/yaml.v3"
)

// Job is a checked job: its paths absolute, its preset applied, its
// camera built. It is made by Read, Lift or With, and never changes.
type Job struct {
	name        string
	file        string
	wire        document
	lens        Camera
	canonical   []byte
	outputForms outputForms
	// inputs is the files the job reads and their digests, shared by
	// the copies of one job and worked out once.
	inputs   *digested
	imported *importSnapshot
}

// Read is the job in the file at path, checked. A refusal says what was
// wrong and its file, line and column, and wraps ErrRefused.
func Read(path string) (Job, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Job{}, err
	}
	raw, err := os.ReadFile(absolute)
	if err != nil {
		return Job{}, err
	}
	from := source{name: path, base: filepath.Dir(absolute),
		file: absolute}
	return admitted(from, raw)
}

// built is the job from its checked wire: the camera built, and the
// canonical form written. Only the camera and a form that cannot be
// written can still be refused here.
func built(from source, root *yaml.Node, wire document) (Job, error) {
	var lens Camera
	if wire.Camera != nil {
		var where []string
		var err error
		lens, where, err = wire.Camera.lens()
		if err != nil {
			return Job{}, refuse(from.name, root, where, err.Error())
		}
	}
	canonical, err := wire.canonical()
	if err != nil {
		return Job{}, refuse(from.name, root, nil,
			"the job cannot be written in its canonical form: "+err.Error())
	}
	inputs, imported := from.inputs, from.imported
	if inputs == nil {
		inputs = &digested{paths: wire.inputs()}
		imported, err = freezeImported(wire.Scene.Set, inputs)
		if err != nil {
			return Job{}, err
		}
	}
	var forms outputForms
	if from.outputForms != nil {
		forms = *from.outputForms
	}
	return Job{name: from.name, file: from.file, wire: wire, lens: lens,
		outputForms: forms,
		canonical:   canonical,
		inputs:      inputs, imported: imported}, nil
}

// Schema is the schema's bytes, for the editor.
func Schema() []byte { return slices.Clone(schemaRaw) }

// File is the job file the job was read from, absolute, or nothing for
// a job lifted from a sequence plan.
func (job Job) File() string { return job.file }

// Canonical is the job's canonical form: JSON with sorted keys, its
// defaults filled in, every value that changes the picture or the
// sound, and under inputs the sha256 of every file the job reads, and
// nothing else.
func (job Job) Canonical() []byte { return job.formed(true) }

// Key is the sha256 of the canonical form, in hex. Two jobs with the
// same key draw the same film from the same files.
func (job Job) Key() string {
	sum := sha256.Sum256(job.Canonical())
	return hex.EncodeToString(sum[:])
}

// Renderer is the renderer the job names, and whether it names one.
func (job Job) Renderer() (string, bool) {
	return job.wire.Renderer, job.wire.Renderer != ""
}

// Quality is exact or preview, and whether the job said; a job that says
// none is exact.
func (job Job) Quality() (string, bool) {
	if job.wire.Quality == "" {
		return defaultQuality, false
	}
	return job.wire.Quality, true
}

// Camera is the job's camera, and whether it says one.
func (job Job) Camera() (Camera, bool) {
	return job.lens, job.wire.Camera != nil
}
