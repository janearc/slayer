package movie

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// every request in the table of refusals is refused, and says why in
// the words the table gives.
func TestRequestsAreRefusedByName(t *testing.T) {
	for _, row := range casesOf(t, refusedFile) {
		_, err := NewSettings(requestIn(t, row))
		says := textIn(t, row, "says")
		if err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("%s: %v, want it to say %q", row["case"], err, says)
		}
	}
}

// the near misses: every request in the table of taken ones is taken,
// coded at the height and the level the table gives.
func TestRequestsAreTakenAtTheirLevel(t *testing.T) {
	for _, row := range casesOf(t, takenFile) {
		settings, err := NewSettings(requestIn(t, row))
		if err != nil {
			t.Errorf("%s: refused: %v", row["case"], err)
			continue
		}
		coded, level := wholeIn(t, row, "coded"), textIn(t, row, "level")
		if settings.Coded().Y != coded || settings.Level() != level {
			t.Errorf("%s: coded %v at %s, want height %d at %s",
				row["case"], settings.Coded(), settings.Level(), coded,
				level)
		}
	}
}

// padding adds rows at the foot only to reach the next multiple of 16,
// and says how many.
func TestPaddingSaysHowManyRows(t *testing.T) {
	request := smallRequest(t)
	request.Height = picked("short-height")
	request.Pad = true
	settings := mustSettings(t, request)
	want := picked("small-height") - picked("short-height")
	if settings.Padding() != want || settings.Picture().Y != request.Height {
		t.Errorf("padding %d, picture %v; want %d rows under %d",
			settings.Padding(), settings.Picture(), want, request.Height)
	}
}

// the anti-test: a height already a multiple of 16 is not padded, even
// when padding is allowed.
func TestPaddingAddsNothingWhenNotNeeded(t *testing.T) {
	request := smallRequest(t)
	request.Pad = true
	if padding := mustSettings(t, request).Padding(); padding != 0 {
		t.Errorf("padded %d rows onto a height of %d", padding,
			request.Height)
	}
}

// a checked request keeps what was asked, and knows its length.
func TestSettingsKeepWhatWasAsked(t *testing.T) {
	request := smallRequest(t)
	request.Sound = tone(t, readRate, 1)
	settings := mustSettings(t, request)
	if settings.Path() != request.Path || settings.Rate() != request.Rate ||
		settings.Frames() != request.Frames ||
		settings.Sound().Length() != request.Sound.Length() ||
		settings.Duration().Seconds() != 1 {
		t.Errorf("settings %+v from request %+v", settings, request)
	}
}

// a directory that does not exist is refused by name.
func TestAMissingDirectoryIsRefused(t *testing.T) {
	request := smallRequest(t)
	request.Path = filepath.Join(t.TempDir(), "not-made", movieName)
	_, err := NewSettings(request)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("a missing directory: %v", err)
	}
}

// a path that is a file and not a directory, as the directory, is too.
func TestAFileAsTheDirectoryIsRefused(t *testing.T) {
	request := smallRequest(t)
	if err := os.WriteFile(request.Path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	request.Path = filepath.Join(request.Path, movieName)
	_, err := NewSettings(request)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("a file as the directory: %v", err)
	}
}

// no path, and a path not ending .mp4, are refused by name.
func TestPathsThatAreNotMoviesAreRefused(t *testing.T) {
	refusals := map[string]string{"": "no path",
		filepath.Join(t.TempDir(), "movie.mov"): "does not end .mp4"}
	for path, says := range refusals {
		request := smallRequest(t)
		request.Path = path
		_, err := NewSettings(request)
		if err == nil || !strings.Contains(err.Error(), says) {
			t.Errorf("path %q: %v, want %q", path, err, says)
		}
	}
}

// a file already at the path is refused unless Replace is set; with it
// the request is taken, and a relative path is made absolute.
func TestAnExistingFileIsReplacedOnlyWhenAsked(t *testing.T) {
	request := smallRequest(t)
	if err := os.WriteFile(request.Path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewSettings(request)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("an existing file: %v", err)
	}
	request.Replace = true
	if _, err := NewSettings(request); err != nil {
		t.Errorf("an existing file, replaced: %v", err)
	}
	request.Path = movieName
	settings := mustSettings(t, request)
	if !filepath.IsAbs(settings.Path()) {
		t.Errorf("relative path kept as %s", settings.Path())
	}
}
