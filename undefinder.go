package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

type location struct {
	path string
	line int
}

var defineRegexp *regexp.Regexp
var useRegexp *regexp.Regexp

func initRegexps() {
	var err error

	defineRegexp, err = regexp.Compile("#\\s*define\\s+([A-Za-z_]\\w*)")
	if err != nil {
		log.Fatal(err)
	}
	defineRegexp.Longest()

	useRegexp, err = regexp.Compile("(^|[^A-Za-z_])([A-Za-z_]\\w*)")
	if err != nil {
		log.Fatal(err)
	}
	useRegexp.Longest()
}

// Read defined and used symbols in the file at path.  Return a set of
// defined and a set of used symbols.
func readDefines(path string) (map[string]location, map[string]bool) {
	definesDefined := make(map[string]location)
	symbolsUsed := make(map[string]bool)

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	numLines := 0
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		numLines += 1

		var definedSymbol string

		defineMatch := defineRegexp.FindStringSubmatch(line)
		if defineMatch != nil {
			definedSymbol = defineMatch[1]
			definesDefined[definedSymbol] = location{path, numLines}
			//log.Print("line ", numLines, " defined ", definedSymbol)
		}

		useMatches := useRegexp.FindAllStringSubmatch(line, -1)
		for _, match := range useMatches {
			symbol := match[2]
			//log.Print("line ", numLines, " in ", path, " uses ", symbol)
			if symbol != definedSymbol {
				symbolsUsed[symbol] = true
			}
		}
	}

	return definesDefined, symbolsUsed
}

func walkFilesForProcessFunc(processFile func(path string)) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		base := filepath.Base(path)
		if info.IsDir() {
			if strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if strings.HasSuffix(base, ".c") || strings.HasSuffix(base, ".h") {
			processFile(path)
		}
		return nil
	}
}

type stringAccumulator struct {
	strings []string
}

func (this *stringAccumulator) String() string {
	return "PATTERN"
}

func (this *stringAccumulator) Set(s string) error {
	this.strings = append(this.strings, s)
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s OPTIONS DIR\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
}

func main() {
	runtime.GOMAXPROCS(8)

	type fileResults struct {
		definesDefined map[string]location
		symbolsUsed    map[string]bool
	}

	flag.Usage = usage

	excludeDefines := stringAccumulator{[]string{}}
	flag.Var(&excludeDefines, "exclude-defines", "exclude defines from files with base names that match PATTERN")

	flag.Parse()

	args := flag.Args()

	if len(args) != 1 {
		usage()
		os.Exit(1)
	}

	rootPath := args[0]

	initRegexps()
	definesDefined := make(map[string]location)
	symbolsUsed := make(map[string]bool)
	countChannel := make(chan bool, 16)
	resultsChannel := make(chan fileResults)

	walkFiles := walkFilesForProcessFunc(func(path string) {
		base := filepath.Base(path)
		includeDefined := true
		for _, pattern := range excludeDefines.strings {
			matched, _ := filepath.Match(pattern, base)
			if matched {
				includeDefined = false
				break
			}
		}
		countChannel <- true
		go func() {
			defined, used := readDefines(path)
			if !includeDefined {
				defined = make(map[string]location)
			}
			resultsChannel <- fileResults{defined, used}
		}()
	})

	go func() {
		filepath.Walk(rootPath, walkFiles)
		close(countChannel)
	}()

	for {
		_, ok := <-countChannel
		if !ok {
			break
		}
		results := <-resultsChannel
		for symbol, location := range results.definesDefined {
			definesDefined[symbol] = location
		}
		for symbol, _ := range results.symbolsUsed {
			symbolsUsed[symbol] = true
		}
	}

	unusedSymbols := []string{}

	for symbol, _ := range definesDefined {
		if !symbolsUsed[symbol] {
			unusedSymbols = append(unusedSymbols, symbol)
		}
	}

	sort.Strings(unusedSymbols)

	for _, symbol := range unusedSymbols {
		location := definesDefined[symbol]
		fmt.Printf("%s:%d: define '%s' not used\n", location.path, location.line, symbol)
	}
}
