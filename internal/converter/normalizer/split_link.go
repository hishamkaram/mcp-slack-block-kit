package normalizer

import "regexp"

// splitLinkLineEnd matches a line that ends with a closing link
// bracket (optionally followed by whitespace). The capture is the
// line content up to and including the `]`.
var splitLinkLineEnd = regexp.MustCompile(`^(.*\])\s*$`)

// splitLinkLineStart matches a line that begins with an opening paren
// containing URL-ish content and ending with a closing paren. Captures
// (1) the parenthesized URL, (2) anything trailing the close paren.
var splitLinkLineStart = regexp.MustCompile(`^\s*\(([^)\s]+)\)(.*)$`)

// applySplitLink repairs `[label]\n(url)` patterns where the LLM has
// inserted a line break between the link's label and its URL. CommonMark
// rejects the construct, so goldmark parses it as two separate text
// runs, producing literal `[label]` and `(url)` text in the rendered
// output. Skips fenced and indented code blocks and table cells.
//
// Algorithm: a line-pair merger. For consecutive prose lines (a, b):
//   - line a ends with `]` (optional trailing whitespace),
//   - line b starts with `(URL)` where URL contains no whitespace and
//     no unbalanced parens, and is URL-ish (has `:`, `/`, `@`, or `#`),
//
// → merge into `a-ending+(URL)+b-trailing` and remove the split.
//
// Catalog code: V8. Evidence: Apify RAG Web Browser issue
// "Markdown Links Split Across Multiple Lines Causing Formatting Issues"
// and OpenAI community thread "Markdown Formatting Issues with GPT-5".
func applySplitLink(lines []Line, _ Options) ([]Line, bool) {
	if len(lines) < 2 {
		return lines, false
	}

	var (
		out   []Line
		fired bool
		i     = 0
	)
	for i < len(lines) {
		// Only consider candidates inside prose. Blank lines are
		// allowed as the "split" so we accept either LineProse or
		// LineBlank for the second slot.
		if lines[i].Kind != LineProse {
			out = append(out, lines[i])
			i++
			continue
		}
		endMatch := splitLinkLineEnd.FindStringSubmatch(lines[i].Text)
		if endMatch == nil {
			out = append(out, lines[i])
			i++
			continue
		}
		// Look at the next non-blank line.
		j := i + 1
		for j < len(lines) && lines[j].Kind == LineBlank {
			j++
		}
		if j >= len(lines) || lines[j].Kind != LineProse {
			out = append(out, lines[i])
			i++
			continue
		}
		startMatch := splitLinkLineStart.FindStringSubmatch(lines[j].Text)
		if startMatch == nil || !looksURLish(startMatch[1]) {
			out = append(out, lines[i])
			i++
			continue
		}
		// Merge: take the first line's `[label]` ending, append
		// `(URL)` + any trailing content from the second line.
		merged := endMatch[1] + "(" + startMatch[1] + ")" + startMatch[2]
		out = append(out, Line{Text: merged, Kind: LineProse})
		fired = true
		// Skip past the second line (and any blank lines we
		// jumped over).
		i = j + 1
	}
	return out, fired
}

// looksURLish reports whether s contains at least one URL-indicating
// character (`:`, `/`, `@`, `#`). LLM links to docs/drive/email all
// satisfy at least one. Pure prose parentheticals like "(actually
// Baz)" don't and the guard rejects them.
func looksURLish(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ':', '/', '@', '#':
			return true
		}
	}
	return false
}
