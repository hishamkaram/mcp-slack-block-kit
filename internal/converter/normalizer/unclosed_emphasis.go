package normalizer

// applyUnclosedEmphasis balances `**bold**` and `*italic*` markers
// inside a paragraph when the model stopped before closing the
// emphasis. Conservative: operates per paragraph (delimited by
// anything other than LineProse), only considers the unambiguous
// single-opener case, and appends the matching closer.
//
// Why so conservative: paragraph-level emphasis balancing is the
// most failure-prone repair in the catalog (V6's full balancer
// is gated behind RepairMismatchedEmphasis). V1 handles only the
// pure "model truncated mid-emphasis" case where the closer is
// provably missing because no candidate exists after the opener.
//
// Skips fenced and indented code blocks.
//
// Catalog code: V1. Evidence: remend npm handlers list; Cultman
// Sachs Medium "opens … and never closes".
func applyUnclosedEmphasis(lines []Line, _ Options) ([]Line, bool) {
	var (
		fired     bool
		paragraph []int // indices of LineProse lines in current paragraph
	)
	flush := func() {
		if len(paragraph) == 0 {
			return
		}
		// V1 only operates on the LAST line of the paragraph. Multi-
		// line emphasis where the opener sits on an earlier line is
		// genuinely ambiguous (the closer could belong to either
		// line). Restricting to the last line keeps the repair
		// local and idempotent — appending a `*` to the last line
		// can't create new patterns elsewhere (verified by fuzz
		// against `*0\n# `).
		lastIdx := paragraph[len(paragraph)-1]
		appended := balanceTrailingEmphasis(lines[lastIdx].Text)
		if appended != "" {
			lines[lastIdx].Text += appended
			fired = true
		}
		paragraph = nil
	}
	for i := range lines {
		if lines[i].Kind == LineProse {
			paragraph = append(paragraph, i)
			continue
		}
		flush()
	}
	flush()
	return lines, fired
}

// balanceTrailingEmphasis returns a closer to append to a paragraph
// that has an unmistakable unclosed emphasis at its end. We are
// deliberately conservative — paragraph-level asterisk balancing is
// the hardest pattern in the catalog (V6 handles the general case
// behind an opt-in flag). V1 only fires when:
//
//  1. The paragraph contains EXACTLY one `**` run AND that run is
//     left-flanking (line-start or preceded by whitespace) AND has
//     word content after it. → append `**`.
//  2. Else, the paragraph contains EXACTLY one standalone `*` run
//     under the same left-flanking guard. → append `*`.
//
// The left-flanking guard mirrors CommonMark §6.2's rule: a `*` is
// NOT an emphasis opener when it's flanked by alphanumerics on both
// sides (e.g. intra-word like `10*x`). Without this guard, headers
// like `Pricing: 10*x + 5_y` would get a stray closer appended,
// turning literal text into an italic span.
//
// Inputs like `**bold*` (mixed/ambiguous) and `***word**` (multiple
// runs) deliberately do nothing. The general balancer is V6's job.
//
// Underscore emphasis (`_italic_`) is intentionally NOT handled —
// CommonMark treats intra-word `_` as literal, so `snake_case`
// patterns would create constant false positives.
func balanceTrailingEmphasis(text string) string {
	asterRuns := asteriskRuns(text)
	if len(asterRuns) != 1 {
		return ""
	}
	run := asterRuns[0]
	if !isLeftFlanking(text, run.start) {
		return ""
	}
	if !hasWordContentAfter(text, run.end) {
		return ""
	}
	switch run.length {
	case 1:
		return "*"
	case 2:
		return "**"
	}
	return ""
}

// isLeftFlanking reports whether the character at position pos is
// left-flanking per CommonMark §6.2 (suitable as an emphasis opener).
// True when either (a) pos is line start or (b) the previous byte is
// whitespace or non-word punctuation.
func isLeftFlanking(s string, pos int) bool {
	if pos == 0 {
		return true
	}
	prev := s[pos-1]
	return !isWordByte(prev)
}

// asteriskRun records a contiguous `*` run.
type asteriskRun struct {
	start, end, length int // end is exclusive
}

// asteriskRuns returns every contiguous `*` run in s.
func asteriskRuns(s string) []asteriskRun {
	var out []asteriskRun
	i := 0
	for i < len(s) {
		if s[i] != '*' {
			i++
			continue
		}
		start := i
		for i < len(s) && s[i] == '*' {
			i++
		}
		out = append(out, asteriskRun{start: start, end: i, length: i - start})
	}
	return out
}

// hasWordContentAfter reports whether s contains at least one word
// (letter/digit) byte at or after position start. The conservative
// V1 guard uses this to distinguish "opener followed by content"
// (which we repair) from "stray asterisk" (which we don't).
func hasWordContentAfter(s string, start int) bool {
	for i := start; i < len(s); i++ {
		if isWordByte(s[i]) {
			return true
		}
	}
	return false
}
