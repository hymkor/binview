package main

import (
	"bufio"
	"io"
)

type Buffer struct {
	Line []Line
	*bufio.Reader
}

func NewBuffer(r io.Reader) *Buffer {
	return &Buffer{
		Line:   []Line{},
		Reader: bufio.NewReader(r),
	}
}

func (b *Buffer) Add(tmp Line) { b.Line = append(b.Line, tmp) }
func (b *Buffer) Len() int     { return len(b.Line) }
func (b *Buffer) LastLine() Line {
	return b.Line[len(b.Line)-1]
}

func (b *Buffer) SetLastLine(line []byte) {
	b.Line[len(b.Line)-1] = line
}

func (b *Buffer) DropLastLine() {
	b.Line = b.Line[:len(b.Line)-1]
}

func (b *Buffer) Begin() *Cursor {
	return &Cursor{buffer: b, index: 0}
}
func (b *Buffer) End() *Cursor {
	return &Cursor{buffer: b, index: b.Len() - 1}
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

func (b *Buffer) fetch() (*Cursor, error) {
	if b.Reader == nil {
		return nil, io.EOF
	}
	var err error
	if b.Line == nil || len(b.Line) <= 0 ||
		len(b.Line[len(b.Line)-1]) == LINE_SIZE {
		err = b.appendLine()
	} else {
		err = b.appendTail()
	}
	if err != nil {
		b.Reader = nil
	}
	return &Cursor{buffer: b, index: len(b.Line) - 1}, err
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

func (b *Buffer) UnshiftLines(_rowIndex *Cursor, carry byte) {
	rowIndex := *_rowIndex
	for {
		carry = rowIndex.Bytes().Unshift(carry)
		if !rowIndex.Next() {
			break
		}
	}
	if rowIndex.Len() < LINE_SIZE {
		rowIndex.Update(append(rowIndex.Bytes(), carry))
	} else {
		b.Line = append(b.Line, []byte{carry})
	}
}

func (b *Buffer) InsertAt(_rowIndex *Cursor, colIndex int, value byte) {
	b.ReadAll()
	rowIndex := *_rowIndex
	carry := rowIndex.Bytes().InsertAt(colIndex, value)
	rowIndex.Next()
	b.UnshiftLines(&rowIndex, carry)
}

func (b *Buffer) deleteOne(rowIndex *Cursor, colIndex int) {
	b.ReadAll()
	carry := byte(0)
	for p := b.End(); p.index > rowIndex.index; p.Prev() {
		carry = p.Bytes().Shift(carry)
	}
	if colIndex < LINE_SIZE {
		copy(rowIndex.Bytes()[colIndex:], rowIndex.Bytes()[colIndex+1:])
	}
	rowIndex.Bytes().SetLastByte(carry)

	end := b.End()
	if end.Bytes().Len() > 1 {
		end.Chop()
	} else {
		b.DropLastLine()
		if b.Len() <= 0 {
			return
		}
		if rowIndex.index >= b.Len() {
			rowIndex.Prev()
			colIndex = len(b.LastLine()) - 1
		}
	}
}
