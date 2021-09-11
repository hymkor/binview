package buffer

import (
	"bufio"
	"container/list"
	"io"
)

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

func (b *Buffer) Add(tmp _Line) {
	b.lines.PushBack(tmp)
}

func (b *Buffer) Len() int {
	return b.lines.Len()
}

func (b *Buffer) AllBytes() int64 {
	n := b.Len()
	if n == 0 {
		return 0
	}
	return int64(n-1)*int64(LINE_SIZE) + int64(b.lines.Back().Value.(_Line).Len())
}

func (b *Buffer) LastLine() _Line {
	return b.lines.Back().Value.(_Line)
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
	newLine1 := newLine()
	n, err := b.Read(newLine1)
	if n > 0 {
		b.Add(newLine1[:n])
	}
	return err
}

func (b *Buffer) appendTail() error {
	last := b.LastLine()

	appendArea := make([]byte, LINE_SIZE-len(last))
	n, err := b.Read(appendArea)
	if n > 0 {
		last = append(last, appendArea[:n]...)
		b.SetLastLine(last)
	}
	return err
}

func (b *Buffer) fetch() error {
	if b.Reader == nil {
		return io.EOF
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
	return err
}

func (b *Buffer) Fetch() (*Cursor, error) {
	err := b.fetch()
	return b.End(), err
}

func (b *Buffer) ReadAll() {
	if b.Reader != nil {
		for b.fetch() == nil {
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
