package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/simplehistory"

	"github.com/zetamatta/binview/internal/encoding"
	"github.com/zetamatta/binview/internal/large"
	"github.com/zetamatta/binview/internal/nonblock"
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
	_KEY_ALT_A  = "\x1Ba"
	_KEY_ALT_U  = "\x1Bu"
	_KEY_ALT_L  = "\x1Bl"
	_KEY_ALT_B  = "\x1Bb"
)

// keyFuncNext moves the cursor to the the next 16-bytes block.
func keyFuncNext(this *Application) error {
	if err := this.cursor.Skip(LINE_SIZE); err != nil {
		if err != io.EOF {
			return err
		}
	}
	return nil
}

// keyFuncBackword move the cursor to the previous byte.
func keyFuncBackword(this *Application) error {
	this.cursor.Prev()
	return nil
}

// keyFuncPrevious moves the cursor the the previous 16-bytes block.
func keyFuncPrevious(this *Application) error {
	this.cursor.Rewind(LINE_SIZE)
	return nil
}

func keyFuncQuit(this *Application) error {
	if yesNo(this.tty1, this.out, "Quit Sure ? [y/n]") {
		io.WriteString(this.out, "\n")
		return io.EOF
	}
	return nil
}

// keyFuncForward moves the cursor to the next one byte.
func keyFuncForward(this *Application) error {
	this.cursor.Next()
	return nil
}

// keyFuncGoBeginOfLine move the cursor the the top of the 16bytes-block.
func keyFuncGoBeginOfLine(this *Application) error {
	n := this.cursor.Address() % LINE_SIZE
	if n > 0 {
		this.cursor.Rewind(n)
	}
	return nil
}

// keyFuncGoEndOfLine move the cursor to the end of the current 16 byte block.
func keyFuncGoEndOfLine(this *Application) error {
	n := LINE_SIZE - this.cursor.Address()%LINE_SIZE - 1
	if n > 0 {
		this.cursor.Skip(n)
	}
	return nil
}

func keyFuncGoBeginOfFile(this *Application) error {
	this.cursor = large.NewPointer(this.buffer)
	this.window = large.NewPointer(this.buffer)
	return nil
}

// keyFuncGoEndOfFile moves the cursor to the end of the file.
func keyFuncGoEndOfFile(this *Application) error {
	this.cursor.GoEndOfFile()
	return nil
}

// keyFuncPasteAfter inserts the top byte of clipboard after the cursor.
func keyFuncPasteAfter(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newByte := this.clipBoard.Pop()
	this.cursor.Append(newByte)
	this.dirty = true
	return nil
}

// keyFuncAddByte inserts the zero after the cursor.
func keyFuncAddByte(this *Application) error {
	this.cursor.Append(0)
	this.dirty = true
	return nil
}

// keyFuncPasteBefore inserts the top of the clipboard at the cursor.
func keyFuncPasteBefore(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newByte := this.clipBoard.Pop()
	this.cursor.Insert(newByte)
	this.dirty = true
	return nil
}

// keyFuncInsertByte inserts the zero where cursor exists.
func keyFuncInsertByte(this *Application) error {
	this.cursor.Insert(0)
	this.dirty = true
	return nil
}

// keyFuncRemoveByte removes the byte where cursor exists.
func keyFuncRemoveByte(this *Application) error {
	this.dirty = true
	this.clipBoard.Push(this.cursor.Value())
	switch this.cursor.Remove() {
	case large.RemoveAll:
		return io.EOF
	case large.RemoveRefresh:
		this.window = this.cursor
		return nil
	default:
		return nil
	}
}

var overWritten = map[string]struct{}{}

func getlineOr(out io.Writer, prompt string, defaultString string, history readline.IHistory, f func() bool) (string, error) {
	worker := nonblock.New(func() (string, error) {
		return getline(out, prompt, defaultString, history)
	})
	result, err := worker.GetOr(f)
	worker.Close()
	return result, err
}

var fnameHistory = simplehistory.New()

