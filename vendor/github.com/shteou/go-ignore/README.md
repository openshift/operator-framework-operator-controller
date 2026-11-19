# go-ignore

A simple gitignore/dockerignore format parser.

This package parses .gitignore and .dockerignore files into a publicly exposed struct
for usage in other packages. It does not provide any matching capabilities, nor does it support
marshalling the Entries.

## .gitignore vs .dockerignore

There are functional differences between .gitignore and .dockerignore. This is manifest in
how files/directories are matched, but that functionality is beyond the scope of this package.

From a file structure perspective the two formats are deemed equivalent, in that both support
paths, negated paths, comments, and whitespace lines.

## API

```
type Entry struct {
  Kind     string
  Value    string
  Original string
}
```

The Parse functions return an array of Entry structs (or an error).

* Kind indicates the type of the entry, either Empty, Comment, Path, or NegatedPath
* Value indicates the effective value of the entry, with the following transformations
  * Empty lines are always empty!
  * Comments are left trimmed for whitespace
  * Paths are unescaped
  * NegatedPaths are unescaped with the negation character removed
* Original provides the original line, unaltered

## Example Usage

```go
package main

import (
        "fmt"

        ignore "github.com/shteou/go-ignore"
)

func main() {
        lines := []string{"# A comment", ".ignore*", "!ignore.dont"}
        ignore, _ := ignore.ParseIgnoreLines(lines)
        fmt.Printf("%v+\n", ignore)
        fmt.Printf("Line 0: Type '%s', Value '%s', Original '%s'\n", ignore[0].Kind, ignore[0].Value, ignore[0].Original)
}
```
