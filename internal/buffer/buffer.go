package buffer

import (
	"bufio"
	"container/list"
	"io"
)

const LINE_SIZE = 16

type Buffer struct {
	lines *list.List
	*bufio.Reader
}

func NewBuffer(r io.Reader) *Buffer {
	return &Buffer{
		lines:  list.New(),
		Reader: bufio.NewReader(r),
	}
}

func (b *Buffer) Add(tmp Line) {
	b.lines.PushBack(tmp)
}

func (b *Buffer) Len() int {
	return b.lines.Len()
}

func (b *Buffer) LastLine() Line {
	return b.lines.Back().Value.(Line)
}

func (b *Buffer) SetLastLine(line []byte) {
	b.lines.Back().Value = line
}

func (b *Buffer) DropLastLine() {
	b.lines.Remove(b.lines.Back())
}

func (b *Buffer) Begin() *Cursor {
	if b.Len() <= 0 {
		p, err := b.Fetch()
		if err != nil {
			return nil
		}
		return p
	}
	return &Cursor{buffer: b, Index: 0, element: b.lines.Front()}
}
func (b *Buffer) End() *Cursor {
	return &Cursor{buffer: b, Index: b.Len() - 1, element: b.lines.Back()}
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

func (b *Buffer) Fetch() (*Cursor, error) {
	if b.Reader == nil {
		return nil, io.EOF
	}
	var err error
	if b.Len() <= 0 || b.End().Len() >= LINE_SIZE {
		err = b.appendLine()
	} else {
		err = b.appendTail()
	}
	if err != nil {
		b.Reader = nil
	}
	return b.End(), err
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

func (b *Buffer) unshiftLines(_rowIndex *Cursor, carry byte) {
	rowIndex := *_rowIndex
	for {
		carry = rowIndex.Bytes().InsertAt(0, carry)
		if !rowIndex.Next() {
			break
		}
	}
	if rowIndex.Len() < LINE_SIZE {
		rowIndex.Update(append(rowIndex.Bytes(), carry))
	} else {
		b.lines.PushBack([]byte{carry})
	}
}

func (b *Buffer) InsertAt(_rowIndex *Cursor, colIndex int, value byte) {
	b.ReadAll()
	rowIndex := *_rowIndex
	carry := rowIndex.Bytes().InsertAt(colIndex, value)
	rowIndex.Next()
	b.unshiftLines(&rowIndex, carry)
}

func (b *Buffer) DeleteAt(rowIndex *Cursor, colIndex int) {
	b.ReadAll()
	carry := byte(0)
	for p := b.End(); p.Index > rowIndex.Index; p.Prev() {
		carry = p.Bytes().RemoveAt(0, carry)
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
		if rowIndex.Index >= b.Len() {
			rowIndex.Prev()
			colIndex = len(b.LastLine()) - 1
		}
	}
}
