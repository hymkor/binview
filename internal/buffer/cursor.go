package buffer

import (
	"container/list"
	"unicode/utf8"
)

type Cursor struct {
	buffer  *Buffer
	Index   int
	element *list.Element
}

func (c *Cursor) Clone() *Cursor {
	tmp := *c
	return &tmp
}
func (c *Cursor) Address() int64    { return int64(c.Index) * LINE_SIZE }
func (c *Cursor) Bytes() _Line      { return c.element.Value.(_Line) }
func (c *Cursor) Byte(pos int) byte { return c.Bytes()[pos] }
func (c *Cursor) SetByte(pos int, value byte) {
	c.Bytes()[pos] = value
}
func (c *Cursor) Len() int { return len(c.Bytes()) }

func (c *Cursor) Update(value _Line) {
	c.element.Value = value
}

func (c *Cursor) Chop() {
	value := c.Bytes()
	c.Update(value[:len(value)-1])
}

func (c *Cursor) Next() bool {
	next := c.element.Next()
	if next == nil {
		return false
	}
	c.Index++
	c.element = next
	return true
}

func (c *Cursor) Prev() bool {
	prev := c.element.Prev()
	if prev == nil {
		return false
	}
	c.Index--
	c.element = prev
	return true
}

func (cursor *Cursor) GotoEnd() {
	cursor.Index = cursor.buffer.Len() - 1
	cursor.element = cursor.buffer.lines.Back()
}

func (cursor Cursor) Rune(c int) (rune, int, int) {
	// Be careful: receiver is not pointer.

	// seek first
	currentPosInRune := 0
	for !utf8.RuneStart(cursor.Byte(c)) {
		c--
		if c < 0 {
			if !cursor.Prev() {
				c = 0
				break
			}
			c = cursor.Len() - 1
		}
		currentPosInRune++
	}
	bytes := make([]byte, 0, utf8.UTFMax)
	for {
		bytes = append(bytes, cursor.Byte(c))
		if len(bytes) >= utf8.UTFMax {
			break
		}
		c++
		if c >= cursor.Len() {
			c = 0
			if !cursor.Next() {
				break
			}
		}
		if utf8.RuneStart(cursor.Byte(c)) {
			break
		}
	}
	theRune, theLen := utf8.DecodeRune(bytes)
	if currentPosInRune >= theLen {
		return utf8.RuneError, 0, 1
	}
	return theRune, currentPosInRune, theLen
}

func (c *Cursor) NextOrFetch() error {
	if c.Next() {
		return nil
	}
	next, err := c.buffer.Fetch()
	if err != nil {
		return err
	}
	*c = *next
	return nil
}
