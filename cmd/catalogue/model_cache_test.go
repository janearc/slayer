package main

import (
	"errors"
	"testing"
)

func TestModelBookCacheKeepsFamiliesPathsAndFailuresSeparate(t *testing.T) {
	type book struct{ source string }
	var cache map[string]*book
	calls := 0
	failure := errors.New("refused source")
	read := func(_ string, _ string, raw []byte) (*book, error) {
		calls++
		if string(raw) == "bad" {
			return nil, failure
		}
		return &book{source: string(raw)}, nil
	}
	first, err := cachedModelBook(&cache, "animal", "a/../book.yaml", []byte("first"), read)
	if err != nil {
		t.Fatal(err)
	}
	again, err := cachedModelBook(&cache, "animal", "book.yaml", []byte("changed"), read)
	if err != nil || first != again || again.source != "first" || calls != 1 {
		t.Fatalf("book changed on reuse: %+v, calls %d, error %v", again, calls, err)
	}
	other, err := cachedModelBook(&cache, "plant", "book.yaml", []byte("plant"), read)
	if err != nil || other == first || calls != 2 {
		t.Fatalf("families share books: %+v, calls %d, error %v", other, calls, err)
	}
	if _, err = cachedModelBook(&cache, "animal", "other.yaml", []byte("bad"), read); !errors.Is(err, failure) {
		t.Fatalf("lost the reader error: %v", err)
	}
	fixed, err := cachedModelBook(&cache, "animal", "other.yaml", []byte("fixed"), read)
	if err != nil || fixed.source != "fixed" || calls != 4 {
		t.Fatalf("failed book was cached: %+v, calls %d, error %v", fixed, calls, err)
	}
}
