package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"
	"github.com/mattn/go-tty"

	"github.com/nyaosorg/go-readline-ny"

	"github.com/zetamatta/binview/internal/argf"
	"github.com/zetamatta/binview/internal/large"
	"github.com/zetamatta/binview/internal/nonblock"
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

var version string = "snapshot"

// See. en.wikipedia.org/wiki/Unicode_control_characters#Control_pictures

func makeHexPart(pointer *large.Pointer, cursorAddress int64, out *strings.Builder) bool {
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

func makeAsciiPart(pointer *large.Pointer, cursorAddress int64, out *strings.Builder) bool {
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
			savePointer := pointer.Clone()
			for j := 1; j < length && pointer.Next() == nil; j++ {
				runeBuffer[j] = pointer.Value()
				readCount++
			}
			c, length = utf8.DecodeRune(runeBuffer[:readCount])
			if c == utf8.RuneError {
				c = '.'
				length = 1
				pointer = savePointer
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

func makeLineImage(pointer *large.Pointer, cursorAddress int64) (string, bool) {
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

func (app *Application) View() (int, error) {
	h := app.screenHeight - 1
	out := app.out
	count := 0

	cursor := app.window.Clone()
	cursorAddress := app.cursor.Address()
	for {
		line, cont := makeLineImage(cursor, cursorAddress)

		if f := app.cache[count]; f != line {
			io.WriteString(out, line)
			app.cache[count] = line
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
	cursor       *large.Pointer
	window       *large.Pointer
	buffer       *large.Buffer
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
	this := &Application{
		savePath:  defaultName,
		in:        in,
		out:       out,
		buffer:    large.NewBuffer(in),
		clipBoard: NewClip(),
	}
	this.window = large.NewPointer(this.buffer)
	if this.window == nil {
		return nil, io.EOF
	}
	this.cursor = large.NewPointer(this.buffer)
	if this.cursor == nil {
		return nil, io.EOF
	}
	var err error
	this.tty1, err = tty.Open()
	if err != nil {
		return nil, err
	}
	io.WriteString(out, _ANSI_CURSOR_OFF)
	return this, nil
}

func (this *Application) Close() error {
	io.WriteString(this.out, _ANSI_CURSOR_ON)
	io.WriteString(this.out, _ANSI_RESET)

	if this.tty1 != nil {
		this.tty1.Close()
	}
	return nil
}

func readRune(cursor *large.Pointer) (rune, int, int) {
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

func (app *Application) printDefaultStatusBar() {
	io.WriteString(app.out, _ANSI_YELLOW)
	fmt.Fprintf(app.out,
		"%[1]c(%[2]d=0x%[2]X/%[3]d=0x%[3]X):%4[4]d=0x%02[4]X",
		app.ChangedMark(),
		app.cursor.Address(),
		app.buffer.Len(),
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
	io.WriteString(app.out, ERASE_SCRN_AFTER)
	io.WriteString(app.out, _ANSI_RESET)
}

func (app *Application) shiftWindowToSeeCursorLine() {
	if app.cursor.Address() < app.window.Address() {
		app.window = app.cursor.Clone()
		if n := app.window.Address() % LINE_SIZE; n > 0 {
			app.window.Rewind(n)
		}
	} else if app.cursor.Address() >= app.window.Address()+LINE_SIZE*int64(app.dataHeight()) {
		app.window = app.cursor.Clone()
		app.window.Rewind(
			app.window.Address()%LINE_SIZE +
				int64(LINE_SIZE*(app.dataHeight()-1)))
	}
}

func mains(args []string) error {
	disable := colorable.EnableColorsStdout(nil)
	if disable != nil {
		defer disable()
	}
	out := colorable.NewColorableStdout()

	fmt.Fprintf(out, "binview %s-%s-%s by %s\n",
		version, runtime.GOOS, runtime.GOARCH, runtime.Version())

	in, err := argf.New(args)
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

	keyWorker := nonblock.New(func() (string, error) { return readline.GetKey(app.tty1) })
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
		if app.buffer.Len() <= 0 {
			return nil
		}
		io.WriteString(app.out, "\r\n") // \r is for Linux & go-tty
		lf++
		if app.message != "" {
			io.WriteString(app.out, _ANSI_YELLOW)
			io.WriteString(app.out, runewidth.Truncate(app.message, app.screenWidth-1, ""))
			io.WriteString(app.out, ERASE_SCRN_AFTER)
			io.WriteString(app.out, _ANSI_RESET)
		} else {
			app.printDefaultStatusBar()
		}

		const interval = 10
		displayUpdateTime := time.Now().Add(time.Second / interval)

		ch, err := keyWorker.GetOr(func() bool {
			err := app.buffer.Fetch()
			if err != nil && err != io.EOF {
				return false
			}
			if app.message != "" {
				return err == nil
			}
			if err == io.EOF || time.Now().After(displayUpdateTime) {
				app.out.Write([]byte{'\r'})
				app.printDefaultStatusBar()
				displayUpdateTime = time.Now().Add(time.Second / interval)
			}
			return err == nil
		})
		if err != nil {
			return err
		}
		app.message = ""
		if hander, ok := jumpTable[ch]; ok {
			if err := hander(app); err != nil {
				return err
			}
		}
		if app.buffer.Len() <= 0 {
			return nil
		}

		app.shiftWindowToSeeCursorLine()

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
