package job

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSourceUsesBufferAndResolvesAgainstOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "job.yaml")
	disk := []byte("$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {set: first.yaml}\nfilm: {width: 32, height: 24}\n")
	if err := os.WriteFile(path, disk, 0600); err != nil {
		t.Fatal(err)
	}
	saved, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := []byte(strings.Replace(string(disk), "first.yaml", "other.yaml", 1))
	edited, err := ReadSource(path, changed)
	if err != nil {
		t.Fatal(err)
	}
	set, has := edited.Scene().Set()
	if !has || set != filepath.Join(dir, "other.yaml") || edited.File() != path {
		t.Fatalf("buffer resolved to %q, file %q", set, edited.File())
	}
	if saved.PictureKey() == edited.PictureKey() {
		t.Fatal("changed scene kept picture key")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(disk) {
		t.Fatalf("disk changed: %q %v", after, err)
	}
}

func TestReadSourceRejectsInvalidBufferAndKeepsReadSemantics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "job.yaml")
	raw := []byte("$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {smoke: 1}\n")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	disk, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	buffer, err := ReadSource(path, raw)
	if err != nil || disk.Key() != buffer.Key() {
		t.Fatalf("buffer differs from read: %v", err)
	}
	if _, err := ReadSource(path, []byte("scene: [")); err == nil {
		t.Fatal("invalid buffer accepted")
	}
	if _, err := ReadSource("", raw); err == nil {
		t.Fatal("empty source path accepted")
	}
}
