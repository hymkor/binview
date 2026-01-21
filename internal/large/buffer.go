package large

import (
	"container/list"
	"errors"
	"io"
	"os"
)

type _Block = []byte

var ALLOC_SIZE = 4096

type Buffer struct {
	lines          *list.List
	reader         io.Reader
	allsize        int64
	allocSize      int
	CustomFetch    func() ([]byte, error)
	CustomTryFetch func() ([]byte, error)
}

func NewBuffer(r io.Reader) *Buffer {
	return &Buffer{
		lines:     list.New(),
		reader:    r,
		allsize:   0,
		allocSize: 8,
	}
}

func (b *Buffer) Len() int64 {
	return b.allsize
}

func (b *Buffer) FetchOnly() ([]byte, error) {
	if b.reader == nil {
		return nil, io.EOF
	}
	if b.allocSize*2 <= ALLOC_SIZE {
		b.allocSize *= 2
	}
	buffer := make([]byte, b.allocSize)
	n, err := b.reader.Read(buffer)
	if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
		b.reader = nil
	}
	return buffer[:n], err
}

func (b *Buffer) StoreOnly(data []byte, err error) bool {
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrDeadlineExceeded) {
		return false
	}
	if len(data) > 0 {
		b.lines.PushBack(_Block(data))
		b.allsize += int64(len(data))
	}
	return !errors.Is(err, os.ErrDeadlineExceeded)
}

func (b *Buffer) Fetch() error {
	var data []byte
	var err error
	if b.CustomFetch != nil {
		data, err = b.CustomFetch()
	} else {
		data, err = b.FetchOnly()
	}
	b.StoreOnly(data, err)
	return err
}

func (b *Buffer) tryFetch() error {
	var data []byte
	var err error
	if b.CustomTryFetch != nil {
		data, err = b.CustomTryFetch()
	} else {
		data, err = b.FetchOnly()
	}
	b.StoreOnly(data, err)
	return err
}

func (b *Buffer) ReadAll() error {
	for {
		err := b.Fetch()
		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func (b *Buffer) WriteTo(w io.Writer) (int64, error) {
	if err := b.ReadAll(); err != nil {
		return 0, err
	}
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
