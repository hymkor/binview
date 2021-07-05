package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"
	"github.com/mattn/go-tty"
)

const (
	CURSOR_COLOR_ON  = "\x1B[37;40;1;7m"
	CURSOR_COLOR_OFF = "\x1B[27;22m"
	CELL1_COLOR_ON   = "\x1B[37;40;22m"
	CELL1_COLOR_OFF  = ""
	CELL2_COLOR_ON   = "\x1B[37;40;1m"
	CELL2_COLOR_OFF  = "\x1B[22m"
	ERASE_LINE       = "\x1B[0K"
	ERASE_SCRN_AFTER = "\x1B[0J"
)

const LINE_SIZE = 16

// See. en.wikipedia.org/wiki/Unicode_control_characters#Control_pictures

func draw(out io.Writer, address int, cursorPos int, slice []byte) {
	if cursorPos >= 0 {
		io.WriteString(out, _ANSI_UNDERLINE_ON)
		defer io.WriteString(out, _ANSI_UNDERLINE_OFF)
	}
	fmt.Fprintf(out, "%s%08X%s ", CELL2_COLOR_ON, address, CELL2_COLOR_OFF)
	for i, s := range slice {
		var fieldSeperator string
		if i > 0 {
			fieldSeperator = " "
		}
		var on, off string
		if i == cursorPos {
			on = CURSOR_COLOR_ON
			off = CURSOR_COLOR_OFF
		} else if ((i >> 2) & 1) == 0 {
			on = CELL1_COLOR_ON
			off = CELL1_COLOR_OFF
		} else {
			on = CELL2_COLOR_ON
			off = CELL2_COLOR_OFF
		}
		fmt.Fprintf(out, "%s%s%02X%s", fieldSeperator, on, s, off)
	}
	io.WriteString(out, " ")
	for i := len(slice); i < LINE_SIZE; i++ {
		io.WriteString(out, "   ")
	}

	for i := 0; i < len(slice); {
		c := rune(slice[i])
		length := 1
		if c < ' ' || c == '\u007F' {
			c = '.'
		} else if c >= utf8.RuneSelf {
			c, length = utf8.DecodeRune(slice[i:])
			if c == utf8.RuneError {
				c = '.'
			}
		}
		var on, off, padding string
		if i <= cursorPos && cursorPos < i+length {
			on = CURSOR_COLOR_ON
			off = CURSOR_COLOR_OFF
		} else {
			on = CELL1_COLOR_ON
			off = CELL1_COLOR_OFF
		}
		if length == 3 {
			padding = " "
		} else if length == 4 {
			padding = "  "
		}
		fmt.Fprintf(out, "%s%c%s%s", on, c, off, padding)
		i += length
	}
	io.WriteString(out, ERASE_LINE)
}

var cache = map[int]string{}

const CELL_WIDTH = 12

func (b *Buffer) View(csrpos, csrlin, w, h int, out io.Writer) (int, error) {
	count := 0
	lfCount := 0
	for {
		if count >= h {
			return lfCount, nil
		}
		record, address, err := b.Fetch()
		if err == io.EOF {
			return lfCount, nil
		}
		if err != nil {
			return lfCount, err
		}
		if count > 0 {
			lfCount++
			fmt.Fprintln(out, "\r") // "\r" is for Linux and go-tty
		}
		var cursorPos int
		if count == csrlin {
			cursorPos = csrpos
		} else {
			cursorPos = -1
		}
		var buffer strings.Builder
		draw(&buffer, address, cursorPos, record)
		line := buffer.String()
		if f := cache[count]; f != line {
			io.WriteString(out, line)
			cache[count] = line
		}
		count++
	}
}

const (
	_ANSI_CURSOR_OFF    = "\x1B[?25l"
	_ANSI_CURSOR_ON     = "\x1B[?25h"
	_ANSI_YELLOW        = "\x1B[0;33;1m"
	_ANSI_RESET         = "\x1B[0m"
	_ANSI_UNDERLINE_ON  = "\x1B[4m"
	_ANSI_UNDERLINE_OFF = "\x1B[24m"
)

