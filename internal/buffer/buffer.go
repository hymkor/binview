package buffer

import (
	"bufio"
	"container/list"
	"io"
)

type _Block = []byte

const _ALLOC_SIZE = 4096

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

func (b *Buffer) AllBytes() int64 {
	return b.allsize
}

func (b *Buffer) Fetch() error {
	if b.Reader == nil {
		return io.EOF
	}
	var buffer [_ALLOC_SIZE]byte
	n, err := b.Reader.Read(buffer[:])

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
	if b.Reader != nil {
		for b.Fetch() == nil {
		}
	}
}

func (b *Buffer) Each(f func([]byte)) {
	b.ReadAll()
	for p := b.lines.Front(); p != nil; p = p.Next() {
		f([]byte(p.Value.(_Block)))
	}
}
