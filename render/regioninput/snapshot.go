// Package regioninput copies an explicit list of files into a snapshot.
package regioninput

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Input names a source path and its relative name in the snapshot.
type Input struct {
	Source string
	Name   string
}

// Entry records the bytes actually copied under one name.
type Entry struct {
	Name   string
	Bytes  int64
	SHA256 [32]byte
}

// Snapshot creates a new private directory and copies at most maxBytes in all.
func Snapshot(target string, inputs []Input, maxBytes int64) (entries []Entry, result error) {
	if target == "" || len(inputs) == 0 || maxBytes <= 0 {
		return nil, fmt.Errorf("regioninput: target, inputs and positive byte limit are required")
	}
	seen := make(map[string]bool, len(inputs))
	for _, input := range inputs {
		if input.Source == "" || !filepath.IsLocal(input.Name) || filepath.Clean(input.Name) != input.Name || input.Name == "." {
			return nil, fmt.Errorf("regioninput: invalid source or relative name %q", input.Name)
		}
		if seen[input.Name] {
			return nil, fmt.Errorf("regioninput: duplicate name %q", input.Name)
		}
		seen[input.Name] = true
	}
	for name := range seen {
		for parent := filepath.Dir(name); parent != "."; parent = filepath.Dir(parent) {
			if seen[parent] {
				return nil, fmt.Errorf("regioninput: file and directory collide at %q", parent)
			}
		}
	}
	if err := os.Mkdir(target, 0700); err != nil {
		return nil, err
	}
	defer func() {
		if result != nil {
			entries = nil
			if err := os.RemoveAll(target); err != nil {
				result = errors.Join(result, fmt.Errorf("regioninput: remove partial snapshot: %w", err))
			}
		}
	}()
	remaining := maxBytes
	for _, input := range inputs {
		entry, err := copyInput(target, input, remaining)
		if err != nil {
			return nil, err
		}
		remaining -= entry.Bytes
		entries = append(entries, entry)
	}
	return entries, nil
}

// copyInput copies and hashes one stable, bounded source file.
func copyInput(target string, input Input, remaining int64) (Entry, error) {
	before, err := os.Lstat(input.Source)
	if err != nil {
		return Entry{}, err
	}
	if !before.Mode().IsRegular() {
		return Entry{}, fmt.Errorf("regioninput: source %q is not a regular file", input.Source)
	}
	source, err := os.Open(input.Source)
	if err != nil {
		return Entry{}, err
	}
	defer source.Close()
	opened, err := source.Stat()
	if err != nil {
		return Entry{}, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Size() > remaining {
		return Entry{}, fmt.Errorf("regioninput: source %q changed or exceeds the byte limit", input.Source)
	}
	path := filepath.Join(target, input.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return Entry{}, err
	}
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return Entry{}, err
	}
	hash := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(out, hash), io.LimitReader(source, remaining))
	var extra [1]byte
	if copyErr == nil {
		if count, readErr := source.Read(extra[:]); count != 0 || readErr != io.EOF {
			copyErr = fmt.Errorf("regioninput: source %q exceeds the byte limit or could not be read", input.Source)
		}
	}
	closeErr := out.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return Entry{}, err
	}
	after, err := source.Stat()
	if err != nil {
		return Entry{}, err
	}
	if after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) || n != opened.Size() {
		return Entry{}, fmt.Errorf("regioninput: source %q changed while copied", input.Source)
	}
	var digest [32]byte
	copy(digest[:], hash.Sum(nil))
	return Entry{Name: strings.Clone(input.Name), Bytes: n, SHA256: digest}, nil
}
