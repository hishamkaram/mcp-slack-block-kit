package normalizer

import "regexp"

// atxNoSpace matches an ATX header opener that's missing the required
// space between the `#` run and the heading text. CommonMark §4.2
// requires "the # characters … followed by a space, tab, or end of
// line". markdownlint MD018 catches the same pattern.
//
// Captures: (1) leading indent (≤3 spaces per CommonMark), (2) the `#`
// run (1–6 characters), (3) the first character of the heading text
// (must NOT be `#` itself or a space — closer/well-formed cases are
// excluded by the regex).
var atxNoSpace = regexp.MustCompile(`^( {0,3})(#{1,6})([^\s#])`)

// applyATXHeaderSpace inserts a space between the leading `#` run and
// the heading text when missing. Skips fence/indented/table contexts.
//
// False-positive guard: the regex rejects `####` (all-hashes lines,
// could be a closer) and `#hashtag` mid-paragraph (the line-start
// anchor + ≤3-space indent + the LineProse Kind check together restrict
// matches to lines whose first non-space character is `#`).
//
// Catalog code: V5. Evidence: eslint-markdown
// no-missing-atx-heading-space rule, markdownlint MD018, mkdocs #2127.
func applyATXHeaderSpace(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		if lines[i].Kind != LineProse {
			continue
		}
		m := atxNoSpace.FindStringSubmatch(lines[i].Text)
		if m == nil {
			continue
		}
		// Rebuild: indent + hashes + " " + remainder.
		lines[i].Text = m[1] + m[2] + " " + lines[i].Text[len(m[1])+len(m[2]):]
		fired = true
	}
	return lines, fired
}
