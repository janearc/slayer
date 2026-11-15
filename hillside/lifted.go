package hillside

import "github.com/janearc/slayer/job"

// LiftSequence is the job a sequence-v1 plan's file makes: read and
// checked as the sequence verb reads it, then lifted with job.Lift, so a
// plan drawn through a job draws the frames the sequence verb draws. A
// plan the sequence verb refuses is refused here in the same words.
func LiftSequence(path string) (job.Job, error) {
	plan, err := readSequence(path)
	if err != nil {
		return job.Job{}, err
	}
	return plan.lifted, nil
}
