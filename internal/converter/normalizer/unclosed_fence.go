package normalizer

// applyUnclosedFence appends a closing fence to the source when a
// fenced code block was opened but never closed. Without this repair
// an unclosed fence would consume the rest of the document — the
// remaining prose, lists, headers, etc. all render as one giant
// `rich_text_preformatted` block. By far the highest-severity
// breakage in the catalog.
//
// Detection: the fence walker has already classified every line.
// After classify, if the final accumulated state is "still inside a
// fence" (the last LineFenceOpen had no matching LineFenceClose)
// then we append a closing fence as a new line. The fence character
// (` or ~) and length are chosen to match the opener.
//
// Catalog code: V3. Evidence: Cultman Sachs Medium "orphaned code
// blocks"; Vercel changelog for the remend package; streamdown
// unterminated-block docs. Highest priority.
func applyUnclosedFence(lines []Line, _ Options) ([]Line, bool) {
	openerIdx := -1
	for i, l := range lines {
		switch l.Kind {
		case LineFenceOpen:
			openerIdx = i
		case LineFenceClose:
			openerIdx = -1
		}
	}
	if openerIdx < 0 {
		return lines, false
	}
	// Find the opener's fence character + length to match it.
	opener := lines[openerIdx].Text
	leading := 0
	for leading < len(opener) && opener[leading] == ' ' {
		leading++
	}
	if leading >= len(opener) {
		return lines, false
	}
	fenceChar := opener[leading]
	if fenceChar != '`' && fenceChar != '~' {
		return lines, false
	}
	fenceLen := 0
	for leading+fenceLen < len(opener) && opener[leading+fenceLen] == fenceChar {
		fenceLen++
	}
	if fenceLen < 3 {
		return lines, false
	}
	// Build the closer. CommonMark requires only the fence run; no
	// info string. Match the opener's indent so the closer aligns
	// visually.
	closer := opener[:leading] + repeatByte(fenceChar, fenceLen)
	return append(lines, Line{Text: closer, Kind: LineFenceClose}), true
}

func repeatByte(b byte, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return string(out)
}
