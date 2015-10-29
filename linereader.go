package streammatch

import (
	// "fmt"
	"io"
)

type LineReader struct {
	Reader   io.Reader
	bufbegin int
	bufend   int
	buf      []byte
	lasterr  error
	total    int
}

//Creates a Line Reader
func NewLineReader(reader io.Reader) *LineReader {
	return &LineReader{Reader: reader}
}

//Todo treat /r/n

//Returns the number of bytes read so far
func (lr *LineReader) BytesRead() int {
	return lr.total - (lr.bufend - lr.bufbegin)
}

//Returns EOL (End Of Line) error when
//a EOL is found
func (lr *LineReader) Read(p []byte) (int, error) {
	written := 0
	var reterr error
	// defer func() { fmt.Printf("Written: %v - Read: %v - reterr: %v\n", written, lr.BytesRead(), reterr) }()
	//Write what we have in the buffer
	for lr.bufend > lr.bufbegin && written < len(p) {
		if lr.buf[lr.bufbegin] == '\n' {
			lr.bufbegin++
			reterr = EOL
			return written, EOL
		}
		p[written] = lr.buf[lr.bufbegin]
		lr.bufbegin++
		written++
	}

	if written >= len(p) {
		return written, nil
	}
	//Read more data
	if lr.buf == nil || len(lr.buf) < len(p) {
		lr.buf = make([]byte, len(p)+3)
	}
	n, err := lr.Reader.Read(lr.buf)
	lr.total += n
	lr.bufbegin, lr.bufend = 0, n

	lasterr := lr.lasterr
	lr.lasterr = err
	if lasterr != nil {
		reterr = lasterr
		return written, reterr
	}

	//Write what we have in the buffer
	for lr.bufend > lr.bufbegin && written < len(p) {
		if lr.buf[lr.bufbegin] == '\n' {
			lr.bufbegin++
			reterr = EOL
			return written, EOL
		}
		p[written] = lr.buf[lr.bufbegin]
		lr.bufbegin++
		written++
	}
	return written, nil
}
