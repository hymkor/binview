package encoding

import (
	"unicode/utf16"
	"unicode/utf8"

	"github.com/nyaosorg/go-windows-mbcs"
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
	EncodeFromString(string) ([]byte, error)
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

func (UTF8Encoding) EncodeFromString(s string) ([]byte, error) {
	return []byte(s), nil
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

func (DBCSEncoding) EncodeFromString(s string) ([]byte, error) {
	return mbcs.UtoA(s, mbcs.ACP)
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

type _UTF16 struct {
	isLittleEndian bool
}

func (this _UTF16) utf16ToRune(first, second rune) rune {
	if this.isLittleEndian {
		return (second << 8) | (first & 255)
	} else {
		return (first << 8) | (second & 255)
	}
}

func UTF16LE() Encoding {
	return _UTF16{isLittleEndian: true}
}

func UTF16BE() Encoding {
	return _UTF16{isLittleEndian: false}
}

func (_UTF16) Count(_ byte, address int64) int {
	if address%2 == 0 {
		return 2
	} else {
		return 1
	}
}

func (this _UTF16) Decode(data []byte) (rune, int) {
	if len(data) == 2 {
		return this.utf16ToRune(rune(data[0]), rune(data[1])), 2
	} else {
		return utf8.RuneError, 1
	}
}

func (this _UTF16) EncodeFromString(s string) ([]byte, error) {
	bytes := make([]byte, 0, len(s)*2)
	for _, utf16data := range utf16.Encode([]rune(s)) {
		if this.isLittleEndian {
			bytes = append(bytes, byte(utf16data), byte(utf16data>>8))
		} else {
			bytes = append(bytes, byte(utf16data>>8), byte(utf16data))
		}
	}
	return bytes, nil
}

func (this _UTF16) RuneOver(cursor Pointer) (rune, int, int) {
	currentPosInRune := 0
	theRune := rune(cursor.Value())
	if cursor.Address()%2 != 0 { // the second byte
		if cursor.Prev() != nil {
			return utf8.RuneError, 0, 1
		}
		theRune = this.utf16ToRune(rune(cursor.Value()), theRune)
		currentPosInRune++
	} else { // the first byte
		if cursor.Next() != nil {
			return utf8.RuneError, 0, 1
		}
		theRune = this.utf16ToRune(theRune, rune(cursor.Value()))
	}
	return theRune, currentPosInRune, 2
}

func (this _UTF16) ModeString() string {
	if this.isLittleEndian {
		return "16LE"
	} else {
		return "16BE"
	}
}
