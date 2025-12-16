//go:build !windows
// +build !windows

package encoding

import (
	"errors"
)

func IsDBCSLeadByte(b byte) bool {
	return false
}

func ToWideChar(bytes ...byte) ([]uint16, error) {
	return []uint16{}, ErrNotSupport
}