const (
	_KEY_CTRL_A = "\x01"
	_KEY_CTRL_B = "\x02"
	_KEY_CTRL_E = "\x05"
	_KEY_CTRL_F = "\x06"
	_KEY_CTRL_L = "\x0C"
	_KEY_CTRL_N = "\x0E"
	_KEY_CTRL_P = "\x10"
	_KEY_DOWN   = "\x1B[B"
	_KEY_ESC    = "\x1B"
	_KEY_LEFT   = "\x1B[D"
	_KEY_RIGHT  = "\x1B[C"
	_KEY_UP     = "\x1B[A"
	_KEY_F2     = "\x1B[OQ"
	_KEY_DEL    = "\x1B[3~"
)

const (
	UNCHANGED = ' '
	CHANGED   = '*'
)

type Clip struct {
	data []byte
}

func NewClip() *Clip {
	return &Clip{data: make([]byte, 0, 100)}
}

func (c *Clip) Push(n byte) {
	c.data = append(c.data, n)
}

func (c *Clip) Pop() byte {
	var newByte byte
	if len(c.data) > 0 {
		tail := len(c.data) - 1
		newByte = c.data[tail]
		c.data = c.data[:tail]
	}
	return newByte
}

func (c *Clip) Len() int {
	return len(c.data)
}

