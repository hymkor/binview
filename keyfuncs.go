package main

import (
	"io"
	"os"
	"path/filepath"
)

func deleteOne(b *Buffer, rowIndex, colIndex int) {
	if colIndex < LINE_SIZE {
		csrline := b.Slices[rowIndex]
		copy(csrline[colIndex:], csrline[colIndex+1:])
	}
	for i := rowIndex; i+1 < b.Count(); i++ {
		b.SetByte(i, b.Count()-1, b.Byte(i+1, 0))
		copy(b.Slices[i+1][:], b.Slices[i+1][1:])
	}
	last := b.Slices[len(b.Slices)-1]
	if b.Reader != nil {
		b.Read(last[len(last)-1:])
	} else {
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
