package queue

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/mock/gomock"
)

// TestInitializationRequiresDirectoryDurability refuses an unflushed new ledger.
func TestInitializationRequiresDirectoryDurability(t *testing.T) {
	for _, failParent := range []bool{false, true} {
		store := At(filepath.Join(t.TempDir(), "queue"))
		files := NewMockinitialFiles(gomock.NewController(t))
		files.EXPECT().MakeDirectory(store.root).DoAndReturn(func(path string) error { return os.Mkdir(path, 0700) })
		files.EXPECT().Create(filepath.Join(store.root, "queue.db")).DoAndReturn(func(path string) (io.Closer, error) {
			return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		})
		broken := errors.New("directory sync failed")
		rootError := broken
		if failParent {
			rootError = nil
		}
		first := files.EXPECT().SyncDirectory(store.root).Return(rootError)
		if failParent {
			files.EXPECT().SyncDirectory(filepath.Dir(store.root)).Return(broken).After(first)
		}
		if err := store.initialize(t.Context(), Config{Game: "/usr/bin/true"}, files); !errors.Is(err, broken) {
			t.Fatal("acknowledged without directory durability", err)
		}
	}
}

// TestInitializationRetainsCreateAndCloseFailures refuses partial initialization.
func TestInitializationRetainsCreateAndCloseFailures(t *testing.T) {
	for _, closeFails := range []bool{false, true} {
		s := At(filepath.Join(t.TempDir(), "queue"))
		files := NewMockinitialFiles(gomock.NewController(t))
		files.EXPECT().MakeDirectory(s.root).Return(nil)
		broken := errors.New("create refused")
		if closeFails {
			file, err := os.CreateTemp(t.TempDir(), "closed")
			if err != nil {
				t.Fatal(err)
			}
			file.Close()
			files.EXPECT().Create(filepath.Join(s.root, "queue.db")).Return(file, nil)
		} else {
			files.EXPECT().Create(filepath.Join(s.root, "queue.db")).Return(nil, broken)
		}
		if err := s.initialize(t.Context(), Config{Game: "/usr/bin/true"}, files); err == nil {
			t.Fatal("acknowledged partial initialization")
		}
	}
}

// TestFingerprintRetainsStatAndReadFailures never supplies a digest on failure.
func TestFingerprintRetainsStatAndReadFailures(t *testing.T) {
	broken := errors.New("source read failed")
	info, err := os.Stat("/usr/bin/true")
	if err != nil {
		t.Fatal(err)
	}
	for _, statFails := range []bool{false, true} {
		file := NewMockinputFile(gomock.NewController(t))
		if statFails {
			file.EXPECT().Stat().Return(nil, broken)
		} else {
			file.EXPECT().Stat().Return(info, nil)
			file.EXPECT().Read(gomock.Any()).Return(0, broken)
		}
		got, err := digest(file, "/input", true)
		if !errors.Is(err, broken) || got != (Fingerprint{}) {
			t.Fatal(got, err)
		}
	}
}
