package terminal

// Keys: bytes from a terminal's keyboard, decoded to named keys.
//
// Design. A Decoder is a small state machine over the bytes it has been
// given and not yet used. Feed adds bytes and returns every key they
// complete; what is left, a lone escape or half a sequence, waits for the
// next Feed. Nothing in it reads a clock: the caller says when the wait for
// more bytes has lapsed by calling Flush, which is what makes a lone escape
// differ from the start of a sequence. Live's reader holds that wait.
//
// Control-right-bracket, byte 0x1d, belongs to the program that may be
// hosting this one, which uses it to take the terminal back. The decoder
// reports it as KeyHost, never as part of another key and never dropped, so
// that a caller can see it and ignore it. A caller MUST NOT bind KeyHost.

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// KeyName says which key a Key is.
type KeyName int

// The keys a decoder names.
const (
	KeyUnknown   KeyName = iota // a sequence or byte with no name here
	KeyChar                     // a printable character; see Key.Rune
	KeyEnter                    // return, or a newline
	KeySpace                    // the space bar
	KeyBackspace                // backspace or delete
	KeyTab                      // tab
	KeyUp                       // the arrows, in both forms terminals send
	KeyDown
	KeyLeft
	KeyRight
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd
	KeyCtrlC  // byte 0x03: raw mode delivers it as a key, not a signal
	KeyEscape // escape, alone
	// KeyHost is byte 0x1d, control-right-bracket. It is the hosting
	// program's. A caller must ignore it and never bind it.
	KeyHost
)

// names are the keys' words, in the order of KeyName.
var names = [...]string{"unknown", "char", "enter", "space", "backspace",
	"tab", "up", "down", "left", "right", "page-up", "page-down", "home",
	"end", "ctrl-c", "escape", "host"}

// String is the key's word, such as page-up.
func (name KeyName) String() string {
	if name < 0 || int(name) >= len(names) {
		return fmt.Sprintf("key(%d)", int(name))
	}
	return names[name]
}

// Key is one key pressed. A character key carries its character.
type Key struct {
	name KeyName
	char rune
}

// Name says which key it is.
func (key Key) Name() KeyName { return key.name }

// Rune is the character of a KeyChar, and nought of any other key.
func (key Key) Rune() rune { return key.char }

// String is the key's word, and for a character, the character.
func (key Key) String() string {
	if key.name == KeyChar {
		return string(key.char)
	}
	return key.name.String()
}

// The bytes the decoder knows by their own, each defined by the terminal's
// control codes (ECMA-48, and xterm's control sequences for the rest).
const (
	byteCtrlC     = 0x03 // defined: end of text
	byteBackspace = 0x08 // defined: backspace
	byteTab       = 0x09 // defined: horizontal tab
	byteNewline   = 0x0a // defined: line feed
	byteReturn    = 0x0d // defined: carriage return
	byteEscape    = 0x1b // defined: escape
	byteHost      = 0x1d // defined: group separator, control-right-bracket
	byteSpace     = 0x20 // defined
	byteDelete    = 0x7f // defined: what most backspace keys send
	byteCSI       = '['  // defined: escape then this opens a sequence
	byteSS3       = 'O'  // defined: escape then this opens the other form
	paramLeast    = 0x20 // defined: lowest byte inside a sequence
	paramMost     = 0x3f // defined: highest parameter or intermediate byte
	finalLeast    = 0x40 // defined: lowest final byte of a sequence
	finalMost     = 0x7e // defined: highest final byte of a sequence
	escapeLength  = 1    // defined: escape alone
	openerLength  = 2    // defined: escape and its opener
	// maxSequence is the most bytes a sequence may run to before it is
	// called unknown and let go. It is picked by the author of this
	// slice, 2026-11-12: the longest real key sequence is under ten.
	maxSequence = 32
)

// plain are the single bytes with a name.
var plain = map[byte]Key{
	byteCtrlC: {name: KeyCtrlC}, byteBackspace: {name: KeyBackspace},
	byteDelete: {name: KeyBackspace}, byteTab: {name: KeyTab},
	byteNewline: {name: KeyEnter}, byteReturn: {name: KeyEnter},
	byteSpace: {name: KeySpace}, byteHost: {name: KeyHost},
}

