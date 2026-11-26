package door

import (
	"fmt"
	"os"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/session"
	yaml "gopkg.in/yaml.v3"
)

// The schemas a file given to a job-taking verb may name. Anything else,
// a set's own schema or none, is a set file.
const (
	jobSchema  = "https://github.com/janearc/slayer/schemas/job-v1.json"
	planSchema = "https://github.com/janearc/slayer/schemas/sequence-v1.json"
)

// readers is how each schema's file becomes a job.
var readers = map[string]func(path string) (job.Job, error){
	jobSchema:  job.Read,
	planSchema: session.LiftSequence,
}

// jobOfSource reads frozen bytes but interprets relative paths from their
// original document. Sequence plans do not yet have a buffered adapter.
func jobOfSource(path, original string) (job.Job, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return job.Job{}, err
	}
	var tree yaml.Node
	if err := yamlfile.Read(path, raw, &tree); err != nil {
		return job.Job{}, fmt.Errorf("%s is not a job or a set: %w", path, err)
	}
	var said heading
	if err := tree.Decode(&said); err != nil {
		return job.Job{}, fmt.Errorf("%s is not a job or a set: %w", path, err)
	}
	switch said.Schema {
	case jobSchema:
		return job.ReadSource(original, raw)
	case planSchema:
		return job.Job{}, fmt.Errorf("a relocated sequence plan is not supported")
	default:
		return job.StillFrom(original, raw)
	}
}

// heading is the one key of a file the door reads itself: its schema.
type heading struct {
	Schema string `yaml:"$schema"`
}

// jobOf is the job a file says: a job read, a sequence plan lifted, or a
// set file as a still of it. A file that is not there, or is not YAML,
// is refused by its name.
func jobOf(path string) (job.Job, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return job.Job{}, err
	}
	var said heading
	if err := yaml.Unmarshal(raw, &said); err != nil {
		return job.Job{}, fmt.Errorf("%s is not a job, a plan or a set: "+
			"%w", path, err)
	}
	if read, known := readers[said.Schema]; known {
		return read(path)
	}
	return job.StillOf(path)
}
