package streammatch

import (
	// "fmt"
	"io"
)

type Sellers struct {
	pattern []byte
	maxdist int

	//States
	dist    []int
	distptr int

	//stream
	offset    int
	buf       []byte
	buflen    int
	bufcursor int

	//err
	lasterr error
}

func NewSellers(pattern []byte, maxdist int) *Sellers {
	plen := len(pattern)

	bsize := 2 * plen

	if defaultBufSize > bsize {
		bsize = defaultBufSize
	}

	buf := make([]byte, bsize)

	dist := make([]int, 2*(plen+1))
	for i := 0; i <= plen; i++ {
		dist[2*i+1] = i
	}
	return &Sellers{pattern: pattern, maxdist: maxdist, dist: dist, buf: buf}
}

func (sel *Sellers) Reset() {
	sel.distptr = 0
	for i := 0; i <= len(sel.pattern); i++ {
		sel.dist[2*i+1] = i
	}

	sel.offset = 0
	sel.buflen = 0
	sel.bufcursor = 0
}

func (sel *Sellers) FindMatch(reader io.Reader) (int, error) {
	lenp := len(sel.pattern)

	for {
		if sel.bufcursor >= sel.buflen {
			if sel.lasterr != nil {
				lasterr := sel.lasterr
				sel.lasterr = nil
				return -1, lasterr
			}
			sel.offset += sel.buflen
			sel.buflen, sel.lasterr = reader.Read(sel.buf)
			sel.bufcursor = 0
		}

		for sel.bufcursor < sel.buflen {
			next := sel.buf[sel.bufcursor]
			// debug()
			cur := sel.distptr
			other := 1 - cur

			sel.dist[2*0+cur] = 0
			last := 0 //sel.dist[cur][i-1]
			la := -1  //sel.dist[other][i-1]
			lb := 0   //sel.dist[other][i]
			for i := 1; i <= lenp; i++ {
				la = lb
				lb = sel.dist[2*i+other]

				val := last + 1
				if lb+1 < val {
					val = lb + 1
				}
				if la+1 < val {
					val = la + 1
				}

				if la < val && next == sel.pattern[i-1] {
					val = la
				}
				sel.dist[2*i+cur] = val
				last = val
			}

			sel.distptr = other
			sel.bufcursor++
			if sel.dist[2*lenp+cur] <= sel.maxdist {
				return sel.offset + sel.bufcursor - 1, nil
			}
		}

		if sel.lasterr != nil {
			lasterr := sel.lasterr
			sel.lasterr = nil
			return -1, lasterr
		}
	}
}
