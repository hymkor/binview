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

	. "github.com/zetamatta/binview/internal/buffer"
)

const LINE_SIZE = 16

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

// See. en.wikipedia.org/wiki/Unicode_control_characters#Control_pictures

func makeHexPart(pointer *Pointer, cursorAddress int64, out *strings.Builder) bool {
	fmt.Fprintf(out, "%s%08X%s ", CELL2_COLOR_ON, pointer.Address(), CELL2_COLOR_OFF)
	var fieldSeperator string
	for i := 0; i < LINE_SIZE; i++ {
		var on, off string
		if pointer.Address() == cursorAddress {
			on = CURSOR_COLOR_ON
			off = CURSOR_COLOR_OFF
		} else if ((i >> 2) & 1) == 0 {
			on = CELL1_COLOR_ON
			off = CELL1_COLOR_OFF
		} else {
			on = CELL2_COLOR_ON
			off = CELL2_COLOR_OFF
		}
		fmt.Fprintf(out, "%s%s%02X%s", fieldSeperator, on, pointer.Value(), off)
		if err := pointer.Next(); err != nil {
			for ; i < LINE_SIZE-1; i++ {
				out.WriteString("   ")
			}
			return false
		}
		fieldSeperator = " "
	}
	return true
}

func runeCount(b byte) int {
	if 0xF0 <= b && b <= 0xF4 {
		return 4
	} else if 0xE0 <= b && b <= 0xEF {
		return 3
	} else if 0xC2 <= b && b <= 0xDF {
		return 2
	} else {
		return 1
	}
}

func makeAsciiPart(pointer *Pointer, cursorAddress int64, out *strings.Builder) bool {
	for i := 0; i < LINE_SIZE; {
		var c rune
		startAddress := pointer.Address()
		b := pointer.Value()
		length := 1
		if b == '\u000A' {
			c = _HALFWIDTH_DOWNWARDS_ARROW
		} else if b == '\u000D' {
			c = _HALFWIDTH_LEFTWARDS_ARROW
		} else if b == '\t' {
			c = _RIGHTWARDS_ARROW_TO_BAR
		} else if b < ' ' || b == '\u007F' {
			c = '.'
		} else if b >= utf8.RuneSelf {
			var runeBuffer [utf8.UTFMax]byte
			length = runeCount(b)
			runeBuffer[0] = b
			readCount := 1
			for j := 1; j < length && pointer.Next() == nil; j++ {
				runeBuffer[j] = pointer.Value()
				readCount++
			}
			c, length = utf8.DecodeRune(runeBuffer[:readCount])
			if c == utf8.RuneError {
				c = '.'
			}
		} else {
			c = rune(b)
		}
		if startAddress <= cursorAddress && cursorAddress <= pointer.Address() {
			out.WriteString(CURSOR_COLOR_ON)
			out.WriteRune(c)
			out.WriteString(CURSOR_COLOR_OFF)
		} else {
			out.WriteString(CELL1_COLOR_ON)
			out.WriteRune(c)
			out.WriteString(CELL1_COLOR_OFF)
		}
		if length == 3 {
			out.WriteByte(' ')
		} else if length == 4 {
			out.WriteString("  ")
		}
		i += length
		if pointer.Next() != nil {
			return false
		}
	}
	return true
}

func makeLineImage(pointer *Pointer, cursorAddress int64) (string, bool) {
	var out strings.Builder
	off := ""
	if p := pointer.Address(); p <= cursorAddress && cursorAddress < p+LINE_SIZE {
		out.WriteString(_ANSI_UNDERLINE_ON)
		off = _ANSI_UNDERLINE_OFF
	}

	asciiPointer := *pointer
	hasNextLine := makeHexPart(pointer, cursorAddress, &out)
	out.WriteByte(' ')
	makeAsciiPart(&asciiPointer, cursorAddress, &out)

	out.WriteString(ERASE_LINE)
	out.WriteString(off)
	return out.String(), hasNextLine
}

