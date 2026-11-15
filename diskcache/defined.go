// Package diskcache is the one place Slayer keeps what it worked out
// before: one folder per kind under one home, one file per entry, keyed
// by the content of what it was made from and the build. A hit is byte
// for byte a fresh computation; README.md says the whole design.
package diskcache

import "time"

// The variables the home is found by, in the order it looks.
const (
	homeVariable    = "SLAYER_CACHE"   // defined: this package's override
	rendersVariable = "SLAYER_RENDERS" // defined: the door's renders home
)

// The folders the home is, under the renders home or the user's cache.
const (
	rendersFolder = "cache"  // defined: the door's working frames' folder
	userFolder    = "slayer" // defined: the program's name
)

// The first line of every entry, and the words of its header's lines.
const (
	entryFirst    = "slayer-diskcache v1\n" // defined: version 1
	recordWord    = "record"
	writtenWord   = "written"
	payloadWord   = "payload"
	partSuffix    = ".part" // defined: as the door marks one
	fanDigits     = 2       // defined: the folder is a name's first two
	nothing       = 0       // defined: no entries, no bytes
	folderMode    = 0o755   // defined: a folder anyone may read
	fileMode      = 0o644   // defined: a file anyone may read
	writtenLayout = time.RFC3339Nano
)

// kindLetters is what a kind's name may be made of: lower-case letters
// and a hyphen between words, so it is a folder name everywhere.
// Defined.
const kindLetters = "abcdefghijklmnopqrstuvwxyz-"

// digestLength is a sha256's length in hex. Defined.
const digestLength = 64

// uniqueBytes is how many random bytes a .part name holds, so no two
// writers pick one name. Defined: as many as a version 4 uuid.
const uniqueBytes = 16
