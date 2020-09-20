package main

import (
	"bufio"
	"flag"
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

type LineView struct {
	Slice     []byte
	CursorPos int
	Out       io.Writer
}

// See. en.wikipedia.org/wiki/Unicode_control_characters#Control_pictures

func (v LineView) Draw(address int) {
	fmt.Fprintf(v.Out, "%08X ", address)
	for i, s := range v.Slice {
		if i > 0 {
			io.WriteString(v.Out, "\x1B[0m ")
		}
		if i == v.CursorPos {
			io.WriteString(v.Out, CURSOR_COLOR)
		} else if ((i >> 2) & 1) == 0 {
			io.WriteString(v.Out, CELL1_COLOR)
		} else {
			io.WriteString(v.Out, CELL2_COLOR)
		}
		fmt.Fprintf(v.Out, "%02X", s)
	}
	v.Out.Write([]byte{' '})
	for i, s := range v.Slice {
		if i == v.CursorPos {
			io.WriteString(v.Out, CURSOR_COLOR)
		} else {
			io.WriteString(v.Out, CELL1_COLOR)
		}
		if s < ' ' || s >= 0x7F {
			s = '.'
		}
		v.Out.Write([]byte{s})
	}
	io.WriteString(v.Out, ERASE_LINE)
}

type BinIn interface {
	Read() ([]byte, error)
	HomeAddress() int
}

var cache = map[int]string{}

const CELL_WIDTH = 12

func view(in BinIn, csrpos, csrlin, w, h int, out io.Writer) (int, error) {
	count := 0
	lfCount := 0
	homeAddress := in.HomeAddress()
	for {
		if count >= h {
			return lfCount, nil
		}
		record, err := in.Read()
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
		var buffer strings.Builder
		v := LineView{
			Slice: record,
			Out:   &buffer,
		}
		if count == csrlin {
			v.CursorPos = csrpos
		} else {
			v.CursorPos = -1
		}

		v.Draw((homeAddress + count) * 16)
		line := buffer.String()
		if f := cache[count]; f != line {
			io.WriteString(out, line)
			cache[count] = line
		}
		count++
	}
}

type MemoryBin struct {
	Data   [][]byte
	StartY int
}

func (this *MemoryBin) Read() ([]byte, error) {
	if this.StartY >= len(this.Data) {
		return nil, io.EOF
	}
	bin := this.Data[this.StartY]
	this.StartY++
	return bin, nil
}

func (this *MemoryBin) HomeAddress() int {
	return this.StartY
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

func getIn() io.ReadCloser {
	pin, pout := io.Pipe()
	go func() {
		args := flag.Args()
		if len(args) <= 0 {
			io.Copy(pout, os.Stdin)
		} else {
			for _, arg1 := range args {
				in, err := os.Open(arg1)
				if err != nil {
					fmt.Fprintf(pout, "\"%s\",\"not found\"\n", arg1)
					continue
				}
				io.Copy(pout, in)
				in.Close()
			}
		}
		pout.Close()
	}()
	return pin
}

func main1() error {
	out := colorable.NewColorableStderr()

	io.WriteString(out, _ANSI_CURSOR_OFF)
	defer io.WriteString(out, _ANSI_CURSOR_ON)

	pin := getIn()
	defer pin.Close()

	in := bufio.NewReader(pin)
	slices := [][]byte{}
	for {
		var slice1 [16]byte
		n, err := in.Read(slice1[:])
		if n > 0 {
			slices = append(slices, slice1[:n])
		}
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
	}
	if len(slices) <= 0 {
		return io.EOF
	}
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
		window := &MemoryBin{Data: slices, StartY: startRow}
		lf, err := view(window, colIndex, rowIndex-startRow, screenWidth-1, screenHeight-1, out)
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
		case ">":
			rowIndex = len(slices) - 1
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
	flag.Parse()
	if err := main1(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
