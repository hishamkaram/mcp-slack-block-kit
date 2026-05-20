package normalizer

import "regexp"

// brTag matches `<br>`, `<br/>`, and `<br />` in any case.
// Whitespace inside the tag is permitted (e.g. `<br />`).
var brTag = regexp.MustCompile(`(?i)<br\s*/?>`)

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
		if !brTag.MatchString(original) {
			outLines = append(outLines, lines[i])
			continue
		}
		fired = true
		if lines[i].Kind == LineTable {
			// Inside a table cell: a newline would break the row,
			// so collapse `<br>` to a single space.
			rewritten := brTag.ReplaceAllString(original, " ")
			outLines = append(outLines, Line{Text: rewritten, Kind: lines[i].Kind})
			continue
		}
		// Outside tables: split the line on `<br>` boundaries and
		// emit each segment as its own line, preserving the visual
		// break the author intended. Empty segments (e.g. a
		// trailing `<br>`) become LineBlank so downstream
		// paragraph-aware repairs (V1) don't merge them with
		// content lines on a subsequent pass — the classifier
		// would later re-tag them LineBlank anyway, so producing
		// the consistent tag here is what makes the pipeline
		// idempotent.
		for _, seg := range brTag.Split(original, -1) {
			kind := LineProse
			if seg == "" {
				kind = LineBlank
			}
			outLines = append(outLines, Line{Text: seg, Kind: kind})
		}
	}
	return outLines, fired
}
