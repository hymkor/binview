package main

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/zetamatta/binview/internal/encoding"
	"github.com/zetamatta/binview/internal/large"
)

var (
	rxUnicodeCodePoint = regexp.MustCompile(`^\s*[uU]\+([0-9A-Fa-f]+)`)
	rxByte             = regexp.MustCompile(`^\s*0x([0-9A-Fa-f]+)`)
	rxDigit            = regexp.MustCompile(`^\s*([0-9]+)`)
	rxString           = regexp.MustCompile(`^\s*[uU]?"([^"]+)"`)
)

func parseInsertData(str string, enc encoding.Encoding) ([]byte, error) {
	bytes := make([]byte, 0)
	for len(str) > 0 {
		if m := rxUnicodeCodePoint.FindStringSubmatch(str); m != nil {
			str = str[len(m[0]):]
			theRune, err := strconv.ParseUint(m[1], 16, 32)
			if err != nil {
				return nil, err
			}
			if bin, err := enc.EncodeFromString(string(rune(theRune))); err == nil {
				bytes = append(bytes, bin...)
			}
		} else if m := rxByte.FindStringSubmatch(str); m != nil {
			str = str[len(m[0]):]
			theByte, err := strconv.ParseUint(m[1], 16, 16)
			if err != nil {
				return nil, err
			}
			bytes = append(bytes, byte(theByte))
		} else if m := rxDigit.FindStringSubmatch(str); m != nil {
			str = str[len(m[0]):]
			value, err := strconv.ParseUint(m[1], 10, 16)
			if err != nil {
				return nil, err
			}
			bytes = append(bytes, byte(value))
		} else if m := rxString.FindStringSubmatch(str); m != nil {
			str = str[len(m[0]):]
			if bin, err := enc.EncodeFromString(m[1]); err == nil {
				bytes = append(bytes, bin...)
			}
		} else {
			return bytes, fmt.Errorf("`%s` are ignored", str)
		}
	}
	return bytes, nil
}

func insertData(str string, enc encoding.Encoding, ptr *large.Pointer) error {
	data, err := parseInsertData(str, enc)
	if err != nil {
		return err
	}
	insertArea := ptr.MakeSpace(len(data))
	copy(insertArea, data)
	return nil
}

func (app *Application) InsertData(str string) error {
	return insertData(str, app.encoding, app.cursor)
}

func appendData(str string, enc encoding.Encoding, ptr *large.Pointer) error {
	data, err := parseInsertData(str, enc)
	if err != nil {
		return err
	}
	insertArea := ptr.MakeSpaceAfter(len(data))
	copy(insertArea, data)
	return nil
}

func (app *Application) AppendData(str string) error {
	return appendData(str, app.encoding, app.cursor)
}