func writeFile(buffer *large.Buffer, tty1 Tty, out io.Writer, fname string) (string, error) {
	var err error
	fname, err = getlineOr(out, "write to>", fname, fnameHistory, func() bool { return buffer.Fetch() == nil })
	if err != nil {
		return "", err
	}
	fd, err := os.OpenFile(fname, os.O_EXCL|os.O_CREATE, 0666)
	if os.IsExist(err) {
		if _, ok := overWritten[fname]; ok {
			os.Remove(fname)
		} else {
			if !yesNo(tty1, out, "Overwrite as \""+fname+"\" [y/n] ?") {
				return "", err
			}
			backupName := fname + "~"
			os.Remove(backupName)
			os.Rename(fname, backupName)
			overWritten[fname] = struct{}{}
		}
		fd, err = os.OpenFile(fname, os.O_EXCL|os.O_CREATE, 0666)
	}
	if err != nil {
		return "", err
	}
	buffer.WriteTo(fd)
	fnameHistory.Add(fname)
	return fname, fd.Close()
}

func keyFuncWriteFile(this *Application) error {
	newfname, err := writeFile(this.buffer, this.tty1, this.out, this.savePath)
	if err != nil {
		this.message = err.Error()
	} else {
		this.dirty = false
		this.savePath = newfname
	}
	return nil
}

var byteHistory = simplehistory.New()

func keyFuncReplaceByte(this *Application) error {
	bytes, err := getlineOr(this.out, "replace>",
		fmt.Sprintf("0x%02X", this.cursor.Value()),
		byteHistory,
		func() bool { return this.buffer.Fetch() == nil })
	if err != nil {
		this.message = err.Error()
		return nil
	}
	if n, err := strconv.ParseUint(bytes, 0, 8); err == nil {
		this.cursor.SetValue(byte(n))
		this.dirty = true
		byteHistory.Add(bytes)
	} else {
		this.message = err.Error()
	}
	return nil
}

func keyFuncRepaint(this *Application) error {
	this.cache = map[int]string{}
	return nil
}

func gotoAddress(app *Application, address int64) error {
	prevousAddress := app.cursor.Address()
	if address > prevousAddress {
		app.cursor.Skip(address - prevousAddress)
	} else if address < prevousAddress {
		app.cursor.Rewind(prevousAddress - address)
	}
	return nil
}

var addressHistory = simplehistory.New()

func keyFuncGoTo(app *Application) error {
	addressStr, err := getlineOr(app.out, "Goto Offset>", "0x", addressHistory, func() bool {
		return app.buffer.Fetch() == nil
	})
	if err != nil {
		app.message = err.Error()
		return nil
	}
	address, err := strconv.ParseInt(addressStr, 0, 64)
	if err != nil {
		app.message = err.Error()
		return nil
	}
	addressHistory.Add(addressStr)
	return gotoAddress(app, address)
}

func keyFuncDbcsMode(app *Application) error {
	app.encoding = encoding.DBCSEncoding{}
	return nil
}

func keyFuncUtf8Mode(app *Application) error {
	app.encoding = encoding.UTF8Encoding{}
	return nil
}

func keyFuncUtf16LeMode(app *Application) error {
	app.encoding = encoding.UTF16LE()
	return nil
}

func keyFuncUtf16BeMode(app *Application) error {
	app.encoding = encoding.UTF16BE()
	return nil
}

var (
	rxUnicodeCodePoint = regexp.MustCompile(`^\s*[uU]\+([0-9A-Fa-f]+)`)
	rxByte             = regexp.MustCompile(`^\s*0x([0-9A-Fa-f]+)`)
	rxDigit            = regexp.MustCompile(`^\s*([0-9]+)`)
	rxString           = regexp.MustCompile(`^\s*[uU]?"([^"]+)"`)
)

func keyFuncInsertData(app *Application) error {
	data, err := readData(app, "insert>")
	if err != nil {
		app.message = err.Error()
	}
	insertArea := app.cursor.MakeSpace(len(data))
	copy(insertArea, data)
	return nil
}

