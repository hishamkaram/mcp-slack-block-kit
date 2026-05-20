package normalizer

import "strings"

// applyTrailingWhitespace strips trailing whitespace from every line
// EXCEPT the two-trailing-spaces hard-break marker (CommonMark §6.7).
// Trailing whitespace breaks GFM table delimiter rows (marked PR
// #2201) and some Slack `markdown` block parsers; the trade-off is
// trivial and the repair is universally safe.
//
// Catalog codes: C5 (delimiter-row whitespace) and C9 (list-item
// trailing whitespace).
func applyTrailingWhitespace(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		// Hard-break preservation: a line whose trailing whitespace is
		// exactly two spaces (and contains other content) is a
		// CommonMark hard line break; we leave it alone.
		text := lines[i].Text
		trimmed := strings.TrimRight(text, " \t")
		if trimmed == text {
			continue
		}
		// Check the suffix we'd strip. If it's exactly "  " (two
		// spaces) and the line has other content, that's a hard
		// break — preserve.
		suffix := text[len(trimmed):]
		if suffix == "  " && trimmed != "" && !strings.ContainsAny(suffix, "\t") {
			continue
		}
		lines[i].Text = trimmed
		fired = true
	}
	return lines, fired
}
