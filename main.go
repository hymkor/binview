package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/zetamatta/go-readline-ny"

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
		c, length := utf8.DecodeRune(slice[i:])
		if c == utf8.RuneError || c < ' ' {
			c = '.'
			length = 1
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

func view(fetch func() ([]byte, int, error), csrpos, csrlin, w, h int, out io.Writer) (int, error) {
	count := 0
	lfCount := 0
	for {
		if count >= h {
			return lfCount, nil
		}
		record, address, err := fetch()
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
)

func readAll(reader io.Reader, slices [][]byte) [][]byte {
	for {
		var data [LINE_SIZE]byte
		n, err := reader.Read(data[:])
		if n > 0 {
			slices = append(slices, data[:n])
		}
		if err != nil {
			return slices
		}
	}
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

	slices := [][]byte{}
	reader := bufio.NewReader(pin)

	tty1, err := tty.Open()
	if err != nil {
		return err
	}
	defer tty1.Close()

	colIndex := 0
	rowIndex := 0
	startRow := 0

	var lastWidth, lastHeight int

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
		y := startRow
		fetch := func() ([]byte, int, error) {
			if y >= len(slices) {
				if reader == nil {
					return nil, y * LINE_SIZE, io.EOF
				}
				var slice1 [LINE_SIZE]byte
				n, err := reader.Read(slice1[:])
				if n > 0 {
					slices = append(slices, slice1[:n])
				}
				if err != nil {
					reader = nil
				}
			}
			if y >= len(slices) {
				return nil, 0, io.EOF
			}
			bin := slices[y]
			y++
			return bin, (y - 1) * LINE_SIZE, nil
		}
		lf, err := view(fetch, colIndex, rowIndex-startRow, screenWidth-1, screenHeight-1, out)
		if err != nil {
			return err
		}
		if len(slices) <= 0 {
			return nil
		}
		fmt.Fprintln(out, "\r") // \r is for Linux & go-tty
		lf++
		if message != "" {
			io.WriteString(out, _ANSI_YELLOW)
			io.WriteString(out, runewidth.Truncate(message, screenWidth-1, ""))
			io.WriteString(out, _ANSI_RESET)
			message = ""
		} else if 0 <= rowIndex && rowIndex < len(slices) {
			if 0 <= colIndex && colIndex < len(slices[rowIndex]) {
				fmt.Fprintf(out, "\x1B[0;33;1m(%08X):%02X\x1B[0m",
					rowIndex*LINE_SIZE+colIndex,
					slices[rowIndex][colIndex])
			}
		}
		fmt.Fprint(out, ERASE_SCRN_AFTER)
		ch, err := readline.GetKey(tty1)
		if err != nil {
			return err
		}
		switch ch {
		case _KEY_CTRL_L:
			cache = map[int]string{}
		case "q", _KEY_ESC:
			io.WriteString(out, _ANSI_YELLOW+"\rQuit Sure ? [y/n]"+ERASE_LINE)
			if ch, err := readline.GetKey(tty1); err == nil && ch == "y" {
				io.WriteString(out, "\n")
				return nil
			}
		case "j", _KEY_DOWN, _KEY_CTRL_N:
			if rowIndex < len(slices)-1 {
				rowIndex++
			} else if _, _, err := fetch(); err == nil {
				rowIndex++
			}
		case "k", _KEY_UP, _KEY_CTRL_P:
			if rowIndex > 0 {
				rowIndex--
			}
		case "h", _KEY_LEFT, _KEY_CTRL_B:
			if colIndex > 0 {
				colIndex--
			}
		case "l", _KEY_RIGHT, _KEY_CTRL_F:
			colIndex++
		case "0", "^", _KEY_CTRL_A:
			colIndex = 0
		case "$", _KEY_CTRL_E:
			colIndex = len(slices[rowIndex]) - 1
		case "<":
			rowIndex = 0
			colIndex = 0
		case ">":
			if reader != nil {
				slices = readAll(reader, slices)
			}
			rowIndex = len(slices) - 1
			colIndex = len(slices[rowIndex]) - 1
			reader = nil
		case "x":
			if colIndex < LINE_SIZE {
				csrline := slices[rowIndex]
				copy(csrline[colIndex+1:], csrline[colIndex:])
			}
			for i := rowIndex; i+1 < len(slices); i++ {
				slices[i][len(slices[i])-1] = slices[i+1][0]
				copy(slices[i+1][:], slices[i+1][1:])
			}
			last := slices[len(slices)-1]
			if reader != nil {
				reader.Read(last[len(last)-1:])
			} else {
				if len(last) > 1 {
					slices[len(slices)-1] = last[:len(last)-1]
				} else {
					slices = slices[:len(slices)-1]
					if len(slices) <= 0 {
						return nil
					}
					if rowIndex >= len(slices) {
						rowIndex--
						colIndex = len(slices[len(slices)-1]) - 1
					}
				}
			}
		case "w":
			fname := "output.new"
			var err error
			if len(args) >= 1 {
				fname, err = filepath.Abs(args[0])
				if err != nil {
					message = err.Error()
					break
				}
				fname += ".new"
			}
			fname, err = getline(out, "write to>", fname)
			if err != nil {
				break
			}
			if reader != nil {
				slices = readAll(reader, slices)
			}
			fd, err := os.OpenFile(fname, os.O_EXCL|os.O_CREATE, 0666)
			if err != nil {
				message = err.Error()
				break
			}
			for _, s := range slices {
				fd.Write(s)
			}
			fd.Close()
		}
		if colIndex >= len(slices[rowIndex]) {
			colIndex = len(slices[rowIndex]) - 1
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
