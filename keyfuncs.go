package main

import (
	"io"
	"os"
	"path/filepath"
)

func deleteOne(reader io.Reader, slices [][]byte, rowIndex, colIndex int) [][]byte {
	if colIndex < LINE_SIZE {
		csrline := slices[rowIndex]
		copy(csrline[colIndex:], csrline[colIndex+1:])
	}
	for i := rowIndex; i+1 < len(slices); i++ {
		slices[i][len(slices[i])-1] = slices[i+1][0]
		copy(slices[i+1][:], slices[i+1][1:])
	}
	last := slices[len(slices)-1]
	if reader != nil {
		reader.Read(last[len(last)-1:])
	} else {
		if len(last) > 1 {
			slices[len(slices)-1] = last[:len(last)-1]
		} else {
			slices = slices[:len(slices)-1]
			if len(slices) <= 0 {
				return nil
			}
			if rowIndex >= len(slices) {
				rowIndex--
				colIndex = len(slices[len(slices)-1]) - 1
			}
		}
	}
	return slices
}

func write(reader io.Reader, out io.Writer, slices [][]byte, args []string) error {
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
	if reader != nil {
		slices = readAll(reader, slices)
	}
	fd, err := os.OpenFile(fname, os.O_EXCL|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	for _, s := range slices {
		fd.Write(s)
	}
	return fd.Close()
}
