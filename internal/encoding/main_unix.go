//go:build !windows
// +build !windows

package encoding

func IsDBCSLeadByte(b byte) bool {
	return false
}

func ToWideChar(bytes ...byte) ([]uint16, error) {
	return []uint16{}, ErrNotSupport
}
