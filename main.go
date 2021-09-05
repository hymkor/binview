package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// for Line feed
const (
	_ARROW_POINTING_DOWNWARDS_THEN_CURVING_LEFTWARDS = '\u2936'
	_DOWNWARDS_ARROW_WITH_CORNER_LEFTWARDS           = '\u21B5'
	_DOWNWARDS_ARROW_WITH_TIP_LEFTWARDS              = '\u21B2'
	_RETURN_SYMBOL                                   = '\u23CE'
	_SYMBOL_FOR_NEWLINE                              = '\u2424'
	_SYMBOL_FOR_LINE_FEED                            = '\u240A'
	_DOWNWARDS_ARROW                                 = '\u2193' // wide
	_HALFWIDTH_DOWNWARDS_ARROW                       = '\uFFEC'
)

// for carriage return
const (
	_SYMBOL_FOR_CARRIAGE_RETURN = '\u240D' // CR
	_LEFTWARDS_ARROW            = '\u2190' // wide
	_HALFWIDTH_LEFTWARDS_ARROW  = '\uFFE9' // <-
)

// for tab
const (
	_SYMBOL_FOR_HORIZONTAL_TABULATION        = '\u2409' // HT
	_RIGHTWARDS_ARROW_TO_BAR                 = '\u21E5' // ->|
	_RIGHTWARDS_TRIANGLE_HEADED_ARROW_TO_BAR = '\u2B72' // ->|
)

const LINE_SIZE = 16

// See. en.wikipedia.org/wiki/Unicode_control_characters#Control_pictures

