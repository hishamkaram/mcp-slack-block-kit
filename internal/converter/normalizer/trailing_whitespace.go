package normalizer

import "strings"

// applyTrailingWhitespace strips trailing whitespace from prose
// lines. Two safety carve-outs:
//
//  1. Skip code contexts (LineFenceOpen/Content/Close, LineIndentedCode).
//     CommonMark §4.5: "the contents of the code block are literally
//     what is between the fences" — stripping there corrupts
//     semantically-significant trailing whitespace (Python doctest
//     fixtures, YAML preservation tests, diff/patch content).
//
//  2. Preserve the CommonMark §6.7 hard line break (two or more
//     trailing spaces of whitespace) but ONLY in genuine paragraph
//     context — when the line is LineProse and the next non-blank
//     line is also LineProse. Table delimiter rows, end-of-input
//     lines, and lines before paragraph breaks all get stripped
//     (their trailing whitespace is overwhelmingly LLM sloppiness,
//     not a deliberate `<br>` marker). This keeps the repair
//     effective for the cases catalog C5/C9 originally targeted
//     (marked PR #2201, markdownlint MD009) while honoring §6.7
//     for any value of "two or more".
//
// Catalog codes: C5 (delimiter-row whitespace) and C9 (list-item
// trailing whitespace).
func applyTrailingWhitespace(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		switch lines[i].Kind {
		case LineFenceOpen, LineFenceContent, LineFenceClose, LineIndentedCode:
			continue
		}
		text := lines[i].Text
		trimmed := strings.TrimRight(text, " \t")
		if trimmed == text {
			continue
		}
		suffix := text[len(trimmed):]
		if isHardBreakContext(lines, i) && trimmed != "" &&
			len(suffix) >= 2 && !strings.ContainsAny(suffix, "\t") {
			continue
		}
		lines[i].Text = trimmed
		fired = true
	}
	return lines, fired
}

// isHardBreakContext reports whether the line at index i sits in a
// position where a §6.7 hard-line-break marker (`  \n`) is
// semantically meaningful. Concretely:
//
//   - The current line must be LineProse (not table/fence/blank/etc).
//   - The next non-blank line must also be LineProse (a paragraph
//     break or end-of-input means there's no break to render).
//   - The current line must NOT start with a list marker. Hard
//     breaks inside list items are rare in deliberate prose but
//     overwhelmingly common in sloppy LLM output (an LLM trailing
//     a `- one` with 3 spaces is almost never trying to render
//     `- one<br>`). Stripping there matches the original C5/C9
//     intent (marked PR #2201, markdownlint MD009) without losing
//     the genuine §6.7 case in pure prose.
//
// End-of-input, end-of-paragraph, table-cell, list-item, and
// non-prose contexts return false, letting the strip path run.
func isHardBreakContext(lines []Line, i int) bool {
	if lines[i].Kind != LineProse {
		return false
	}
	if startsWithListMarker(lines[i].Text) {
		return false
	}
	for _, next := range lines[i+1:] {
		switch next.Kind {
		case LineBlank:
			return false
		case LineProse:
			return true
		default:
			return false
		}
	}
	return false
}

// startsWithListMarker reports whether s opens with a bullet
// (`-`, `*`, `+`) or numbered (`<digits>.` / `<digits>)`) marker
// followed by a space. Matches CommonMark §5.2 list-marker syntax
// at the strict shape (the C3/C4 repairs handle the malformed
// missing-space variants separately).
func startsWithListMarker(s string) bool {
	i := 0
	for i < len(s) && i < 3 && s[i] == ' ' {
		i++
	}
	if i >= len(s) {
		return false
	}
	switch s[i] {
	case '-', '*', '+':
		return i+1 < len(s) && s[i+1] == ' '
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' && i-start < 9 {
		i++
	}
	if i == start {
		return false
	}
	if i+1 < len(s) && (s[i] == '.' || s[i] == ')') && s[i+1] == ' ' {
		return true
	}
	return false
}
