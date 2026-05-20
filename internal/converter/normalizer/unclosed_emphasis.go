package normalizer

import "strings"

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
		var b strings.Builder
		for i, idx := range paragraph {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(lines[idx].Text)
		}
		appended := balanceTrailingEmphasis(b.String())
		if appended != "" {
			lastIdx := paragraph[len(paragraph)-1]
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
//     followed by at least one word character with no further `*`
//     of any length later. → append `**`.
//  2. Else, the paragraph contains EXACTLY one standalone `*` run
//     (not part of any `**` or `***`) AND that run is followed by
//     at least one word character with no further `*` later. →
//     append `*`.
//
// Inputs like `**bold*` (mixed/ambiguous) and `***word**` (multiple
// runs) deliberately do nothing — repairing them risks producing
// non-idempotent output. The general balancer is V6's job.
//
// Underscore emphasis (`_italic_`) is intentionally NOT handled —
// CommonMark treats intra-word `_` as literal, so `snake_case`
// patterns would create constant false positives.
func balanceTrailingEmphasis(text string) string {
	asterRuns := asteriskRuns(text)
	// Bold case: exactly one `**` run (length 2), nothing else.
	if len(asterRuns) == 1 && asterRuns[0].length == 2 {
		if hasWordContentAfter(text, asterRuns[0].end) {
			return "**"
		}
		return ""
	}
	// Italic case: exactly one `*` run (length 1), nothing else.
	if len(asterRuns) == 1 && asterRuns[0].length == 1 {
		if hasWordContentAfter(text, asterRuns[0].end) {
			return "*"
		}
		return ""
	}
	return ""
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
