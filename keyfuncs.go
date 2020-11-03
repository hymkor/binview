package main

import (
	"io"
	"os"
	"path/filepath"
)

func lastByte(b []byte) byte {
	return b[len(b)-1]
}

func insertOne(b *Buffer, rowIndex, colIndex int) {
	b.ReadAll()
	carry := lastByte(b.Slices[rowIndex])
	copy(b.Slices[rowIndex][colIndex+1:], b.Slices[rowIndex][colIndex:])
	for i := rowIndex + 1; i < b.Count(); i++ {
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
	csrline[len(csrline)-1] = carry

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

func write(buffer *Buffer, out io.Writer, args []string) error {
	fname := "output.new"
	var err error
	if len(args) >= 1 {
		fname, err = filepath.Abs(args[0])
		if err != nil {
			return err
		}
		fname += ".new"
	}
	fname, err = getline(out, "write to>", fname)
	if err != nil {
		return err
	}
	buffer.ReadAll()
	fd, err := os.OpenFile(fname, os.O_EXCL|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	for _, s := range buffer.Slices {
		fd.Write(s)
	}
	return fd.Close()
}
