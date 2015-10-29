package streammatch

import (
	// "fmt"
	"io"
)

type KMP struct {
	pattern      []byte
	failfunction []int
	buf          []byte
	offset       int
	bufcursor    int
	buflen       int
	ptncursor    int
	lasterr      error
}

func NewKMP(pattern []byte) *KMP {
	plen := len(pattern)
	fail := computeFailFunction(pattern)

	bsize := 2 * plen

	if defaultBufSize > bsize {
		bsize = defaultBufSize
	}

	buf := make([]byte, bsize)

	return &KMP{pattern: pattern, failfunction: fail, buf: buf}
}

func computeFailFunction(pattern []byte) (failFunction []int) {
	plen := len(pattern)

	if plen == 0 {
		return nil
	}
	// fmt.Println("ALLOALLO", string(pattern), plen)

	f := make([]int, plen)
	f[0] = -1
	if plen > 1 {
		f[1] = 0
	}

	// fmt.Println("FOI", string(pattern), plen)

	i, j := 2, 0

	for i < plen {
		//If it matches, increase the border
		if pattern[i-1] == pattern[j] {
			j++
			f[i] = j
			i++
			//If it doesnt
		} else {
			//Try next prefix
			if j > 0 {
				j = f[j]

				//Try next position
			} else {
				f[i] = 0
				i++
			}
		}
	}

	return f
}

func (kmp *KMP) Reset() {
	kmp.offset = 0
	kmp.bufcursor = 0
	kmp.buflen = 0
	kmp.ptncursor = 0
}

func (kmp *KMP) FindMatch(reader io.Reader) (int, error) {
	plen := len(kmp.pattern)

	if plen == 0 {
		return 0, EmptyPatternError
	}

	offset, ptncursor := kmp.offset, kmp.ptncursor
	buflen, bufcursor := kmp.buflen, kmp.bufcursor
	for {
		//cursor has no unseen data
		if bufcursor >= buflen {
			//If had previous error
			if kmp.lasterr != nil {
				//save data and report error
				kmp.offset, kmp.ptncursor = offset, ptncursor
				kmp.buflen, kmp.bufcursor = buflen, bufcursor
				lasterr := kmp.lasterr
				kmp.lasterr = nil
				return -1, lasterr
			}
			buflen, kmp.lasterr = reader.Read(kmp.buf)
			bufcursor = 0
		}

		for bufcursor < buflen {
			next := kmp.buf[bufcursor]

			if kmp.pattern[ptncursor] == next {
				if ptncursor == plen-1 {
					//Match
					match_offset := offset

					//Update state machine
					offset = offset + ptncursor - kmp.failfunction[ptncursor]
					if kmp.failfunction[ptncursor] > -1 {
						ptncursor = kmp.failfunction[ptncursor]
					} else {
						ptncursor = 0
						bufcursor++
					}

					//Save data and report match
					kmp.offset, kmp.ptncursor = offset, ptncursor
					kmp.buflen, kmp.bufcursor = buflen, bufcursor

					return match_offset + plen - 1, nil
				} else {
					bufcursor++
					ptncursor++
				}
			} else {
				offset = offset + ptncursor - kmp.failfunction[ptncursor]
				if kmp.failfunction[ptncursor] > -1 {
					ptncursor = kmp.failfunction[ptncursor]
				} else {
					ptncursor = 0
					bufcursor++
				}
			}
		}

		if kmp.lasterr != nil {
			//Save data and quit
			kmp.offset, kmp.ptncursor = offset, ptncursor
			kmp.buflen, kmp.bufcursor = buflen, bufcursor
			lasterr := kmp.lasterr
			kmp.lasterr = nil
			return -1, lasterr
		}
	}
}
