package door

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/janearc/slayer/job"
)

// keys is what a film's ledger records it was drawn from: its frames'
// key, from the job's picture, the files it reads and the build; and its
// movie's, the same with the sound, so a changed tune remakes the movie
// from the frames already drawn.
type keys struct{ frames, movie string }

// keysOf is the keys of a job drawn by a build.
func keysOf(wanted job.Job, by string) keys {
	return keys{frames: joined(wanted.PictureKey(), by),
		movie: joined(wanted.Key(), by)}
}

// joined is the sha256 of a job's key and the build, in hex.
func joined(key, by string) string {
	sum := sha256.Sum256([]byte(key + "\n" + by))
	return hex.EncodeToString(sum[:])
}
