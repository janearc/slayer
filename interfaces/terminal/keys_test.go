package terminal

import (
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

// key and char are the keys a test expects.
func key(name KeyName) Key { return Key{name: name} }
func char(r rune) Key      { return Key{name: KeyChar, char: r} }

// decoded is the keys of bytes fed in the chunks given, then flushed if
// flush is set.
func decoded(chunks []string, flush bool) []Key {
	var decoder Decoder
	var keys []Key
	for _, chunk := range chunks {
		raw := []byte(chunk)
		keys = append(keys, decoder.Feed(raw)...)
	}
	if flush {
		keys = append(keys, decoder.Flush()...)
	}
	return keys
}

// each key is decoded from the bytes a terminal sends for it, from xterm's
// control sequences. Both forms of an arrow are the same key.
func TestEachKeyDecodes(t *testing.T) {
	for what, row := range map[string]struct {
		sent string
		want []Key
	}{
		"letter":            {"a", []Key{char('a')}},
		"symbol":            {"~", []Key{char('~')}},
		"wide character":    {"é", []Key{char('é')}},
		"three byte":        {"€", []Key{char('€')}},
		"return":            {"\r", []Key{key(KeyEnter)}},
		"newline":           {"\n", []Key{key(KeyEnter)}},
		"space":             {" ", []Key{key(KeySpace)}},
		"delete byte":       {"\x7f", []Key{key(KeyBackspace)}},
		"backspace byte":    {"\x08", []Key{key(KeyBackspace)}},
		"tab":               {"\t", []Key{key(KeyTab)}},
		"up":                {"\x1b[A", []Key{key(KeyUp)}},
		"down":              {"\x1b[B", []Key{key(KeyDown)}},
		"right":             {"\x1b[C", []Key{key(KeyRight)}},
		"left":              {"\x1b[D", []Key{key(KeyLeft)}},
		"up, other form":    {"\x1bOA", []Key{key(KeyUp)}},
		"down, other form":  {"\x1bOB", []Key{key(KeyDown)}},
		"right, other form": {"\x1bOC", []Key{key(KeyRight)}},
		"left, other form":  {"\x1bOD", []Key{key(KeyLeft)}},
		"page up":           {"\x1b[5~", []Key{key(KeyPageUp)}},
		"page down":         {"\x1b[6~", []Key{key(KeyPageDown)}},
		"home":              {"\x1b[H", []Key{key(KeyHome)}},
		"end":               {"\x1b[F", []Key{key(KeyEnd)}},
		"home, other form":  {"\x1bOH", []Key{key(KeyHome)}},
		"end, other form":   {"\x1bOF", []Key{key(KeyEnd)}},
		"home, number":      {"\x1b[1~", []Key{key(KeyHome)}},
		"end, number":       {"\x1b[4~", []Key{key(KeyEnd)}},
		"control-c":         {"\x03", []Key{key(KeyCtrlC)}},
		"two at once":       {"a\x1b[Ab", []Key{char('a'), key(KeyUp), char('b')}},
	} {
		if got := decoded([]string{row.sent}, false); !reflect.DeepEqual(got, row.want) {
			t.Errorf("%s: got %v, want %v", what, got, row.want)
		}
	}
}

// a lone escape is escape once the wait has lapsed, and not before.
// anti-test: before it lapses nothing is reported, and an escape with more
// after it is not an escape.
func TestALoneEscapeIsEscapeOnlyWhenTheWaitLapses(t *testing.T) {
	var decoder Decoder
	if got := decoder.Feed([]byte{byteEscape}); len(got) != 0 || !decoder.Waiting() {
		t.Fatalf("a lone escape reported at once: %v, waiting %v", got,
			decoder.Waiting())
	}
	if got := decoder.Flush(); !reflect.DeepEqual(got, []Key{key(KeyEscape)}) {
		t.Errorf("flushed: %v", got)
	}
	if decoder.Waiting() || decoder.Flush() != nil {
		t.Error("a flushed decoder still held bytes")
	}
	if got := decoded([]string{"\x1b[A"}, true); !reflect.DeepEqual(got, []Key{key(KeyUp)}) {
		t.Errorf("an arrow then a flush: %v", got)
	}
}

// bytes split across reads decode as if they came together.
// anti-test: a split that is not completed is one unknown, once flushed,
// not the bytes of it.
func TestSplitBytesDecodeAsOne(t *testing.T) {
	for what, chunks := range map[string][]string{
		"escape, then the rest":   {"\x1b", "[A"},
		"after the bracket":       {"\x1b[", "A"},
		"inside a number":         {"\x1b[5", "~"},
		"one byte at a time":      {"\x1b", "[", "5", "~"},
		"other form":              {"\x1bO", "A"},
		"other form, after the O": {"\x1b", "O", "A"},
	} {
		got := decoded(chunks, true)
		if len(got) != 1 || got[0].Name() == KeyUnknown || got[0].Name() == KeyEscape {
			t.Errorf("%s: got %v", what, got)
		}
	}
	if got := decoded([]string{"€"[:1], "€"[1:2], "€"[2:]}, true); !reflect.DeepEqual(got, []Key{char('€')}) {
		t.Errorf("a character split over reads: %v", got)
	}
	for what, chunks := range map[string][]string{
		"half a sequence":  {"\x1b["},
		"half a number":    {"\x1b[5"},
		"half a character": {"€"[:2]},
		"half the other":   {"\x1bO"},
	} {
		if got := decoded(chunks, true); !reflect.DeepEqual(got, []Key{key(KeyUnknown)}) {
			t.Errorf("%s: got %v, want one unknown", what, got)
		}
	}
}

// what has no name is one unknown key, never the characters in it.
// anti-test beside it: named things in the same bytes are not unknown.
func TestWhatHasNoNameIsOneUnknownKey(t *testing.T) {
	for what, sent := range map[string]string{
		"other number":     "\x1b[99~",
		"arrow with a mod": "\x1b[1;5A",
		"mouse report":     "\x1b[<0;10;20M",
		"function key":     "\x1bOP",
		"alt letter":       "\x1bx",
		"control letter":   "\x01",
		"null":             "\x00",
		"not utf-8":        "\xff",
		"control, c1":      "\u0085",
		"escape then rest": "\x1b\x7f",
		"long, no final":   "\x1b[" + "1111111111111111111111111111111111111111",
	} {
		got := decoded([]string{sent}, true)
		if len(got) != 1 || got[0].Name() != KeyUnknown {
			t.Errorf("%s: got %v, want one unknown", what, got)
		}
	}
	if got := decoded([]string{"\x1b[A"}, true); got[0].Name() == KeyUnknown {
		t.Error("an arrow was unknown")
	}
}

// byte 0x1d is its own key, in any company, and is never taken into
// another key: not after an escape, inside a sequence, after the O, or in
// half a character.
// anti-test: it is not escape, not unknown and not a character.
func TestTheHostKeyIsReportedAndNeverSwallowed(t *testing.T) {
	for what, row := range map[string]struct {
		sent string
		want []Key
	}{
		"alone":           {"\x1d", []Key{key(KeyHost)}},
		"between letters": {"a\x1db", []Key{char('a'), key(KeyHost), char('b')}},
		"after escape":    {"\x1b\x1d", []Key{key(KeyEscape), key(KeyHost)}},
		"inside sequence": {"\x1b[\x1d", []Key{key(KeyUnknown), key(KeyHost)}},
		"after parameter": {"\x1b[5\x1d", []Key{key(KeyUnknown), key(KeyHost)}},
		"after the O":     {"\x1bO\x1d", []Key{key(KeyUnknown), key(KeyHost)}},
		"in half a rune":  {"\xe2\x82\x1d", []Key{key(KeyUnknown), key(KeyUnknown), key(KeyHost)}},
		"twice":           {"\x1d\x1d", []Key{key(KeyHost), key(KeyHost)}},
	} {
		if got := decoded([]string{row.sent}, true); !reflect.DeepEqual(got, row.want) {
			t.Errorf("%s: got %v, want %v", what, got, row.want)
		}
	}
}

// whatever bytes come and however they are split, the keys are the same as
// when they come whole, and each 0x1d byte is one host key.
func TestPropertySplitsChangeNothingAndHostIsNeverLost(t *testing.T) {
	rapid.Check(t, func(test *rapid.T) {
		bytes := rapid.SliceOf(rapid.SampledFrom([]byte{
			0x1b, '[', 'O', 'A', '5', '~', ';', 0x1d, 'a', ' ', 0xe2,
			0x82, 0xac, 0x00, 0x03, '\r'})).Draw(test, "bytes")
		whole := decoded([]string{string(bytes)}, true)
		var chunks []string
		for rest := bytes; len(rest) > 0; {
			cut := rapid.IntRange(1, len(rest)).Draw(test, "cut")
			chunks = append(chunks, string(rest[:cut]))
			rest = rest[cut:]
		}
		parted := decoded(chunks, true)
		hosts := 0
		for _, b := range bytes {
			if b == byteHost {
				hosts++
			}
		}
		count := func(keys []Key) (n int) {
			for _, k := range keys {
				if k.Name() == KeyHost {
					n++
				}
			}
			return n
		}
		if count(whole) != hosts || count(parted) != hosts {
			test.Fatalf("%d host bytes, %d and %d host keys", hosts,
				count(whole), count(parted))
		}
		if len(chunks) == 1 && !reflect.DeepEqual(whole, parted) {
			test.Fatalf("one chunk decoded two ways: %v %v", whole, parted)
		}
	})
}

// the words of keys and names. anti-test: a name out of range says so.
func TestKeysAndNamesHaveWords(t *testing.T) {
	if char('x').String() != "x" || key(KeyPageUp).String() != "page-up" ||
		char('x').Rune() != 'x' || key(KeyUp).Rune() != 0 {
		t.Error("a key's word or rune was wrong")
	}
	if got := KeyName(99).String(); got != "key(99)" {
		t.Errorf("out of range: %q", got)
	}
	if got := KeyName(-1).String(); got != "key(-1)" {
		t.Errorf("below range: %q", got)
	}
}
