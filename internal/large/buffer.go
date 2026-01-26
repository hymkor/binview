package large

import (
	"container/list"
	"errors"
	"io"
	"os"
)

type _Block = []byte

type Buffer struct {
	lines        *list.List
	allsize      int64
	FetchFunc    func() ([]byte, error)
	TryFetchFunc func() ([]byte, error)
}

func NewBuffer(r io.Reader) *Buffer {
	f := &bufferFetch{
		reader:    r,
		allocSize: 8,
	}
	return &Buffer{
		lines:        list.New(),
		allsize:      0,
		FetchFunc:    f.FetchOnly,
		TryFetchFunc: f.FetchOnly,
	}
}

func (b *Buffer) Len() int64 {
	return b.allsize
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
	data, err := b.FetchFunc()
	b.StoreOnly(data, err)
	return err
}

func (b *Buffer) tryFetch() error {
	data, err := b.TryFetchFunc()
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
