package main

import (
	"bufio"
	"io"
	"unicode/utf8"
)

func lastByte(b []byte) byte {
	return b[len(b)-1]
}

func setLastByte(b []byte, lastByte byte) {
	b[len(b)-1] = lastByte
}

type Buffer struct {
	Slices [][]byte
	*bufio.Reader
	CursorY int
}

func NewBuffer(r io.Reader) *Buffer {
	return &Buffer{
		Slices:  [][]byte{},
		Reader:  bufio.NewReader(r),
		CursorY: 0,
	}
}

func (b *Buffer) Add(tmp []byte)              { b.Slices = append(b.Slices, tmp) }
func (b *Buffer) Count() int                  { return len(b.Slices) }
func (b *Buffer) Line(n int) []byte           { return b.Slices[n] }
func (b *Buffer) Byte(r, c int) byte          { return b.Slices[r][c] }
func (b *Buffer) SetByte(r, c int, data byte) { b.Slices[r][c] = data }
func (b *Buffer) WidthAt(r int) int           { return len(b.Slices[r]) }
func (b *Buffer) LastLine() []byte {
	return b.Slices[len(b.Slices)-1]
}

func (b *Buffer) Rune(r, c int) (rune, int, int) {
	// seek first
	currentPosInRune := 0
	for !utf8.RuneStart(b.Byte(r, c)) {
		c--
		if c < 0 {
			r--
			if r < 0 {
				r = 0
				c = 0
				break
			}
			c = len(b.Slices[r]) - 1
		}
		currentPosInRune++
	}
	bytes := make([]byte, 0, utf8.UTFMax)
	for {
		bytes = append(bytes, b.Byte(r, c))
		if len(bytes) >= utf8.UTFMax {
			break
		}
		c++
		if c >= len(b.Slices[r]) {
			c = 0
			r++
			if r >= len(b.Slices) {
				break
			}
		}
		if utf8.RuneStart(b.Byte(r, c)) {
			break
		}
	}
	theRune, theLen := utf8.DecodeRune(bytes)
	if currentPosInRune >= theLen {
		return utf8.RuneError, 0, 1
	}
	return theRune, currentPosInRune, theLen
}

func (b *Buffer) SetLastLine(line []byte) {
	b.Slices[len(b.Slices)-1] = line
}
func (b *Buffer) DropLastLine() {
	b.Slices = b.Slices[:len(b.Slices)-1]
}

func (b *Buffer) Shift(r int, appendByte byte) (deleteByte byte) {
	deleteByte = b.Slices[r][0]
	copy(b.Slices[r][:], b.Slices[r][1:])
	setLastByte(b.Slices[r], appendByte)
	return
}

func (b *Buffer) Unshift(r int, appendByte byte) (deleteByte byte) {
	deleteByte = lastByte(b.Slices[r])
	copy(b.Slices[r][1:], b.Slices[r])
	b.Slices[r][0] = appendByte
	return
}

func (b *Buffer) appendLine() error {
	var slice1 [LINE_SIZE]byte
	n, err := b.Read(slice1[:])
	if n > 0 {
		b.Add(slice1[:n])
	}
	return err
}

func (b *Buffer) appendTail() error {
	last := b.LastLine()

	slice1 := make([]byte, LINE_SIZE-len(last))
	n, err := b.Read(slice1)
	if n > 0 {
		last = append(last, slice1[:n]...)
		b.SetLastLine(last)
	}
	return err
}

func (b *Buffer) Fetch() ([]byte, int, error) {
	if b.CursorY >= len(b.Slices) {
		if b.Reader == nil {
			return nil, b.CursorY * LINE_SIZE, io.EOF
		}
		var err error
		if b.Slices == nil || len(b.Slices) <= 0 ||
			len(b.Slices[len(b.Slices)-1]) == LINE_SIZE {
			err = b.appendLine()
		} else {
			err = b.appendTail()
		}
		if err != nil {
			b.Reader = nil
			if err != io.EOF {
				return nil, 0, err
			}
		}
	}
	if b.CursorY >= len(b.Slices) {
		return nil, 0, io.EOF
	}
	bin := b.Line(b.CursorY)
	b.CursorY++
	return bin, (b.CursorY - 1) * LINE_SIZE, nil
}

func (b *Buffer) ReadAll() {
	if b.Reader == nil {
		return
	}
	for {
		var data [LINE_SIZE]byte
		n, err := b.Read(data[:])
		if n > 0 {
			b.Add(data[:n])
		}
		if err != nil {
			b.Reader = nil
			break
		}
	}
}
