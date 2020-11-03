package main

import (
	"bufio"
	"io"
)

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
func (b *Buffer) DropLastLine() {
	b.Slices = b.Slices[:len(b.Slices)-1]
}

func (b *Buffer) Shift(r int, appendByte byte) (deleteByte byte) {
	deleteByte = b.Slices[r][0]
	copy(b.Slices[r][:], b.Slices[r][1:])
	b.Slices[r][len(b.Slices[r])-1] = appendByte
	return
}

func (b *Buffer) Unshift(r int, appendByte byte) (deleteByte byte) {
	deleteByte = b.Slices[r][len(b.Slices[r])-1]
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
	last := b.Slices[len(b.Slices)-1]

	slice1 := make([]byte, LINE_SIZE-len(last))
	n, err := b.Read(slice1)
	if n > 0 {
		last = append(last, slice1[:n]...)
		b.Slices[len(b.Slices)-1] = last
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
