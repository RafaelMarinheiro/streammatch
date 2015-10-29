package main

import (
	"bufio"
	"code.google.com/p/getopt"
	"fmt"
	"github.com/RafaelMarinheiro/streammatch"
	"github.com/mgutz/ansi"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
)

var (
	highlightCode = ansi.ColorCode("green+hu:black")
	alternateCode = ansi.ColorCode("orange+hu:black")
	resetCode     = ansi.ColorCode("reset")
)

const (
	defaultBufSize = 100 * 4096
)

func findFilesMatch(filenamepattern []string) map[string]bool {
	fileset := make(map[string]bool)
	for _, filepattern := range filenamepattern {
		filepaths, err := filepath.Glob(filepattern)
		if err != nil {
			fmt.Printf("%v", err)
		}

		for _, fp := range filepaths {
			if !fileset[fp] {
				fileset[fp] = true
			}
		}
	}
	return fileset
}

var cpuprofile string
var memprofile string

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// var needle string
	// var filepattern string
	var distance int
	var patternFile string
	var help bool
	var verbose bool
	var simpleoutput bool

	getopt.IntVarLong(&distance, "edit", 'e', "Compute the approximate matching", "max_dist")
	getopt.StringVarLong(&patternFile, "pattern", 'p', "Use line-break separated patterns from a file", "filepath")
	getopt.StringVarLong(&cpuprofile, "cpuprofile", 0, "Write cpuprofile file", "path")
	getopt.StringVarLong(&memprofile, "memprofile", 0, "Write memprofile file", "path")
	getopt.BoolVarLong(&help, "help", 'h', "Shows this message")
	getopt.BoolVarLong(&verbose, "verbose", 'v', "Show log messages")
	getopt.BoolVarLong(&simpleoutput, "simple", 's', "Show simple output")
	getopt.SetProgram("pmt")
	getopt.SetParameters("needle [haystack ...]")
	getopt.SetUsage(func() {
		getopt.PrintUsage(os.Stderr)
		fmt.Fprintf(os.Stderr, "needle - only if -p was not used\n")
		fmt.Fprint(os.Stderr, "haystack\n")
	})
	getopt.Parse()

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if help {
		getopt.Usage()
		return
	}

	var patterns []string
	var files []string

	if patternFile == "" {
		// if(len(getopt.Args() )
		if getopt.NArgs() < 1 {
			fmt.Fprintf(os.Stderr, "Needle is missing!\n")
			getopt.Usage()
			os.Exit(1)
		}
		patterns = getopt.Args()[:1]
		files = getopt.Args()[1:]

	} else {
		var err error
		patterns, err = readLinesFromFile(patternFile)
		if err != nil {
			log.Fatal(err)
		}
		files = getopt.Args()
	}

	if verbose {
		log.Printf("%v %v %v\n", patterns, files, distance)
	}

	fileset := findFilesMatch(files)

	if distance == 0 {
		if len(patterns) == 1 {
			matcher := streammatch.NewKMP([]byte(patterns[0]))
			for fp, _ := range fileset {
				file, err := os.Open(fp)
				if err != nil {
					log.Fatal(err)
				}
				matches, err := processSingleExactMatcher(file, matcher, true)

				if err != nil {
					log.Fatal(err)
				}

				if !simpleoutput {
					printMatches(fp, file, patterns, matches, distance)
					if len(matches) > 0 {
						fmt.Println("###")
					}
				} else {
					printSimpleMatches(fp, file, patterns, matches)
				}

			}
		} else if len(patterns) > 1 {
			bpatterns := make([][]byte, len(patterns))
			for i, pattern := range patterns {
				bpatterns[i] = []byte(pattern)
			}
			matcher := streammatch.NewAhoCorasick(bpatterns)
			for fp, _ := range fileset {
				file, err := os.Open(fp)
				if err != nil {
					log.Fatal(err)
				}
				matches, err := processMultiExactMatcher(file, matcher)

				if err != nil {
					log.Fatal(err)
				}

				if !simpleoutput {
					printMatches(fp, file, patterns, matches, distance)
					if len(matches) > 0 {
						fmt.Println("###")
					}
				} else {
					printSimpleMatches(fp, file, patterns, matches)
				}

			}
		}
	} else {
		matchers := make([]streammatch.Matcher, 0, len(patterns))
		for i := 0; i < len(patterns); i++ {
			matchers = append(matchers, streammatch.NewSellers([]byte(patterns[i]), distance))
		}

		for fp, _ := range fileset {
			file, err := os.Open(fp)
			if err != nil {
				log.Fatal(err)
			}

			bufreader := bufio.NewReaderSize(file, defaultBufSize)
			allmatches := make([]matchRecord, 0, 2)
			for _, matcher := range matchers {
				_, err := file.Seek(0, 0)
				if err != nil {
					log.Fatal(err)
				}
				bufreader.Reset(file)
				matches, err := processSingleExactMatcher(bufreader, matcher, false)
				if err != nil {
					log.Fatal(err)
				}
				allmatches = append(allmatches, matches...)
			}

			sort.Stable(matchRecordList(allmatches))

			if !simpleoutput {
				printMatches(fp, file, patterns, allmatches, distance)
				if len(allmatches) > 0 {
					fmt.Println("###")
				}
			} else {
				printSimpleMatches(fp, file, patterns, allmatches)
			}
		}
	}
	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		defer f.Close()
		return
	}
}

