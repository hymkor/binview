package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/mattn/go-tty"

	. "github.com/zetamatta/binview/internal/buffer"
)

func keyFuncNext(this *Application) error {
	if err := this.cursor.Skip(LINE_SIZE); err != nil {
		if err != io.EOF {
			return err
		}
	}
	return nil
}

func keyFuncBackword(this *Application) error {
	this.cursor.Prev()
	return nil
}

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

func keyFuncGoEndofLine(this *Application) error {
	n := LINE_SIZE - this.cursor.Address()%LINE_SIZE
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

func keyFuncGoEndOfFile(this *Application) error {
	this.cursor.GoEndOfFile()
	return nil
}

func keyFuncPasteAfter(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newByte := this.clipBoard.Pop()
	this.cursor.AppendByte(newByte)
	return nil
}

func keyFuncAddByte(this *Application) error {
	this.cursor.AppendByte(0)
	return nil
}

func keyFuncPasteBefore(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newByte := this.clipBoard.Pop()
	this.cursor.InsertByte(newByte)
	return nil
}

func keyFuncInsertByte(this *Application) error {
	this.cursor.InsertByte(0)
	return nil
}

func keyFuncRemoveByte(this *Application) error {
	this.dirty = true
	this.clipBoard.Push(this.cursorByte())
	switch this.cursor.DeleteByte() {
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

func writeFile(buffer *Buffer, tty1 *tty.TTY, out io.Writer, fname string) (string, error) {
	var err error

	fname, err = getline(out, "write to>", fname)
	if err != nil {
		return "", err
	}
	buffer.ReadAll()
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
	buffer.Each(func(block []byte) {
		fd.Write(block)
	})
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
	bytes, err := getline(this.out, "replace>",
		fmt.Sprintf("0x%02X", this.cursorByte()))
	if err != nil {
		this.message = err.Error()
		return nil
	}
	if n, err := strconv.ParseUint(bytes, 0, 8); err == nil {
		this.setCursorByte(byte(n))
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
	prevousAddress := app.cursorAddress()
	if address > prevousAddress {
		app.cursor.Skip(address - prevousAddress)
	} else if address < prevousAddress {
		app.cursor.Rewind(prevousAddress - address)
	}
	return nil
}

func keyFuncGoTo(app *Application) error {
	addressStr, err := getline(app.out, "Goto Offset>", "0x")
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
	"$":         keyFuncGoEndofLine,
	_KEY_CTRL_E: keyFuncGoEndofLine,
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
