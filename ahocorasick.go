package streammatch

import (
	// "fmt"
	"io"
)

type AhoCorasick struct {
	patterns [][]byte

	//States
	trie            [][256]int
	failfunction    []int
	occurrences     []int
	occurrence_last []int

	//stream
	offset    int
	buf       []byte
	buflen    int
	bufcursor int
	state     int

	//err
	lasterr error
}

func NewAhoCorasick(patterns [][]byte) *AhoCorasick {

	trie, occurrences := aho_computeTrie(patterns)
	failfunction, occurrences_last := aho_computeFailFunction(trie, occurrences)
	// fmt.Printf("%v\n%v\n%v\n", occurrences, failfunction, occurrences_last)

	bsize := defaultBufSize

	// bsize = 2

	buf := make([]byte, bsize)

	return &AhoCorasick{
		patterns:        patterns,
		trie:            trie,
		failfunction:    failfunction,
		occurrences:     occurrences,
		occurrence_last: occurrences_last,
		buf:             buf,
	}
}

func aho_computeTrie(patterns [][]byte) ([][256]int, []int) {
	num_patterns := len(patterns)

	num_states := 0
	trie := make([][256]int, 0, num_patterns)
	occurrences := make([]int, 0, num_patterns)

	num_states++
	trie = append(trie, [256]int{})
	occurrences = append(occurrences, -1)

	for p := 0; p < num_patterns; p++ {
		pat := patterns[p]
		len_pat := len(pat)

		cur_state := 0
		for i := 0; i < len_pat; i++ {
			char := pat[i]
			if trie[cur_state][char] == 0 {
				trie[cur_state][char] = num_states

				num_states++
				trie = append(trie, [256]int{})
				occurrences = append(occurrences, -1)
			}
			cur_state = trie[cur_state][char]
		}
		occurrences[cur_state] = p
	}

	return trie, occurrences
}

func aho_computeFailFunction(trie [][256]int, occurrences []int) ([]int, []int) {
	num_states := len(occurrences)
	failfunction := make([]int, num_states)
	occurrence_pointer := make([]int, num_states)

	last := 0
	queue := make([]int, num_states)

	queue[last] = 0
	last++

	//For each state in the queue
	for next := 0; next < last; next++ {
		state := queue[next]
		for char := 0; char < 256; char++ {
			if trie[state][char] != 0 {
				next_state := trie[state][char]

				//Search for previous that match the character
				fail := failfunction[state]
				for fail != 0 && trie[fail][char] == 0 {
					fail = failfunction[fail]
				}

				fail = trie[fail][char]

				if fail != next_state {
					failfunction[next_state] = fail

					//Search for previous that has occurrence
					occur := fail
					for occur != 0 && occurrences[occur] == -1 {
						occur = occurrence_pointer[occur]
					}
					occurrence_pointer[next_state] = occur
				}

				//Push the state to the queue
				queue[last] = next_state
				last++
			}
		}
	}

	return failfunction, occurrence_pointer
}

func (aho *AhoCorasick) Reset() {
	aho.state = 0
	aho.offset = 0
	aho.buflen = 0
	aho.bufcursor = 0
}

func (aho *AhoCorasick) FindMultipleMatches(reader io.Reader) (int, []int, error) {

	state, offset := aho.state, aho.offset
	buflen, bufcursor := aho.buflen, aho.bufcursor
	for {
		if bufcursor >= buflen {
			if aho.lasterr != nil {
				aho.state, aho.offset = state, offset
				aho.buflen, aho.bufcursor = buflen, bufcursor
				lasterr := aho.lasterr
				aho.lasterr = nil
				return -1, nil, lasterr
			}
			offset = offset + buflen
			buflen, aho.lasterr = reader.Read(aho.buf)
			bufcursor = 0
		}
		// fmt.Printf("AHOBUF: %v\n", string(aho.buf[bufcursor:buflen]))
		for bufcursor < buflen {
			next := aho.buf[bufcursor]

			for state != 0 && aho.trie[state][next] == 0 {
				state = aho.failfunction[state]
			}

			state = aho.trie[state][next]

			//Has occurrences
			if aho.occurrences[state] != -1 || aho.occurrences[aho.occurrence_last[state]] != -1 {

				occur := make([]int, 0)

				//Collect occurrences
				st := state
				for {
					if aho.occurrences[st] != -1 {
						occur = append(occur, aho.occurrences[st])
					}
					if st == 0 {
						break
					}
					st = aho.occurrence_last[st]
				}

				//Save state
				aho.state, aho.offset = state, offset
				aho.buflen, aho.bufcursor = buflen, bufcursor+1
				return offset + bufcursor, occur, nil
			}
			bufcursor++
		}

		if aho.lasterr != nil {
			//Save state
			aho.state, aho.offset = state, offset
			aho.buflen, aho.bufcursor = buflen, bufcursor
			lasterr := aho.lasterr
			aho.lasterr = nil
			return -1, nil, lasterr
		}
	}
}
