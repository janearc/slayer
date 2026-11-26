package regioninput

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func sourceFile(t *testing.T, dir, name, value string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSnapshotRecordsTheCopiedBytes(t *testing.T) {
	dir := t.TempDir()
	source := sourceFile(t, dir, "source", "light")
	target := filepath.Join(dir, "snapshot")
	got, err := Snapshot(target, []Input{{Source: source, Name: "nested/file"}}, 5)
	if err != nil || len(got) != 1 || got[0].Bytes != 5 ||
		got[0].Name != "nested/file" || got[0].SHA256 != sha256.Sum256([]byte("light")) {
		t.Fatalf("manifest %+v, %v", got, err)
	}
	bytes, err := os.ReadFile(filepath.Join(target, "nested/file"))
	if err != nil || string(bytes) != "light" {
		t.Fatalf("copy %q, %v", bytes, err)
	}
}

func TestSnapshotRefusesInvalidRequestsWithoutLeavingPartialData(t *testing.T) {
	dir := t.TempDir()
	source := sourceFile(t, dir, "source", "light")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(source, link); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]Input{
		"empty":              {},
		"escape":             {{Source: source, Name: "../outside"}},
		"absolute":           {{Source: source, Name: filepath.Join(dir, "absolute")}},
		"unclean":            {{Source: source, Name: "sub/../file"}},
		"duplicate":          {{Source: source, Name: "a"}, {Source: source, Name: "a"}},
		"collision":          {{Source: source, Name: "a"}, {Source: source, Name: "a/b"}},
		"directory":          {{Source: dir, Name: "a"}},
		"symlink":            {{Source: link, Name: "a"}},
		"missing after copy": {{Source: source, Name: "a"}, {Source: filepath.Join(dir, "absent"), Name: "b"}},
	}
	for name, inputs := range cases {
		t.Run(name, func(t *testing.T) {
			target := filepath.Join(dir, name)
			got, err := Snapshot(target, inputs, 100)
			if err == nil || got != nil {
				t.Fatalf("accepted %+v: %+v, %v", inputs, got, err)
			}
			if _, err := os.Lstat(target); !os.IsNotExist(err) {
				t.Fatalf("left partial target: %v", err)
			}
		})
	}
}

func TestSnapshotRefusesLimitAndPreservesExistingTarget(t *testing.T) {
	dir := t.TempDir()
	source := sourceFile(t, dir, "source", "light")
	inputs := []Input{{Source: source, Name: "a"}}
	target := filepath.Join(dir, "snapshot")
	for _, limit := range []int64{0, -1, 4} {
		if _, err := Snapshot(target, inputs, limit); err == nil {
			t.Fatalf("accepted limit %d", limit)
		}
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatalf("left target: %v", err)
		}
	}
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	sourceFile(t, target, "kept", "old")
	if _, err := Snapshot(target, inputs, 100); err == nil {
		t.Fatal("accepted existing target")
	}
	if bytes, err := os.ReadFile(filepath.Join(target, "kept")); err != nil || string(bytes) != "old" {
		t.Fatalf("existing target changed: %q, %v", bytes, err)
	}
}
