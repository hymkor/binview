package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/mattn/go-tty"
)

func keyFuncNext(this *Application) error {
	if this.rowIndex < this.buffer.Count()-1 {
		this.rowIndex++
	} else if _, _, err := this.buffer.Fetch(); err == nil {
		this.rowIndex++
	} else if err != io.EOF {
		return err
	}
	return nil
}

func keyFuncBackword(this *Application) error {
	if this.colIndex > 0 {
		this.colIndex--
	} else if this.rowIndex > 0 {
		this.rowIndex--
		this.colIndex = LINE_SIZE - 1
	}
	return nil
}

func keyFuncPrevious(this *Application) error {
	if this.rowIndex > 0 {
		this.rowIndex--
	}
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
	if this.colIndex < LINE_SIZE-1 {
		this.colIndex++
	} else if this.rowIndex < this.buffer.Count()-1 {
		this.rowIndex++
		this.colIndex = 0
	} else if _, _, err := this.buffer.Fetch(); err == nil {
		this.rowIndex++
		this.colIndex = 0
	} else if err != io.EOF {
		return err
	}
	return nil
}

func keyFuncGoBeginOfLine(this *Application) error {
	this.colIndex = 0
	return nil
}

func keyFuncGoEndofLine(this *Application) error {
	this.colIndex = this.buffer.Line[this.rowIndex].Len() - 1
	return nil
}

func keyFuncGoBeginOfFile(this *Application) error {
	this.rowIndex = 0
	this.colIndex = 0
	return nil
}

func keyFuncGoEndOfFile(this *Application) error {
	this.buffer.ReadAll()
	this.rowIndex = this.buffer.Count() - 1
	this.colIndex = this.buffer.Line[this.rowIndex].Len() - 1
	this.buffer.Reader = nil
	return nil
}

func keyFuncPasteAfter(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newByte := this.clipBoard.Pop()
	return _addbyte(this, newByte)
}

func _addbyte(this *Application, newByte byte) error {
	if this.colIndex+1 < len(this.buffer.Line[this.rowIndex]) {
		this.colIndex++
	} else {
		this.colIndex = 0
		this.rowIndex++
	}
	this.buffer.InsertAt(this.rowIndex, this.colIndex, newByte)
	this.dirty = true
	return nil
}

func keyFuncAddByte(this *Application) error {
	return _addbyte(this, 0)
}

func keyFuncPasteBefore(this *Application) error {
	if this.clipBoard.Len() <= 0 {
		return nil
	}
	newByte := this.clipBoard.Pop()
	return _insertByte(this, newByte)
}

func _insertByte(this *Application, value byte) error {
	this.buffer.InsertAt(this.rowIndex, this.colIndex, value)
	this.dirty = true
	return nil
}

func keyFuncInsertByte(this *Application) error {
	return _insertByte(this, 0)
}

func keyFuncRemoveByte(this *Application) error {
	this.clipBoard.Push(this.buffer.Line[this.rowIndex][this.colIndex])
	this.buffer.deleteOne(this.rowIndex, this.colIndex)
	this.dirty = true
	return nil
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
	for _, s := range buffer.Line {
		fd.Write(s)
	}
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
		fmt.Sprintf("0x%02X", this.buffer.Byte(this.rowIndex, this.colIndex)))
	if err != nil {
		this.message = err.Error()
		return nil
	}
	if n, err := strconv.ParseUint(bytes, 0, 8); err == nil {
		this.buffer.SetByte(this.rowIndex, this.colIndex, byte(n))
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

var jumpTable = map[string]func(this *Application) error{
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
