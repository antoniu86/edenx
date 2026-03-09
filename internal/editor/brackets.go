package editor

// bracketOpen maps an opening bracket to its closing counterpart.
var bracketOpen = map[byte]byte{
	'(': ')',
	'[': ']',
	'{': '}',
}

// bracketClose maps a closing bracket to its opening counterpart.
var bracketClose = map[byte]byte{
	')': '(',
	']': '[',
	'}': '{',
}

// findMatchingBracket returns the positions of the bracket under the cursor
// and its matching pair. ok is false if the cursor is not on a bracket or no
// match is found.
func findMatchingBracket(buf *Buffer) (a, b Pos, ok bool) {
	row, col := buf.cursor.Row, buf.cursor.Col
	if row >= len(buf.lines) {
		return
	}
	line := buf.lines[row]
	if col >= len(line) {
		return
	}
	ch := line[col]

	if close, isOpen := bracketOpen[ch]; isOpen {
		// Scan forward for the matching closing bracket.
		depth := 1
		for r := row; r < len(buf.lines) && depth > 0; r++ {
			start := 0
			if r == row {
				start = col + 1
			}
			for c := start; c < len(buf.lines[r]); c++ {
				switch buf.lines[r][c] {
				case ch:
					depth++
				case close:
					depth--
					if depth == 0 {
						return Pos{row, col}, Pos{r, c}, true
					}
				}
			}
		}
	} else if open, isClose := bracketClose[ch]; isClose {
		// Scan backward for the matching opening bracket.
		depth := 1
		for r := row; r >= 0 && depth > 0; r-- {
			end := len(buf.lines[r]) - 1
			if r == row {
				end = col - 1
			}
			for c := end; c >= 0; c-- {
				switch buf.lines[r][c] {
				case ch:
					depth++
				case open:
					depth--
					if depth == 0 {
						return Pos{r, c}, Pos{row, col}, true
					}
				}
			}
		}
	}
	return
}
