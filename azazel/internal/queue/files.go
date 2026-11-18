package queue

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// initialFiles contains the filesystem operations needed before the first commit.
type initialFiles interface {
	MakeDirectory(string) error
	Create(string) (io.Closer, error)
	SyncDirectory(string) error
}

// inputFile exposes the operations required to fingerprint submitted bytes.
type inputFile interface {
	io.Reader
	Stat() (os.FileInfo, error)
}

type nativeFiles struct{}

// SyncDirectory is the directory durability boundary after initial creation.
func (nativeFiles) SyncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}

// MakeDirectory creates a private queue without adopting an existing directory.
func (nativeFiles) MakeDirectory(path string) error { return os.Mkdir(path, 0700) }

// Create exclusively creates the private initial database file.
func (nativeFiles) Create(path string) (io.Closer, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
}

// digest verifies file type before streaming the submitted bytes.
func digest(file inputFile, path string, executable bool) (Fingerprint, error) {
	var out Fingerprint
	info, err := file.Stat()
	if err != nil {
		return out, err
	}
	if !info.Mode().IsRegular() || executable && info.Mode()&0111 == 0 {
		return out, fmt.Errorf("not a regular executable or input: %s", path)
	}
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return out, err
	}
	return Fingerprint{Path: path, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}
