package encoding

import (
	"unicode/utf8"
)

type Pointer interface {
	Value() byte
	Next() error
	Prev() error
}

type Encoding interface {
	Count(byte) int
	Decode([]byte) (rune, int)
	RuneOver(Pointer) (rune, int, int)
}

type UTF8Encoding struct{}

func (UTF8Encoding) Count(b byte) int {
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
	count := enc.Count(cursor.Value())
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

type DBCSEncoding struct{}

func (DBCSEncoding) Count(b byte) int {
	if IsDBCSLeadByte(b) {
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
