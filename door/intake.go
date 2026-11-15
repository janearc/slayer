package door

import (
	"fmt"
	"os"

	"github.com/janearc/slayer/hillside"
	"github.com/janearc/slayer/job"
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
	planSchema: hillside.LiftSequence,
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
