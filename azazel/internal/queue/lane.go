package queue

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// heldLane checks Game's inherited mark against its actual locked lane file.
func heldLane(kind string) error {
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	directory := filepath.Join(cache, "game", "lanes", kind)
	for _, mark := range filepath.SplitList(os.Getenv("GAME_LANES")) {
		named, path, ok := strings.Cut(mark, "=")
		if !ok || named != kind {
			continue
		}
		if filepath.Dir(path) != directory || !strings.HasPrefix(filepath.Base(path), "lane-") {
			return ErrUnsafe
		}
		file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}
		laneDescriptor := int(file.Fd())
		err = syscall.Flock(laneDescriptor, syscall.LOCK_EX|syscall.LOCK_NB)
		closeErr := file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return closeErr
		}
		return fmt.Errorf("the Game lane is not held: %w", ErrUnsafe)
	}
	return fmt.Errorf("execution requires an inherited Game %s lane", kind)
}