type matchRecord struct {
	patternindex int
	line         int
	lineOffset   int
	lineSize     int
	linepos      int
	isLastLine   bool
}

type matchRecordList []matchRecord

// Len is part of sort.Interface.
func (mr matchRecordList) Len() int {
	m := []matchRecord(mr)
	return len(m)
}

// Swap is part of sort.Interface.
func (mr matchRecordList) Swap(i, j int) {
	m := []matchRecord(mr)
	m[i], m[j] = m[j], m[i]
}

// Less is part of sort.Interface.
func (mr matchRecordList) Less(i, j int) bool {
	m := []matchRecord(mr)
	ri := m[i]
	rj := m[j]
	if ri.line < rj.line {
		return true
	} else if ri.line == rj.line {
		if ri.linepos <= rj.linepos {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func processSingleExactMatcher(file io.Reader, matcher streammatch.Matcher, needBuffer bool) ([]matchRecord, error) {
	var reader io.Reader

	if needBuffer {
		reader = bufio.NewReaderSize(file, defaultBufSize)
	} else {
		reader = file
	}

	linereader := streammatch.NewLineReader(reader)

	lineOffset := 0
	line := 0
	lastLineMatches := 0
	matches := make([]matchRecord, 0, 2)
	matcher.Reset()
	for {
		pos, err := matcher.FindMatch(linereader)
		if err == nil {
			matches = append(matches, matchRecord{line: line, lineOffset: lineOffset, linepos: pos})
		} else if err == streammatch.EOL {
			newLineOffset := linereader.BytesRead()
			lastLineSize := newLineOffset - lineOffset

			for i := lastLineMatches; i < len(matches); i++ {
				matches[i].lineSize = lastLineSize
			}

			lastLineMatches = len(matches)
			line++
			lineOffset = newLineOffset

			matcher.Reset()
		} else if err == io.EOF {
			newLineOffset := linereader.BytesRead()
			lastLineSize := newLineOffset - lineOffset
			for i := lastLineMatches; i < len(matches); i++ {
				matches[i].lineSize = lastLineSize
				matches[i].isLastLine = true
			}
			return matches, nil
		} else {
			return matches, err
		}
	}
}

func processMultiExactMatcher(file io.Reader, matcher streammatch.MultiMatcher) ([]matchRecord, error) {
	reader := bufio.NewReader(file)
	linereader := streammatch.NewLineReader(reader)

	lineOffset := 0
	line := 0
	lastLineMatches := 0
	matches := make([]matchRecord, 0, 2)
	matcher.Reset()
	for {
		pos, ptrns, err := matcher.FindMultipleMatches(linereader)
		if err == nil {
			for _, ptrn := range ptrns {
				matches = append(matches, matchRecord{line: line, lineOffset: lineOffset, linepos: pos, patternindex: ptrn})
			}
		} else if err == streammatch.EOL {
			newLineOffset := linereader.BytesRead()
			lastLineSize := newLineOffset - lineOffset

			for i := lastLineMatches; i < len(matches); i++ {
				matches[i].lineSize = lastLineSize
			}

			lastLineMatches = len(matches)
			line++
			lineOffset = newLineOffset

			matcher.Reset()
		} else if err == io.EOF {
			newLineOffset := linereader.BytesRead()
			lastLineSize := newLineOffset - lineOffset
			for i := lastLineMatches; i < len(matches); i++ {
				matches[i].lineSize = lastLineSize
				matches[i].isLastLine = true
			}
			return matches, nil
		} else {
			return matches, err
		}
	}
}

func printSimpleMatches(title string, reader io.ReaderAt, patterns []string, matches []matchRecord) {
	for _, match := range matches {
		fmt.Printf("%v %v %d %d\n", title, patterns[match.patternindex], match.line+1, match.linepos+1)
	}
}
func printMatches(title string, reader io.ReaderAt, patterns []string, matches []matchRecord, distance int) {
	lastLine := -1
	var line []byte
	for _, match := range matches {
		if match.line != lastLine {
			if line == nil || len(line) < match.lineSize {
				line = make([]byte, match.lineSize)
			}
			sectionReader := io.NewSectionReader(reader, int64(match.lineOffset), int64(match.lineSize))
			cur := 0
			for {
				n, err := sectionReader.Read(line[cur:match.lineSize])
				cur += n
				if err == io.EOF {
					break
				}
			}
		}
		if lastLine != -1 && match.line > lastLine+1 {
			fmt.Printf("...\n")
		}

		start := match.linepos - len(patterns[match.patternindex]) + 1
		end := match.linepos + 1
		maybe := start - distance

		if maybe < 0 {
			maybe = 0
		}
		if start < 0 {
			start = 0
		}
		if end > match.lineSize {
			end = match.lineSize
		}
		fmt.Printf("(%v:%4d:%3d) - ", title, match.line+1, match.linepos+1)

		fmt.Printf("%v", string(line[0:maybe]))
		if maybe < start {
			fmt.Printf("%v%v%v", alternateCode, string(line[maybe:start]), resetCode)
		}
		if start < end {
			fmt.Printf("%v%v%v", highlightCode, string(line[start:end]), resetCode)
		}
		fmt.Printf("%v", string(line[end:match.lineSize]))
		if match.isLastLine {
			fmt.Println()
		}
		lastLine = match.line
	}
}

func readLinesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	lines := make([]string, 0, 2)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()
	return lines, err
}
