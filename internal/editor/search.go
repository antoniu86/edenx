package editor

import (
	"regexp"
	"strings"
)

// Match represents a found search result in the buffer
type Match struct {
	Start Pos
	End   Pos
}

// FindAll finds all occurrences of term in the buffer lines.
// When useRegex is true the term is compiled as a regular expression;
// an error is returned if the pattern is invalid.
func FindAll(lines []string, term string, caseInsensitive, useRegex bool) ([]Match, error) {
	if term == "" {
		return nil, nil
	}
	if useRegex {
		prefix := ""
		if caseInsensitive {
			prefix = "(?i)"
		}
		re, err := regexp.Compile(prefix + term)
		if err != nil {
			return nil, err
		}
		return findAllRegex(lines, re), nil
	}

	// Plain substring search.
	var matches []Match
	searchTerm := term
	for row, line := range lines {
		haystack := line
		if caseInsensitive {
			haystack = strings.ToLower(line)
			searchTerm = strings.ToLower(term)
		}
		col := 0
		for {
			idx := strings.Index(haystack[col:], searchTerm)
			if idx < 0 {
				break
			}
			start := Pos{row, col + idx}
			end := Pos{row, col + idx + len(searchTerm)}
			matches = append(matches, Match{Start: start, End: end})
			col += idx + len(searchTerm)
			if col >= len(haystack) {
				break
			}
		}
	}
	return matches, nil
}

func findAllRegex(lines []string, re *regexp.Regexp) []Match {
	var matches []Match
	for row, line := range lines {
		for _, loc := range re.FindAllStringIndex(line, -1) {
			matches = append(matches, Match{
				Start: Pos{row, loc[0]},
				End:   Pos{row, loc[1]},
			})
		}
	}
	return matches
}

// RegexExpand returns the replacement string with capture group references
// ($1, $2, $0 etc.) expanded using the submatches of re against src.
func RegexExpand(re *regexp.Regexp, src, repl string) string {
	idx := re.FindStringSubmatchIndex(src)
	if idx == nil {
		return repl
	}
	return string(re.ExpandString(nil, repl, src, idx))
}

// MatchContains returns true if pos falls within the match range
func MatchContains(m Match, row, col int) bool {
	p := Pos{row, col}
	return !posLess(p, m.Start) && posLess(p, m.End)
}
