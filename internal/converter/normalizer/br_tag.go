package normalizer

import "regexp"

// brTag matches `<br>`, `<br/>`, and `<br />` in any case.
// Whitespace inside the tag is permitted (e.g. `<br />`).
var brTag = regexp.MustCompile(`(?i)<br\s*/?>`)

// replaceRangesWith returns s with every byte range in ranges
// replaced by repl. Ranges must be sorted by start and
// non-overlapping (the regex FindAllStringIndex guarantees both).
func replaceRangesWith(s string, ranges [][]int, repl string) string {
	if len(ranges) == 0 {
		return s
	}
	out := make([]byte, 0, len(s))
	cursor := 0
	for _, r := range ranges {
		out = append(out, s[cursor:r[0]]...)
		out = append(out, repl...)
		cursor = r[1]
	}
	out = append(out, s[cursor:]...)
	return string(out)
}

// splitOnRanges returns the substrings of s split at the byte
// ranges, mirroring brTag.Split but for an arbitrary range slice.
// Empty segments at boundaries are preserved (matches Split's
// semantics: a leading or trailing match produces an empty string).
func splitOnRanges(s string, ranges [][]int) []string {
	if len(ranges) == 0 {
		return []string{s}
	}
	out := make([]string, 0, len(ranges)+1)
	cursor := 0
	for _, r := range ranges {
		out = append(out, s[cursor:r[0]])
		cursor = r[1]
	}
	out = append(out, s[cursor:])
	return out
}

// applyBRTag rewrites `<br>` family tags to a newline OUTSIDE table
// cells and to a space INSIDE table cells. Slack's `markdown` block
// does not render raw HTML, and rich_text has no in-cell line-break
// primitive — so a literal `<br>` would render as visible HTML tag
// text. The newline replacement gives the prose a natural break;
// inside a table cell the space avoids breaking the row.
//
// Skips fenced and indented code blocks (those should render
// literally).
//
// Catalog code: R8. Evidence: MacMD Viewer line-break guide
// ("Reserve `<br>` for situations where you need precise control —
// like inside table cells").
func applyBRTag(lines []Line, _ Options) ([]Line, bool) {
	var (
		fired    bool
		outLines = make([]Line, 0, len(lines))
	)
	for i := range lines {
		switch lines[i].Kind {
		case LineFenceOpen, LineFenceContent, LineFenceClose, LineIndentedCode:
			outLines = append(outLines, lines[i])
			continue
		}
		original := lines[i].Text
		matches := brTag.FindAllStringIndex(original, -1)
		if len(matches) == 0 {
			outLines = append(outLines, lines[i])
			continue
		}
		// Inline-code-span guard: skip any `<br>` whose start byte
		// falls inside backtick-delimited content. CommonMark §6.1
		// makes code-span content literal — a `<br>` documented
		// inside backticks (e.g. "Use `<br>` for HTML breaks")
		// must survive unchanged.
		mask := inlineCodeMask(original)
		live := matches[:0]
		for _, m := range matches {
			if m[0] < len(mask) && mask[m[0]] {
				continue
			}
			live = append(live, m)
		}
		if len(live) == 0 {
			outLines = append(outLines, lines[i])
			continue
		}
		fired = true
		if lines[i].Kind == LineTable {
			// Inside a table cell: a newline would break the row,
			// so collapse <br> to a single space. Apply replacements
			// in reverse so byte positions stay stable.
			rewritten := replaceRangesWith(original, live, " ")
			outLines = append(outLines, Line{Text: rewritten, Kind: lines[i].Kind})
			continue
		}
		// Outside tables: split the line on the surviving (not-in-
		// code-span) `<br>` boundaries and emit each segment as its
		// own line, preserving the visual break the author intended.
		// Empty segments (e.g. a trailing `<br>`) become LineBlank
		// so downstream paragraph-aware repairs (V1) don't merge
		// them with content lines on a subsequent pass — the
		// classifier would later re-tag them LineBlank anyway, so
		// producing the consistent tag here is what makes the
		// pipeline idempotent.
		for _, seg := range splitOnRanges(original, live) {
			kind := LineProse
			if seg == "" {
				kind = LineBlank
			}
			outLines = append(outLines, Line{Text: seg, Kind: kind})
		}
	}
	return outLines, fired
}
