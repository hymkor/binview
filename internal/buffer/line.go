package buffer

const LINE_SIZE = 16

type _Line []byte

var allocBuffer []byte

func newLine() _Line {
	// return _Line(make([]byte, LINE_SIZE))
	if allocBuffer == nil || len(allocBuffer) < LINE_SIZE {
		allocBuffer = make([]byte, LINE_SIZE*1024)
	}
	border := len(allocBuffer) - LINE_SIZE
	newLine1 := allocBuffer[border:]
	allocBuffer = allocBuffer[:border]
	return newLine1
}

func (line _Line) LastByte() byte {
	return line[len(line)-1]
}

func (line _Line) SetLastByte(value byte) {
	line[len(line)-1] = value
}

func (line _Line) Len() int {
	return len(line)
}

func (line _Line) RemoveAt(pos int, appendByte byte) byte {
	deleteByte := line[pos]
	copy(line[pos:], line[pos+1:])
	line.SetLastByte(appendByte)
	return deleteByte
}

func (line _Line) InsertAt(pos int, value byte) byte {
	deleteByte := line.LastByte()
	copy(line[pos+1:], line[pos:])
	line[pos] = value
	return deleteByte
}
