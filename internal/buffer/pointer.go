package buffer

import (
	"container/list"
	"io"
)

type Pointer struct {
	buffer  *Buffer
	address int64
	element *list.Element
	offset  int
}

func (p *Pointer) Clone() *Pointer {
	clone := *p
	return &clone
}

func (p *Pointer) Address() int64 { return p.address }

func NewPointer(b *Buffer) *Pointer {
	element := b.lines.Front()
	if element == nil {
		if err := b.Fetch(); err != nil && err != io.EOF {
			return nil
		}
		element = b.lines.Front()
		if element == nil {
			return nil
		}
	}
	return &Pointer{
		buffer:  b,
		address: 0,
		element: element,
		offset:  0,
	}
}

func (p *Pointer) Value() byte {
	return p.element.Value.(_Block)[p.offset]
}

func (p *Pointer) SetValue(value byte) {
	p.element.Value.(_Block)[p.offset] = value
}

func (p *Pointer) Prev() error {
	return p.Rewind(1)
}

func (p *Pointer) Next() error {
	return p.Skip(1)
}

func (p *Pointer) Rewind(n int64) error {
	for {
		if n <= int64(p.offset) {
			p.offset -= int(n)
			p.address -= n
			return nil
		}
		prevElement := p.element.Prev()
		if prevElement == nil {
			return io.EOF
		}
		p.address -= int64(p.offset)
		n -= int64(p.offset)
		p.element = prevElement
		p.offset = len(p.element.Value.(_Block))
	}
}

func (p *Pointer) Skip(n int64) error {
	for {
		if int64(p.offset)+n < int64(len(p.element.Value.(_Block))) {
			p.offset += int(n)
			p.address += n
			return nil
		}
		nextElement := p.element.Next()
		if nextElement == nil {
			if err := p.buffer.Fetch(); err != nil {
				return err
			}
			nextElement = p.buffer.lines.Back()
		}
		moveBytes := len(p.element.Value.(_Block)) - p.offset
		n -= int64(moveBytes)
		p.element = nextElement
		p.offset = 0
		p.address += int64(moveBytes)
	}
}

func (p *Pointer) GoEndOfFile() {
	p.buffer.ReadAll()
	p.element = p.buffer.lines.Back()
	p.address = p.buffer.AllBytes() - 1
	p.offset = len(p.element.Value.(_Block)) - 1
}

func (p *Pointer) Insert(value byte) {
	block := p.element.Value.(_Block)
	block = append(block, 0)
	copy(block[p.offset+1:], block[p.offset:])
	block[p.offset] = value
	p.element.Value = _Block(block)
}

func (p *Pointer) Append(value byte) {
	block := p.element.Value.(_Block)
	if len(block) == p.offset+1 {
		block = append(block, value)
	} else {
		block = append(block, 0)
		copy(block[p.offset+2:], block[p.offset+1:])
		block[p.offset+1] = value
	}
	p.element.Value = _Block(block)
}

const (
	DeleteSuccess = iota
	DeleteAll
	DeleteRefresh
)

func (p *Pointer) Delete() int {
	block := p.element.Value.(_Block)
	if len(block) <= 1 {
		defer p.buffer.lines.Remove(p.element)
		if next := p.element.Next(); next != nil {
			p.element = next
			p.offset = 0
			return DeleteSuccess
		} else if prev := p.element.Prev(); prev != nil {
			p.element = prev
			p.address--
			p.offset = len(p.element.Value.(_Block)) - 1
			return DeleteRefresh
		} else {
			return DeleteAll
		}
	}
	copy(block[p.offset:], block[p.offset+1:])
	block = block[:len(block)-1]
	p.element.Value = _Block(block)
	if p.offset >= len(block) {
		p.offset = len(block) - 1
		p.address--
	}
	return DeleteSuccess
}
