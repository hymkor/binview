package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"
	"github.com/mattn/go-tty"
)

const (
	CURSOR_COLOR     = "\x1B[0;40;37;1;7m"
	CELL1_COLOR      = "\x1B[0;40;37m"
	CELL2_COLOR      = "\x1B[0;40;37;1m"
	ERASE_LINE       = "\x1B[0m\x1B[0K"
	ERASE_SCRN_AFTER = "\x1B[0m\x1B[0J"
)

// See. en.wikipedia.org/wiki/Unicode_control_characters#Control_pictures

func draw(out io.Writer, address int, cursorPos int, slice []byte) {
	fmt.Fprintf(out, "%08X ", address)
	for i, s := range slice {
		if i > 0 {
			io.WriteString(out, "\x1B[0m ")
		}
		if i == cursorPos {
			io.WriteString(out, CURSOR_COLOR)
		} else if ((i >> 2) & 1) == 0 {
			io.WriteString(out, CELL1_COLOR)
		} else {
			io.WriteString(out, CELL2_COLOR)
		}
		fmt.Fprintf(out, "%02X", s)
	}
	io.WriteString(out, "\x1B[0m ")
	for i := len(slice); i < 16; i++ {
		io.WriteString(out, "   ")
	}

	for i := 0; i < len(slice); i++ {
		s := slice[i]
		length := 0
		if 0x20 <= s && s <= 0x7E {
			length = 1
		} else if 0xC2 <= s && s <= 0xDF {
			length = 2
		} else if 0xE0 <= s && s <= 0xEF {
			length = 3
		} else if 0xF0 <= s && s <= 0xF4 {
			length = 4
		}

		if i+length >= len(slice) {
			length = 0
		} else {
			for j := 1; j < length; j++ {
				if c := slice[i+j]; c < 0x80 || c > 0xBF {
					length = 0
					break
				}
			}
		}
		if length == 0 {
			if i == cursorPos {
				io.WriteString(out, CURSOR_COLOR)
			} else {
				io.WriteString(out, CELL1_COLOR)
			}
			io.WriteString(out, ".")
		} else {
			if i <= cursorPos && cursorPos < i+length {
				io.WriteString(out, CURSOR_COLOR)
			} else {
				io.WriteString(out, CELL1_COLOR)
			}
			out.Write(slice[i : i+length])
			i += length - 1
			if length == 3 {
				io.WriteString(out, " ")
			} else if length == 4 {
				io.WriteString(out, "  ")
			}
		}
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
	_ANSI_CURSOR_OFF = "\x1B[?25l"
	_ANSI_CURSOR_ON  = "\x1B[?25h"
	_ANSI_YELLOW     = "\x1B[0;33;1m"
	_ANSI_RESET      = "\x1B[0m"
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

func main1() error {
	disable := colorable.EnableColorsStdout(nil)
	if disable != nil {
		defer disable()
	}
	out := colorable.NewColorableStdout()

	io.WriteString(out, _ANSI_CURSOR_OFF)
	defer io.WriteString(out, _ANSI_CURSOR_ON)

	pin, err := NewArgf(os.Args[1:])
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
					return nil, y * 16, io.EOF
				}
				var slice1 [16]byte
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
			return bin, y * 16, nil
		}
		lf, err := view(fetch, colIndex, rowIndex-startRow, screenWidth-1, screenHeight-1, out)
		if err != nil {
			return err
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
					rowIndex*16+colIndex,
					slices[rowIndex][colIndex])
			}
		}
		fmt.Fprint(out, ERASE_SCRN_AFTER)
		ch, err := getKey(tty1)
		if err != nil {
			return err
		}
		switch ch {
		case _KEY_CTRL_L:
			cache = map[int]string{}
		case "q", _KEY_ESC:
			io.WriteString(out, _ANSI_YELLOW+"\rQuit Sure ? [y/n]"+ERASE_LINE)
			if ch, err := getKey(tty1); err == nil && ch == "y" {
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
				for {
					var data [16]byte
					n, err := reader.Read(data[:])
					if n > 0 {
						slices = append(slices, data[:n])
					}
					if err != nil {
						break
					}
				}
			}
			rowIndex = len(slices) - 1
			colIndex = len(slices[rowIndex]) - 1
			reader = nil
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
	if err := main1(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
