// Package diskcache is the one place Slayer keeps what it worked out
// before: one folder per kind under one home, one file per entry, keyed
// by the content of what it was made from and the build. A hit is byte
// for byte a fresh computation; README.md says the whole design.
//
// The home is SLAYER_CACHE, an absolute folder, and nothing else: every
// cache slayer keeps lives under it, and where it is not set the
// asker is refused by name. There is no fallback to the renders home,
// the user's cache folder or the temporary folder.
//
// A store lists, counts and removes only what it wrote: a regular file
// at folder/FAN/NAME, NAME a key's name and FAN its first digits, and
// the .part files of such a name. Anything else in the folder is a
// stranger, and is never listed, counted or removed.
package diskcache

import "time"

// homeVariable is the one variable the home is found by.
const homeVariable = "SLAYER_CACHE" // defined: the cache home

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

// hexLetters is what a key's name is made of: sha256 in lower-case
// hex. Defined.
const hexLetters = "0123456789abcdef"

// uniqueBytes is how many random bytes a .part name holds, so no two
// writers pick one name. Defined: as many as a version 4 uuid.
const uniqueBytes = 16
