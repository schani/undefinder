package main

import (
	"bufio"
	"log"
	"os"
	"io"
	"regexp"
	"path/filepath"
	"strings"
)

var defineRegexp *regexp.Regexp
var useRegexp *regexp.Regexp

func initRegexps () {
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

var definesDefined map[string]bool
var symbolsUsed map[string]bool

func readDefines (path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	reader := bufio.NewReader (file)

	numLines := 0
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break;
		} else if err != nil {
			log.Fatal(err)
		}

		numLines += 1

		var definedSymbol string

		defineMatch := defineRegexp.FindStringSubmatch(line)
		if defineMatch != nil {
			definedSymbol = defineMatch[1]
			definesDefined[definedSymbol] = true
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
}

var files map[string]bool

func walkFiles (path string, info os.FileInfo, err error) error {
	base := filepath.Base(path)
	if info.IsDir() {
		if strings.HasPrefix(base, ".") {
			return filepath.SkipDir
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return nil;
	}
	if strings.HasSuffix(base, ".c") || strings.HasSuffix(base, ".h") {
		files[path] = true
	}
	return nil
}

func main() {
	initRegexps()
	files = make(map[string]bool)
	definesDefined = make(map[string]bool)
	symbolsUsed = make(map[string]bool)

	filepath.Walk("/Users/schani/Work/mono/mono/mono", walkFiles)

	for path, _ := range files {
		readDefines(path)
	}

	for symbol, _ := range definesDefined {
		if !symbolsUsed[symbol] {
			log.Print("define ", symbol, " not used")
		}
	}
}
