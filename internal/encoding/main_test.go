package encoding

import (
	"errors"
	"runtime"
	"testing"
)

var _ Encoding = UTF8Encoding{}
var _ Encoding = DBCSEncoding{}
var _ Encoding = UTF16LE()
var _ Encoding = UTF16BE()

func TestIsDBCSLeadByte(t *testing.T) {
	if runtime.GOOS == "windows" && !IsDBCSLeadByte(0x83) { // Japanese katakana SO
		t.Fail()
		return
	}
	if IsDBCSLeadByte('a') {
		t.Fail()
		return
	}
}

func TestToWideChar(t *testing.T) {
	utf16, err := ToWideChar(0x83, 0x5C)
	if err != nil {
		if !errors.Is(err, ErrNotSupport) {
			t.Fatal(err.Error())
		}
		return
	}
	if len(utf16) != 1 {
		t.Fatalf("len(utf16)==%d", len(utf16))
		return
	}
	if utf16[0] != 0x30BD {
		t.Fatalf("utf16==0x%X", utf16[0])
		return
	}
}
