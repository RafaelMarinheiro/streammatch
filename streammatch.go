package streammatch

import (
	"errors"
	"io"
)

const (
	defaultBufSize = 4096
)

var (
	EmptyPatternError = errors.New("Empty Pattern")
	EOL               = errors.New("End Of Line")
)

type Resetter interface {
	Reset()
}

type Matcher interface {
	//Resets the Matcher
	Resetter

	//Finds next match and return the byte offset of the end of the match
	//May return an error as well
	FindMatch(reader io.Reader) (offset int, err error)
}

type MultiMatcher interface {
	Resetter
	FindMultipleMatches(reader io.Reader) (int, []int, error)
}
