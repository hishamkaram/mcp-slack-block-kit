package normalizer

// applyUnclosedInlineCode appends a closer to any line whose
// backtick runs leave an unmatched opener. Per CommonMark §6.1, a
// code span is delimited by two equal-length backtick runs and any
// backticks BETWEEN them are content (regardless of length).
//
// Algorithm:
//  1. Walk runs left-to-right; for each not-yet-matched run, look
//     ahead for the next same-length unmatched run and pair them.
//     Intervening runs are left alone (they become span content).
//  2. After the walk, the unmatched runs in input order are
//     truly unclosed. Append ONE closer matching the FIRST
//     unmatched run's length — any subsequent unmatched runs then
//     become content of the newly-closed span.
//
// Skips fenced and indented code blocks (their content is literal).
//
// Catalog code: V2. Evidence: remend npm; streaming-markdown README.
func applyUnclosedInlineCode(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		switch lines[i].Kind {
		case LineFenceOpen, LineFenceContent, LineFenceClose, LineIndentedCode:
			continue
		}
		text := lines[i].Text
		// Idempotence guard #1: if the last non-whitespace byte is
		// already a backtick, an appended closer would merge with
		// it into a larger run, producing a NEW unmatched opener
		// on the next pass. Skip.
		if endsInBacktick(text) {
			continue
		}
		runs, positions := backtickRunsWithPositions(text)
		if len(runs) == 0 {
			continue
		}
		unmatched, firstPos := unmatchedRunsWithFirstPos(runs, positions)
		if len(unmatched) == 0 {
			continue
		}
		// Idempotence guard #2: only fire when there's at least one
		// non-backtick byte between the unmatched opener and end of
		// line. Without this, lines that are entirely backticks
		// (e.g. `` `` ``) would non-idempotently grow on every pass.
		openerEnd := firstPos + unmatched[0]
		if !hasNonBacktickAfter(text, openerEnd) {
			continue
		}
		// Append exactly one closer of the first unmatched run's
		// length. That collapses every subsequent unmatched run
		// into the new code span's content — semantically
		// equivalent to what a CommonMark-aware caller would do
		// by hand.
		closer := repeatByteBytes('`', unmatched[0])
		lines[i].Text = text + string(closer)
		fired = true
	}
	return lines, fired
}

// backtickRunsWithPositions returns both run lengths and the start
// byte offset of each run, used by the unclosed-opener guard.
func backtickRunsWithPositions(s string) (lengths, starts []int) {
	i := 0
	for i < len(s) {
		if s[i] != '`' {
			i++
			continue
		}
		start := i
		for i < len(s) && s[i] == '`' {
			i++
		}
		lengths = append(lengths, i-start)
		starts = append(starts, start)
	}
	return lengths, starts
}

// unmatchedRunsWithFirstPos returns the unmatched run lengths plus
// the byte position of the first unmatched run (or -1 when all are
// matched). Wraps unmatchedRuns with positional information so the
// caller can apply a content-after-opener guard.
func unmatchedRunsWithFirstPos(lengths, starts []int) (unmatched []int, firstPos int) {
	matched := make([]bool, len(lengths))
	i := 0
	for i < len(lengths) {
		if matched[i] {
			i++
			continue
		}
		paired := false
		for j := i + 1; j < len(lengths); j++ {
			if matched[j] {
				continue
			}
			if lengths[j] == lengths[i] {
				matched[i] = true
				matched[j] = true
				for k := i + 1; k < j; k++ {
					matched[k] = true
				}
				i = j + 1
				paired = true
				break
			}
		}
		if !paired {
			i++
		}
	}
	firstPos = -1
	for idx, m := range matched {
		if m {
			continue
		}
		unmatched = append(unmatched, lengths[idx])
		if firstPos == -1 {
			firstPos = starts[idx]
		}
	}
	return unmatched, firstPos
}

// hasNonBacktickAfter reports whether s has any non-backtick byte
// at or after start. The V2 guard uses this to skip lines that are
// entirely backticks (no content to close around).
func hasNonBacktickAfter(s string, start int) bool {
	for i := start; i < len(s); i++ {
		if s[i] != '`' {
			return true
		}
	}
	return false
}

// endsInBacktick reports whether s's last non-whitespace byte is `\“.
// V2's idempotence guard skips lines for which this is true.
func endsInBacktick(s string) bool {
	for i := len(s) - 1; i >= 0; i-- {
		switch s[i] {
		case ' ', '\t':
			continue
		case '`':
			return true
		default:
			return false
		}
	}
	return false
}

// repeatByteBytes returns a []byte of length n filled with b.
func repeatByteBytes(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
