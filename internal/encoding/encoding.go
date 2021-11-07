package encoding

import (
	"unicode/utf8"
)

type Pointer interface {
	Value() byte
	Next() error
	Prev() error
	Address() int64
}

type Encoding interface {
	Count(value byte, at int64) int
	Decode([]byte) (rune, int)
	RuneOver(Pointer) (rune, int, int)
	ModeString() string
}

type UTF8Encoding struct{}

func (UTF8Encoding) Count(b byte, _ int64) int {
	if 0xF0 <= b && b <= 0xF4 {
		return 4
	} else if 0xE0 <= b && b <= 0xEF {
		return 3
	} else if 0xC2 <= b && b <= 0xDF {
		return 2
	} else {
		return 1
	}
}

func (UTF8Encoding) Decode(data []byte) (rune, int) {
	return utf8.DecodeRune(data)
}

func (enc UTF8Encoding) RuneOver(cursor Pointer) (rune, int, int) {
	currentPosInRune := 0
	for !utf8.RuneStart(cursor.Value()) && cursor.Prev() == nil {
		currentPosInRune++
	}
	bytes := make([]byte, 0, utf8.UTFMax)
	count := enc.Count(cursor.Value(), 0)
	for i := 0; i < count; i++ {
		bytes = append(bytes, cursor.Value())
		if cursor.Next() != nil {
			break
		}
	}
	theRune, theLen := utf8.DecodeRune(bytes)
	if currentPosInRune >= theLen {
		return utf8.RuneError, 0, 1
	}
	return theRune, currentPosInRune, theLen
}

func (UTF8Encoding) ModeString() string {
	return "UTF8"
}

type DBCSEncoding struct{}

func (DBCSEncoding) Count(value byte, _ int64) int {
	if IsDBCSLeadByte(value) {
		return 2
	} else {
		return 1
	}
}

func (DBCSEncoding) Decode(data []byte) (rune, int) {
	utf16s, err := ToWideChar(data...)
	if err != nil {
		return utf8.RuneError, 1
	} else {
		return rune(utf16s[0]), 2
	}
}

func (DBCSEncoding) RuneOver(cursor Pointer) (rune, int, int) {
	return utf8.RuneError, 1, 1
}

func (DBCSEncoding) ModeString() string {
	return "ANSI"
}

type UTF16LE struct{}

func (UTF16LE) Count(_ byte, address int64) int {
	if address%2 == 0 {
		return 2
	} else {
		return 1
	}
}

func (UTF16LE) Decode(data []byte) (rune, int) {
	if len(data) == 2 {
		return (rune(data[1]) << 8) | (rune(data[0]) & 255), 2
	} else {
		return utf8.RuneError, 1
	}
}

func (UTF16LE) RuneOver(cursor Pointer) (rune, int, int) {
	currentPosInRune := 0
	theRune := rune(cursor.Value())
	if cursor.Address()%2 != 0 { // the second byte
		if cursor.Prev() != nil {
			return utf8.RuneError, 0, 1
		}
		theRune = (rune(cursor.Value()) << 8) | (theRune & 255)
		currentPosInRune++
	} else { // the first byte
		if cursor.Next() != nil {
			return utf8.RuneError, 0, 1
		}
		theRune = (theRune << 8) | (rune(cursor.Value()) & 255)
	}
	return theRune, currentPosInRune, 2
}

func (UTF16LE) ModeString() string {
	return "16LE"
}