func keyFuncAppendData(app *Application) error {
	data, err := readData(app, "append>")
	if err != nil {
		app.message = err.Error()
	}
	insertArea := app.cursor.MakeSpaceAfter(len(data))
	copy(insertArea, data)
	return nil
}

var dataHistory = simplehistory.New()

func readData(app *Application, prompt string) ([]byte, error) {
	str, err := getlineOr(app.out, prompt, "0x00", dataHistory, func() bool { return app.buffer.Fetch() == nil })
	if err != nil {
		return nil, err
	}
	dataHistory.Add(str)
	var buffer bytes.Buffer
	for len(str) > 0 {
		if m := rxUnicodeCodePoint.FindStringSubmatch(str); m != nil {
			str = str[len(m[0]):]
			theRune, err := strconv.ParseUint(m[1], 16, 32)
			if err != nil {
				return nil, err
			}
			if bin, err := app.encoding.EncodeFromString(string(theRune)); err == nil {
				buffer.Write(bin)
			}
		} else if m := rxByte.FindStringSubmatch(str); m != nil {
			str = str[len(m[0]):]
			theByte, err := strconv.ParseUint(m[1], 16, 16)
			if err != nil {
				return nil, err
			}
			buffer.WriteByte(byte(theByte))
		} else if m := rxDigit.FindStringSubmatch(str); m != nil {
			str = str[len(m[0]):]
			value, err := strconv.ParseUint(m[1], 10, 16)
			if err != nil {
				return nil, err
			}
			buffer.WriteByte(byte(value))
		} else if m := rxString.FindStringSubmatch(str); m != nil {
			str = str[len(m[0]):]
			if bin, err := app.encoding.EncodeFromString(m[1]); err == nil {
				buffer.Write(bin)
			}
		} else {
			app.message = fmt.Sprintf("`%s` are ignored", str)
			break
		}
	}
	return buffer.Bytes(), nil
}

var jumpTable = map[string]func(this *Application) error{
	"i":         keyFuncInsertData,
	"a":         keyFuncAppendData,
	_KEY_ALT_A:  keyFuncDbcsMode,
	_KEY_ALT_U:  keyFuncUtf8Mode,
	_KEY_ALT_L:  keyFuncUtf16LeMode,
	_KEY_ALT_B:  keyFuncUtf16BeMode,
	"&":         keyFuncGoTo,
	"q":         keyFuncQuit,
	_KEY_ESC:    keyFuncQuit,
	"j":         keyFuncNext,
	_KEY_DOWN:   keyFuncNext,
	_KEY_CTRL_N: keyFuncNext,
	"h":         keyFuncBackword,
	"\b":        keyFuncBackword,
	_KEY_LEFT:   keyFuncBackword,
	_KEY_CTRL_B: keyFuncBackword,
	"k":         keyFuncPrevious,
	_KEY_UP:     keyFuncPrevious,
	_KEY_CTRL_P: keyFuncPrevious,
	"l":         keyFuncForward,
	" ":         keyFuncForward,
	_KEY_RIGHT:  keyFuncForward,
	_KEY_CTRL_F: keyFuncForward,
	"0":         keyFuncGoBeginOfLine,
	"^":         keyFuncGoBeginOfLine,
	_KEY_CTRL_A: keyFuncGoBeginOfLine,
	"$":         keyFuncGoEndOfLine,
	_KEY_CTRL_E: keyFuncGoEndOfLine,
	"<":         keyFuncGoBeginOfFile,
	">":         keyFuncGoEndOfFile,
	"G":         keyFuncGoEndOfFile,
	"p":         keyFuncPasteAfter,
	"P":         keyFuncPasteBefore,
	"x":         keyFuncRemoveByte,
	_KEY_DEL:    keyFuncRemoveByte,
	"w":         keyFuncWriteFile,
	"r":         keyFuncReplaceByte,
	_KEY_CTRL_L: keyFuncRepaint,
	// "a":keyFuncAddByte,
	// "i":keyFuncInsertByte,
}