var cache = map[int]string{}

func (app *Application) windowPointer() (*Pointer, error) {
	return app.window, nil
}

func (app *Application) cursorPointer() (*Pointer, error) {
	return app.cursor, nil
}

func (app *Application) cursorAddress() int64 {
	return app.cursor.Address()
}

func (app *Application) cursorByte() byte {
	return app.cursor.Value()
}

func (app *Application) setCursorByte(value byte) {
	app.cursor.SetValue(value)
}

func (app *Application) View() (int, error) {
	h := app.screenHeight - 1
	out := app.out
	count := 0

	cursor, err := app.windowPointer()
	if err != nil {
		return 0, err
	}
	cursor = cursor.Clone()
	cursorAddress := app.cursorAddress()
	for {
		line, cont := makeLineImage(cursor, cursorAddress)
		if f := cache[count]; f != line {
			io.WriteString(out, line)
			cache[count] = line
		}
		if !cont || count+1 >= h {
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

type Application struct {
	tty1         *tty.TTY
	in           io.Reader
	out          io.Writer
	screenWidth  int
	screenHeight int
	cursor       *Pointer
	window       *Pointer
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
	this.window = NewPointer(this.buffer)
	this.cursor = NewPointer(this.buffer)

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

func readRune(cursor *Pointer) (rune, int, int) {
	cursor = cursor.Clone()
	currentPosInRune := 0
	for !utf8.RuneStart(cursor.Value()) && cursor.Prev() == nil {
		currentPosInRune++
	}
	bytes := make([]byte, 0, utf8.UTFMax)
	count := runeCount(cursor.Value())
	for i := 0; i < count; i++ {
		bytes = append(bytes, cursor.Value())
		if cursor.Next() != nil {
			break
		}
	}
	theRune, theLen := utf8.DecodeRune(bytes)
	if currentPosInRune >= theLen {
		return utf8.RuneError, 0, 1
	}
	return theRune, currentPosInRune, theLen
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

	keyWorker := NewNonBlock(func() (string, error) { return getkey(app.tty1) })
	defer keyWorker.Close()

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
		if app.buffer.AllBytes() <= 0 {
			return nil
		}
		io.WriteString(app.out, "\r\n") // \r is for Linux & go-tty
		lf++
		if app.message != "" {
			io.WriteString(app.out, _ANSI_YELLOW)
			io.WriteString(app.out, runewidth.Truncate(app.message, app.screenWidth-1, ""))
			io.WriteString(app.out, _ANSI_RESET)
			app.message = ""
		} else {
			fmt.Fprintf(app.out, "\x1B[0;33;1m%c(%X/%X):0x%02X=%-4[4]d",
				app.ChangedMark(),
				app.cursor.Address(),
				app.buffer.AllBytes(),
				app.cursor.Value())

			theRune, thePosInRune, theLenOfRune := readRune(app.cursor)
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
		io.WriteString(app.out, ERASE_SCRN_AFTER)
		ch, err := keyWorker.GetOr(func() { app.buffer.Fetch() })
		if err != nil {
			return err
		}
		if hander, ok := jumpTable[ch]; ok {
			if err := hander(app); err != nil {
				return err
			}
		}
		if app.buffer.AllBytes() <= 0 {
			return nil
		}

		if app.cursor.Address() < app.window.Address() {
			app.window = app.cursor.Clone()
			if n := app.window.Address() % LINE_SIZE; n > 0 {
				app.window.Rewind(n)
			}
		} else if app.cursor.Address() >= app.window.Address()+LINE_SIZE*int64(app.dataHeight()) {
			app.window = app.cursor.Clone()
			if n := app.window.Address() % LINE_SIZE; n > 0 {
				app.window.Rewind(n)
			}
			for i := app.dataHeight() - 1; i > 0; i-- {
				app.window.Rewind(LINE_SIZE)
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
