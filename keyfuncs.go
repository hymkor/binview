package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/mattn/go-tty"
)

func unshiftLines(b *Buffer, rowIndex int, carry byte) {
	for i := rowIndex; i < b.Count(); i++ {
		carry = b.Unshift(i, carry)
	}
	last := b.Slices[b.Count()-1]
	if len(last) < LINE_SIZE {
		last = append(last, carry)
		b.Slices[b.Count()-1] = last
	} else {
		b.Slices = append(b.Slices, []byte{carry})
	}
}

func insertOne(b *Buffer, rowIndex, colIndex int) {
	b.ReadAll()
	carry := lastByte(b.Slices[rowIndex])
	copy(b.Slices[rowIndex][colIndex+1:], b.Slices[rowIndex][colIndex:])

	unshiftLines(b, rowIndex+1, carry)
}

func appendOne(b *Buffer, rowIndex, colIndex int) {
	b.ReadAll()
	if colIndex+1 < len(b.Slices[rowIndex]) {
		// colIndex <= 14
		insertOne(b, rowIndex, colIndex+1)
		return
	}
	// colIndex == 15 and insert at colindex == 16
	unshiftLines(b, rowIndex+1, 0)
}

func deleteOne(b *Buffer, rowIndex, colIndex int) {
	b.ReadAll()
	carry := byte(0)
	for i := b.Count() - 1; i > rowIndex; i-- {
		carry = b.Shift(i, carry)
	}
	csrline := b.Slices[rowIndex]
	if colIndex < LINE_SIZE {
		copy(csrline[colIndex:], csrline[colIndex+1:])
	}
	setLastByte(csrline, carry)

	last := b.Slices[len(b.Slices)-1]
	if len(last) > 1 {
		b.Slices[len(b.Slices)-1] = last[:len(last)-1]
	} else {
		b.DropLastLine()
		if b.Count() <= 0 {
			return
		}
		if rowIndex >= b.Count() {
			rowIndex--
			colIndex = len(b.LastLine()) - 1
		}
	}
}

var overWritten = map[string]struct{}{}

func write(buffer *Buffer, tty1 *tty.TTY, out io.Writer, args []string) error {
	fname := "output.new"
	var err error
	if len(args) >= 1 {
		fname, err = filepath.Abs(args[0])
		if err != nil {
			return err
		}
	}
	fname, err = getline(out, "write to>", fname)
	if err != nil {
		return err
	}
	buffer.ReadAll()
	fd, err := os.OpenFile(fname, os.O_EXCL|os.O_CREATE, 0666)
	if os.IsExist(err) {
		if _, ok := overWritten[fname]; ok {
			os.Remove(fname)
		} else {
			if !yesNo(tty1, out, "Overwrite as \""+fname+"\" [y/n] ?") {
				return err
			}
			backupName := fname + "~"
			os.Remove(backupName)
			os.Rename(fname, backupName)
			overWritten[fname] = struct{}{}
		}
		fd, err = os.OpenFile(fname, os.O_EXCL|os.O_CREATE, 0666)
	}
	if err != nil {
		return err
	}
	for _, s := range buffer.Slices {
		fd.Write(s)
	}
	return fd.Close()
}