func mains(args []string) error {
	disable := colorable.EnableColorsStdout(nil)
	if disable != nil {
		defer disable()
	}
	out := colorable.NewColorableStdout()

	io.WriteString(out, _ANSI_CURSOR_OFF)
	defer io.WriteString(out, _ANSI_CURSOR_ON)

	pin, err := NewArgf(args)
	if err != nil {
		return err
	}
	defer pin.Close()

	buffer := NewBuffer(pin)

	tty1, err := tty.Open()
	if err != nil {
		return err
	}
	defer tty1.Close()

	colIndex := 0
	rowIndex := 0
	startRow := 0

	var lastWidth, lastHeight int

	clipBoard := NewClip()

	isChanged := UNCHANGED
	message := ""
	for {
		screenWidth, screenHeight, err := tty1.Size()
		if err != nil {
			return err
		}
		if lastWidth != screenWidth || lastHeight != screenHeight {
			cache = map[int]string{}
			lastWidth = screenWidth
			lastHeight = screenHeight
			io.WriteString(out, _ANSI_CURSOR_OFF)
		}
		buffer.CursorY = startRow
		fetch := func() ([]byte, int, error) {
			return buffer.Fetch()
		}
		lf, err := buffer.View(colIndex, rowIndex-startRow, screenWidth-1, screenHeight-1, out)
		if err != nil {
			return err
		}
		if buffer.Count() <= 0 {
			return nil
		}
		fmt.Fprintln(out, "\r") // \r is for Linux & go-tty
		lf++
		if message != "" {
			io.WriteString(out, _ANSI_YELLOW)
			io.WriteString(out, runewidth.Truncate(message, screenWidth-1, ""))
			io.WriteString(out, _ANSI_RESET)
			message = ""
		} else if 0 <= rowIndex && rowIndex < buffer.Count() {
			if 0 <= colIndex && colIndex < buffer.WidthAt(rowIndex) {
				fmt.Fprintf(out, "\x1B[0;33;1m%[3]c(%08[1]X):0x%02[2]X=%-4[2]d",
					rowIndex*LINE_SIZE+colIndex,
					buffer.Byte(rowIndex, colIndex),
					isChanged)

				theRune, thePosInRune, theLenOfRune := buffer.Rune(rowIndex, colIndex)
				if theRune != utf8.RuneError {
					fmt.Fprintf(out, "(%d/%d:U+%X)",
						thePosInRune+1,
						theLenOfRune,
						theRune)
				} else {
					io.WriteString(out, "(not UTF8)")
				}
				io.WriteString(out, "\x1B[0m")
			}
		}
		fmt.Fprint(out, ERASE_SCRN_AFTER)
		ch, err := getkey(tty1)
		if err != nil {
			return err
		}
		var newByte byte = 0
		switch ch {
		case _KEY_CTRL_L:
			cache = map[int]string{}
		case "q", _KEY_ESC:
			if yesNo(tty1, out, "Quit Sure ? [y/n]") {
				io.WriteString(out, "\n")
				return nil
			}
		case "j", _KEY_DOWN, _KEY_CTRL_N:
			if rowIndex < buffer.Count()-1 {
				rowIndex++
			} else if _, _, err := fetch(); err == nil {
				rowIndex++
			} else if err != io.EOF {
				return err
			}
		case "k", _KEY_UP, _KEY_CTRL_P:
			if rowIndex > 0 {
				rowIndex--
			}
		case "h", "\b", _KEY_LEFT, _KEY_CTRL_B:
			if colIndex > 0 {
				colIndex--
			} else if rowIndex > 0 {
				rowIndex--
				colIndex = LINE_SIZE - 1
			}
		case "l", " ", _KEY_RIGHT, _KEY_CTRL_F:
			if colIndex < LINE_SIZE-1 {
				colIndex++
			} else if rowIndex < buffer.Count()-1 {
				rowIndex++
				colIndex = 0
			} else if _, _, err := fetch(); err == nil {
				rowIndex++
				colIndex = 0
			} else if err != io.EOF {
				return err
			}
		case "0", "^", _KEY_CTRL_A:
			colIndex = 0
		case "$", _KEY_CTRL_E:
			colIndex = buffer.WidthAt(rowIndex) - 1
		case "<":
			rowIndex = 0
			colIndex = 0
		case ">", "G":
			buffer.ReadAll()
			rowIndex = buffer.Count() - 1
			colIndex = buffer.WidthAt(rowIndex) - 1
			buffer.Reader = nil
		case "p":
			if clipBoard.Len() <= 0 {
				break
			}
			newByte = clipBoard.Pop()
			fallthrough
		case "a":
			appendOne(buffer, rowIndex, colIndex)
			if colIndex+1 < len(buffer.Slices[rowIndex]) {
				colIndex++
			} else {
				colIndex = 0
				rowIndex++
			}
			buffer.Slices[rowIndex][colIndex] = newByte
			isChanged = CHANGED
		case "P":
			if clipBoard.Len() <= 0 {
				break
			}
			newByte = clipBoard.Pop()
			fallthrough
		case "i":
			insertOne(buffer, rowIndex, colIndex)
			buffer.Slices[rowIndex][colIndex] = newByte
			isChanged = CHANGED
		case "x", _KEY_DEL:
			clipBoard.Push(buffer.Slices[rowIndex][colIndex])
			deleteOne(buffer, rowIndex, colIndex)
			isChanged = CHANGED
		case "w":
			if err := write(buffer, tty1, out, args); err != nil {
				message = err.Error()
			} else {
				isChanged = UNCHANGED
			}
		case "r":
			bytes, err := getline(out, "replace>",
				fmt.Sprintf("0x%02X", buffer.Byte(rowIndex, colIndex)))
			if err != nil {
				message = err.Error()
				break
			}
			if n, err := strconv.ParseUint(bytes, 0, 8); err == nil {
				buffer.SetByte(rowIndex, colIndex, byte(n))
				isChanged = CHANGED
			} else {
				message = err.Error()
			}
		}
		if buffer.Count() <= 0 {
			return nil
		}
		if rowIndex >= buffer.Count() {
			rowIndex--
			colIndex = LINE_SIZE
		}
		if colIndex >= buffer.WidthAt(rowIndex) {
			colIndex = buffer.WidthAt(rowIndex) - 1
		}

		if rowIndex < startRow {
			startRow = rowIndex
		} else if rowIndex >= startRow+screenHeight-1 {
			startRow = rowIndex - (screenHeight - 1) + 1
		}
		if lf > 0 {
			fmt.Fprintf(out, "\r\x1B[%dA", lf)
		} else {
			fmt.Fprint(out, "\r")
		}
	}
}

func main() {
	if err := mains(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
