package ignore

import (
	"strings"
)

type Entry struct {
	Kind     string
	Value    string
	Original string
}

// TrimRightSpace trims any trailing spaces unless preceeded by a \
func TrimRightSpace(s string) string {
	stop := len(s) - 1
	for ; stop >= 0 && s[stop] == ' '; stop-- {
	}

	if stop >= 0 && s[stop] == '\\' {
		stop++
	}

	return s[:stop+1]
}

// Unescape Unuscape any backslashes
func Unescape(s string) string {
	res := ""

	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			res = res + string(s[i])
		} else if i < len(s)-1 && s[i+1] == '\\' {
			res = res + string(s[i])
			i++
		}
	}

	return res
}

// ParseIgnoreLines Parse the supplied lines into a Go representation of that those
// ignore entries
func ParseIgnoreLines(lines []string) ([]Entry, error) {
	ignoreEntries := []Entry{}

	for _, l := range lines {
		stripped := TrimRightSpace(l)

		if stripped == "" {
			ignoreEntries = append(ignoreEntries, Entry{"Empty", l, l})
		} else if strings.HasPrefix(l, "#") {
			ignoreEntries = append(ignoreEntries, Entry{"Comment", strings.TrimSpace(stripped[1:]), l})
		} else if strings.HasPrefix(l, "!") {
			ignoreEntries = append(ignoreEntries, Entry{"NegatedPath", Unescape(stripped[1:]), l})
		} else {
			ignoreEntries = append(ignoreEntries, Entry{"Path", Unescape(stripped), l})
		}
	}

	return ignoreEntries, nil
}

// ParseIgnoreBytes Parse the supplied byte array into a Go representation of those
// gnore entries
func ParseIgnoreBytes(bytes []byte) ([]Entry, error) {
	lines := strings.Split(string(bytes), "\n")
	return ParseIgnoreLines(lines)
}
