package main

import (
	"bufio"
	"io"
	"unicode/utf8"
)

type Buffer struct {
	Line []Line
	*bufio.Reader
	CursorY int
}

func NewBuffer(r io.Reader) *Buffer {
	return &Buffer{
		Line:    []Line{},
		Reader:  bufio.NewReader(r),
		CursorY: 0,
	}
}

func (b *Buffer) Add(tmp Line)                { b.Line = append(b.Line, tmp) }
func (b *Buffer) Count() int                  { return len(b.Line) }
func (b *Buffer) Byte(r, c int) byte          { return b.Line[r][c] }
func (b *Buffer) SetByte(r, c int, data byte) { b.Line[r][c] = data }
func (b *Buffer) LastLine() Line {
	return b.Line[len(b.Line)-1]
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
			c = len(b.Line[r]) - 1
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
		if c >= len(b.Line[r]) {
			c = 0
			r++
			if r >= len(b.Line) {
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
	b.Line[len(b.Line)-1] = line
}
func (b *Buffer) DropLastLine() {
	b.Line = b.Line[:len(b.Line)-1]
}

func (b *Buffer) Shift(r int, appendByte byte) byte {
	return b.Line[r].Shift(appendByte)
}

func (b *Buffer) Unshift(r int, appendByte byte) byte {
	return b.Line[r].Unshift(appendByte)
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

func (b *Buffer) PreFetch() ([]byte, int, error) {
	if b.CursorY >= len(b.Line) {
		if b.Reader == nil {
			return nil, b.CursorY * LINE_SIZE, io.EOF
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
			if err != io.EOF {
				return nil, 0, err
			}
		}
	}
	if b.CursorY >= len(b.Line) {
		return nil, 0, io.EOF
	}
	bin := b.Line[b.CursorY]
	return bin, b.CursorY * LINE_SIZE, nil
}

func (b *Buffer) Fetch() ([]byte, int, error) {
	bin, size, err := b.PreFetch()
	if err != nil {
		return bin, size, err
	}
	b.CursorY++
	return bin, size, nil
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

func (b *Buffer) UnshiftLines(rowIndex int, carry byte) {
	for i := rowIndex; i < b.Count(); i++ {
		carry = b.Unshift(i, carry)
	}
	last := b.Line[b.Count()-1]
	if len(last) < LINE_SIZE {
		last = append(last, carry)
		b.Line[b.Count()-1] = last
	} else {
		b.Line = append(b.Line, []byte{carry})
	}
}

func (b *Buffer) InsertAt(rowIndex, colIndex int, value byte) {
	b.ReadAll()
	carry := b.Line[rowIndex].InsertAt(colIndex, value)
	b.UnshiftLines(rowIndex+1, carry)
}

func (b *Buffer) deleteOne(rowIndex, colIndex int) {
	b.ReadAll()
	carry := byte(0)
	for i := b.Count() - 1; i > rowIndex; i-- {
		carry = b.Shift(i, carry)
	}
	csrline := b.Line[rowIndex]
	if colIndex < LINE_SIZE {
		copy(csrline[colIndex:], csrline[colIndex+1:])
	}
	csrline.SetLastByte(carry)

	last := b.Line[len(b.Line)-1]
	if len(last) > 1 {
		b.Line[len(b.Line)-1] = last[:len(last)-1]
	} else {
		b.DropLastLine()
		if b.Count() <= 0 {
			return
		}
		if rowIndex >= b.Count() {
			rowIndex--
			colIndex = len(b.LastLine()) - 1
		}
	}
}
