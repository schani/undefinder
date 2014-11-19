# Undefinder

Undefinder is a small utility that searches C codebases for unused
preprocessor `#define`s.

Its biggest shortcoming is that it doesn't understand comments, so it
will treat commented out macros as defined, and it will treat textual
occurrences of macros in comments as uses.

## Usage

Requires [go](https://golang.org/).

    go undefinder.go /my/c/codebase

The output can be parsed like C compiler errors/warnings.
