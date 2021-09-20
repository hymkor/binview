package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	. "github.com/zetamatta/binview/internal/buffer"
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
	this.cursor = NewPointer(this.buffer)
	this.window = NewPointer(this.buffer)
	return nil
}

// keyFuncGoEndOfFile moves the cursor to the end of the file.
func keyFuncGoEndOfFile(this *Application) error {
	this.cursor.GoEndOfFile()
	return nil
}

func keyFuncPasteAfter(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newByte := this.clipBoard.Pop()
	this.cursor.Append(newByte)
	return nil
}

func keyFuncAddByte(this *Application) error {
	this.cursor.Append(0)
	return nil
}

func keyFuncPasteBefore(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newByte := this.clipBoard.Pop()
	this.cursor.Insert(newByte)
	return nil
}

// keyFuncInsertByte inserts the zero where cursor exists.
func keyFuncInsertByte(this *Application) error {
	this.cursor.Insert(0)
	return nil
}

// keyFuncRemoveByte removes the byte where cursor exists.
func keyFuncRemoveByte(this *Application) error {
	this.dirty = true
	this.clipBoard.Push(this.cursor.Value())
	switch this.cursor.Delete() {
	case DeleteAll:
		return io.EOF
	case DeleteRefresh:
		this.window = this.cursor
		return nil
	default:
		return nil
	}
}

var overWritten = map[string]struct{}{}

func getlineOr(out io.Writer, prompt string, defaultString string, f func() bool) (string, error) {
	worker := NewNonBlock(func() (string, error) {
		return getline(out, prompt, defaultString)
	})
	result, err := worker.GetOr(f)
	worker.Close()
	return result, err
}

func writeFile(buffer *Buffer, tty1 Tty, out io.Writer, fname string) (string, error) {
	var err error
	fname, err = getlineOr(out, "write to>", fname, func() bool { return buffer.Fetch() == nil })
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

func keyFuncReplaceByte(this *Application) error {
	bytes, err := getlineOr(this.out, "replace>",
		fmt.Sprintf("0x%02X", this.cursor.Value()),
		func() bool { return this.buffer.Fetch() == nil })
	if err != nil {
		this.message = err.Error()
		return nil
	}
	if n, err := strconv.ParseUint(bytes, 0, 8); err == nil {
		this.cursor.SetValue(byte(n))
		this.dirty = true
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

func keyFuncGoTo(app *Application) error {
	addressStr, err := getlineOr(app.out, "Goto Offset>", "0x", func() bool {
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
	return gotoAddress(app, address)
}

var jumpTable = map[string]func(this *Application) error{
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
	"a":         keyFuncAddByte,
	"P":         keyFuncPasteBefore,
	"i":         keyFuncInsertByte,
	"x":         keyFuncRemoveByte,
	_KEY_DEL:    keyFuncRemoveByte,
	"w":         keyFuncWriteFile,
	"r":         keyFuncReplaceByte,
	_KEY_CTRL_L: keyFuncRepaint,
}