func draw(out io.Writer, address int, cursorPos int, current []byte, next []byte) {
	if cursorPos >= 0 {
		io.WriteString(out, _ANSI_UNDERLINE_ON)
		defer io.WriteString(out, _ANSI_UNDERLINE_OFF)
	}
	fmt.Fprintf(out, "%s%08X%s ", CELL2_COLOR_ON, address, CELL2_COLOR_OFF)
	for i, s := range current {
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
	for i := len(current); i < LINE_SIZE; i++ {
		io.WriteString(out, "   ")
	}

	var joinline [LINE_SIZE * 2]byte
	copy(joinline[:], current)
	if next != nil {
		copy(joinline[len(current):], next)
	}
	for i := 0; i < len(current); {
		c := rune(joinline[i])
		length := 1
		if c == '\u000A' {
			c = _HALFWIDTH_DOWNWARDS_ARROW
		} else if c == '\u000D' {
			c = _HALFWIDTH_LEFTWARDS_ARROW
		} else if c == '\t' {
			c = _RIGHTWARDS_ARROW_TO_BAR
		} else if c < ' ' || c == '\u007F' {
			c = '.'
		} else if c >= utf8.RuneSelf {
			c, length = utf8.DecodeRune(joinline[i:])
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

func (app *Application) View() (int, error) {
	h := app.screenHeight - 1
	out := app.out
	count := 0

	if app.buffer.Len() <= 0 {
		var err error
		app.window, err = app.buffer.fetch()
		if err != nil {
			return 0, err
		}
	}
	cursor := app.window.Clone()
	bytes := cursor.Bytes()
	address := cursor.Address()
	for {
		var cursorPos int
		if address == app.rowIndex.Address() {
			cursorPos = app.colIndex
		} else {
			cursorPos = -1
		}
		var fetchErr error
		if !cursor.Next() {
			cursor, fetchErr = app.buffer.fetch()
		}
		var nextBytes []byte
		var nextAddress int
		if cursor != nil {
			nextBytes = cursor.Bytes()
			nextAddress = cursor.Address()
		}

		var buffer strings.Builder
		draw(&buffer, address, cursorPos, bytes, nextBytes)
		line := buffer.String()

		if f := cache[count]; f != line {
			io.WriteString(out, line)
			cache[count] = line
		}

		if fetchErr == io.EOF {
			return count, nil
		}
		if fetchErr != nil {
			return count, fetchErr
		}
		bytes = nextBytes
		address = nextAddress
		if count+1 >= h {
			return count, nil
		}
		count++
		io.WriteString(out, "\r\n") // "\r" is for Linux and go-tty
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

type Application struct {
	tty1         *tty.TTY
	in           io.Reader
	out          io.Writer
	screenWidth  int
	screenHeight int
	colIndex     int
	rowIndex     *Cursor
	window       *Cursor
	buffer       *Buffer
	clipBoard    *Clip
	dirty        bool
	savePath     string
	message      string
	cache        map[int]string
}

func (app *Application) dataHeight() int {
	return app.screenHeight - 1
}

func (app *Application) ChangedMark() rune {
	if app.dirty {
		return '*'
	} else {
		return ' '
	}
}

func NewApplication(in io.Reader, out io.Writer, defaultName string) (*Application, error) {
	this := &Application{}

	this.savePath = defaultName
	this.in = in
	this.out = out

	var err error
	this.tty1, err = tty.Open()
	if err != nil {
		return nil, err
	}
	this.clipBoard = NewClip()

	this.buffer = NewBuffer(this.in)
	this.rowIndex = this.buffer.Begin()
	this.window = this.buffer.Begin()

	io.WriteString(this.out, _ANSI_CURSOR_OFF)

	this.dirty = false
	this.message = ""

	return this, nil
}

func (this *Application) Close() error {
	io.WriteString(this.out, _ANSI_CURSOR_ON)

	if this.tty1 != nil {
		this.tty1.Close()
	}
	return nil
}

func mains(args []string) error {
	disable := colorable.EnableColorsStdout(nil)
	if disable != nil {
		defer disable()
	}
	out := colorable.NewColorableStdout()

	in, err := NewArgf(args)
	if err != nil {
		return err
	}
	defer in.Close()

	savePath := "output.new"
	if len(args) > 0 {
		savePath, err = filepath.Abs(args[0])
		if err != nil {
			return err
		}
	}

	app, err := NewApplication(in, out, savePath)
	if err != nil {
		return err
	}
	defer app.Close()

	var lastWidth, lastHeight int
	for {
		app.screenWidth, app.screenHeight, err = app.tty1.Size()
		if err != nil {
			return err
		}
		if lastWidth != app.screenWidth || lastHeight != app.screenHeight {
			app.cache = map[int]string{}
			lastWidth = app.screenWidth
			lastHeight = app.screenHeight
			io.WriteString(app.out, _ANSI_CURSOR_OFF)
		}
		lf, err := app.View()
		if err != nil {
			return err
		}
		if app.buffer.Len() <= 0 {
			return nil
		}
		io.WriteString(app.out, "\r\n") // \r is for Linux & go-tty
		lf++
		if app.message != "" {
			io.WriteString(app.out, _ANSI_YELLOW)
			io.WriteString(app.out, runewidth.Truncate(app.message, app.screenWidth-1, ""))
			io.WriteString(app.out, _ANSI_RESET)
			app.message = ""
		} else if 0 <= app.rowIndex.index && app.rowIndex.index < app.buffer.Len() {
			if 0 <= app.colIndex && app.colIndex < app.rowIndex.Len() {
				fmt.Fprintf(app.out, "\x1B[0;33;1m%[3]c(%08[1]X):0x%02[2]X=%-4[2]d",
					app.rowIndex.Address()+app.colIndex,
					app.rowIndex.Byte(app.colIndex),
					app.ChangedMark())

				theRune, thePosInRune, theLenOfRune := app.rowIndex.Rune(app.colIndex)
				if theRune != utf8.RuneError {
					fmt.Fprintf(app.out, "(%d/%d:U+%X)",
						thePosInRune+1,
						theLenOfRune,
						theRune)
				} else {
					io.WriteString(app.out, "(not UTF8)")
				}
				io.WriteString(app.out, "\x1B[0m")
			}
		}
		io.WriteString(app.out, ERASE_SCRN_AFTER)
		ch, err := getkey(app.tty1)
		if err != nil {
			return err
		}
		if hander, ok := jumpTable[ch]; ok {
			if err := hander(app); err != nil {
				return err
			}
		}
		if app.buffer.Len() <= 0 {
			return nil
		}
		if app.rowIndex.index >= app.buffer.Len() {
			app.rowIndex.Prev()
			app.colIndex = LINE_SIZE
		}
		if app.colIndex >= app.rowIndex.Len() {
			app.colIndex = app.rowIndex.Len() - 1
		}

		if app.rowIndex.index < app.window.index {
			app.window = app.rowIndex.Clone()
		} else if app.rowIndex.index >= app.window.index+app.screenHeight-1 {
			app.window = app.rowIndex.Clone()
			for i := app.screenHeight - 1 + 1; i > 0; i-- {
				if !app.window.Prev() {
					break
				}
			}
		}
		if lf > 0 {
			fmt.Fprintf(app.out, "\r\x1B[%dA", lf)
		} else {
			io.WriteString(app.out, "\r")
		}
	}
}

func main() {
	if err := mains(os.Args[1:]); err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
