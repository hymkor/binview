package main

import (
	"unicode/utf8"
)

type Cursor struct {
	buffer *Buffer
	index  int
}

func (c *Cursor) Address() int      { return c.index * LINE_SIZE }
func (c *Cursor) Bytes() Line       { return c.buffer.Line[c.index] }
func (c *Cursor) Byte(pos int) byte { return c.Bytes()[pos] }
func (c *Cursor) SetByte(pos int, value byte) {
	c.Bytes()[pos] = value
}
func (c *Cursor) Len() int { return len(c.Bytes()) }

func (c *Cursor) Update(value []byte) {
	c.buffer.Line[c.index] = value
}

func (c *Cursor) Chop() {
	value := c.buffer.Line[c.index]
	c.buffer.Line[c.index] = value[:len(value)-1]
}

func (c *Cursor) Next() bool {
	if c.index >= c.buffer.Len()-1 {
		return false
	}
	c.index++
	return true
}

func (c *Cursor) Prev() bool {
	if c.index <= 0 {
		return false
	}
	c.index--
	return true
}

func (cursor *Cursor) GotoEnd() {
	cursor.index = cursor.buffer.Len() - 1
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
