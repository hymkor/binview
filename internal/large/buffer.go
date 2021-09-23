package large

import (
	"bufio"
	"container/list"
	"io"
)

type _Block = []byte

var ALLOC_SIZE = 4096

type Buffer struct {
	lines *list.List
	*bufio.Reader
	allsize int64
}

func NewBuffer(r io.Reader) *Buffer {
	return &Buffer{
		lines:   list.New(),
		Reader:  bufio.NewReader(r),
		allsize: 0,
	}
}

func (b *Buffer) Len() int64 {
	return b.allsize
}

func (b *Buffer) Fetch() error {
	if b.Reader == nil {
		return io.EOF
	}
	buffer := make([]byte, ALLOC_SIZE)
	n, err := b.Reader.Read(buffer)

	if n > 0 {
		b.lines.PushBack(_Block(buffer[:n]))
		b.allsize += int64(n)
	}
	if err != nil {
		b.Reader = nil
	}
	return err
}

func (b *Buffer) ReadAll() {
	for b.Fetch() == nil {
	}
}

func (b *Buffer) WriteTo(w io.Writer) (int64, error) {
	b.ReadAll()
	n := int64(0)
	for p := b.lines.Front(); p != nil; p = p.Next() {
		m, err := w.Write(p.Value.(_Block))
		n += int64(m)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
