package main

import (
	"go/scanner"
	"go/token"
	"strconv"
)

// swaps is what each operator and word is changed to, one at a time.
var swaps = map[string]string{
	"==": "!=", "!=": "==", "<": "<=", "<=": "<", ">": ">=", ">=": ">",
	"&&": "||", "||": "&&", "+": "-", "-": "+", "*": "/", "/": "*",
	"true": "false", "false": "true",
}

// change is one thing altered in a file: where, what it was, what it
// becomes, and the line.
type change struct {
	offset   int
	from, to string
	line     int
}

// apply is the file's text with the change made.
func (one change) apply(source []byte) []byte {
	made := append([]byte{}, source[:one.offset]...)
	made = append(made, one.to...)
	return append(made, source[one.offset+len(one.from):]...)
}

// changes is every change that can be made to the source between two
// lines. strings and comments are not scanned for them.
func changes(source []byte, first, last int) []change {
	var found []change
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(source))
	var reading scanner.Scanner
	reading.Init(file, source, nil, 0)
	for {
		pos, kind, text := reading.Scan()
		if kind == token.EOF {
			return found
		}
		if kind.IsOperator() {
			text = kind.String()
		}
		line := file.Line(pos)
		if line < first || line > last {
			continue
		}
		if to, ok := becomes(kind, text); ok {
			offset := file.Offset(pos)
			found = append(found, change{offset, text, to, line})
		}
	}
}

// becomes is what a token is changed to, if it can be.
func becomes(kind token.Token, text string) (string, bool) {
	if kind == token.INT {
		number, err := strconv.ParseInt(text, 0, 64)
		return strconv.FormatInt(number+1, 10), err == nil
	}
	to, ok := swaps[text]
	return to, ok && (kind.IsOperator() || kind == token.IDENT)
}