// arrows are the sequences after escape that name a key: the letter forms
// both ways round, and the number forms that end in a tilde.
var arrows = map[string]KeyName{
	"[A": KeyUp, "[B": KeyDown, "[C": KeyRight, "[D": KeyLeft,
	"OA": KeyUp, "OB": KeyDown, "OC": KeyRight, "OD": KeyLeft,
	"[H": KeyHome, "[F": KeyEnd, "OH": KeyHome, "OF": KeyEnd,
	"[1~": KeyHome, "[4~": KeyEnd, "[7~": KeyHome, "[8~": KeyEnd,
	"[5~": KeyPageUp, "[6~": KeyPageDown,
}

// Decoder turns bytes into keys, across reads that split them. The zero
// value is ready.
type Decoder struct {
	held []byte
}

// Feed adds bytes and returns the keys they complete. Bytes that do not yet
// make a key stay in the decoder; Waiting says so.
func (decoder *Decoder) Feed(bytes []byte) []Key {
	decoder.held = append(decoder.held, bytes...)
	var keys []Key
	for len(decoder.held) > 0 {
		key, used := decode(decoder.held)
		if used == 0 {
			break
		}
		keys = append(keys, key)
		decoder.held = decoder.held[used:]
	}
	decoder.held = append([]byte(nil), decoder.held...)
	return keys
}

// Waiting is whether the decoder holds bytes that are not yet a key.
func (decoder *Decoder) Waiting() bool { return len(decoder.held) > 0 }

// Flush is called when the wait for more bytes has lapsed. A lone escape is
// then KeyEscape. Any other half-sequence is one KeyUnknown. Nothing is
// held afterwards.
func (decoder *Decoder) Flush() []Key {
	held := decoder.held
	decoder.held = nil
	switch {
	case len(held) == 0:
		return nil
	case len(held) == escapeLength && held[0] == byteEscape:
		return []Key{{name: KeyEscape}}
	}
	return []Key{{name: KeyUnknown}}
}

// decode is the first key of held and how many bytes it took, or no bytes
// when held is the start of one and needs more.
func decode(held []byte) (Key, int) {
	if held[0] == byteEscape {
		return decodeEscape(held)
	}
	if key, named := plain[held[0]]; named {
		return key, 1
	}
	if held[0] < byteSpace {
		return Key{name: KeyUnknown}, 1
	}
	if !utf8.FullRune(held) {
		return Key{}, 0
	}
	char, width := utf8.DecodeRune(held)
	if char == utf8.RuneError || !unicode.IsPrint(char) {
		return Key{name: KeyUnknown}, width
	}
	return Key{name: KeyChar, char: char}, width
}

// decodeEscape is the key that begins with an escape byte. An escape
// followed by a control byte is escape alone, and the control byte is left
// for the next key.
func decodeEscape(held []byte) (Key, int) {
	if len(held) < openerLength {
		return Key{}, 0
	}
	if held[1] < byteSpace {
		return Key{name: KeyEscape}, escapeLength
	}
	switch held[1] {
	case byteCSI:
		return decodeSequence(held)
	case byteSS3:
		return decodeShifted(held)
	}
	return Key{name: KeyUnknown}, openerLength
}

// decodeShifted is the key of an escape, a capital O and one more byte. A
// control byte there is not part of it and is left for the next key.
func decodeShifted(held []byte) (Key, int) {
	if len(held) < openerLength+1 {
		return Key{}, 0
	}
	if held[2] < paramLeast {
		return Key{name: KeyUnknown}, openerLength
	}
	return named(held[1:openerLength+1], openerLength+1)
}

// decodeSequence is the key of a bracket sequence: parameters, then a final
// byte. A byte that cannot be inside one ends it unknown and is left for
// the next key, so that control-right-bracket is never swallowed.
func decodeSequence(held []byte) (Key, int) {
	for i := openerLength; i < len(held); i++ {
		switch b := held[i]; {
		case b >= finalLeast && b <= finalMost:
			return named(held[1:i+1], i+1)
		case b < paramLeast || b > paramMost:
			return Key{name: KeyUnknown}, i
		}
	}
	if len(held) >= maxSequence {
		return Key{name: KeyUnknown}, len(held)
	}
	return Key{}, 0
}

// named is the key a sequence names, or unknown, and the bytes it used.
func named(sequence []byte, used int) (Key, int) {
	if name, known := arrows[string(sequence)]; known {
		return Key{name: name}, used
	}
	return Key{name: KeyUnknown}, used
}
