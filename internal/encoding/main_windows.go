package encoding

import (
	"golang.org/x/sys/windows"
)

var kernel32dll = windows.NewLazySystemDLL("kernel32")
var procIsDBCSLeadByte = kernel32dll.NewProc("IsDBCSLeadByte")

func IsDBCSLeadByte(b byte) bool {
	result, _, _ := procIsDBCSLeadByte.Call(uintptr(b))
	return result != 0
}

func ToWideChar(bytes ...byte) ([]uint16, error) {
	var wideBuffer [10]uint16

	nwrite, err := windows.MultiByteToWideChar(
		windows.GetACP(),
		MB_ERR_INVALID_CHARS,
		&bytes[0],
		int32(len(bytes)),
		&wideBuffer[0],
		int32(len(wideBuffer)))

	return wideBuffer[:nwrite], err
}
