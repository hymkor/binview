package encoding

import (
	"testing"
)

var _ Encoding = UTF8Encoding{}
var _ Encoding = DBCSEncoding{}
var _ Encoding = UTF16LE{}

func TestIsDBCSLeadByte(t *testing.T) {
	if !IsDBCSLeadByte(0x83) { // Japanese katakana SO
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
		t.Fatal(err.Error())
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
