package buffer

type Line []byte

func (line Line) LastByte() byte {
	return line[len(line)-1]
}

func (line Line) SetLastByte(value byte) {
	line[len(line)-1] = value
}

func (line Line) Len() int {
	return len(line)
}

func (line Line) RemoveAt(pos int, appendByte byte) byte {
	deleteByte := line[pos]
	copy(line[pos:], line[pos+1:])
	line.SetLastByte(appendByte)
	return deleteByte
}

func (line Line) InsertAt(pos int, value byte) byte {
	deleteByte := line.LastByte()
	copy(line[pos+1:], line[pos:])
	line[pos] = value
	return deleteByte
}
