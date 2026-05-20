package normalizer

import (
	"regexp"
	"unicode/utf8"
)

// urlInLink matches the URL portion of a CommonMark inline link, an
// autolink, or an image. We rewrite Unicode look-alikes ONLY inside
// these spans so prose like "the spec — see here" keeps its em-dash.
//
// Captures (1) the opening delimiter, (2) the URL content, (3) the
// closing delimiter. The URL content excludes the closing delimiter
// character — image+link parens match `[^)]*`, autolinks match
// `[^>]*`.
var (
	urlInBrackets = regexp.MustCompile(`(\]\()([^)\n]*)(\))`)
	urlInAutolink = regexp.MustCompile(`(<)([a-zA-Z][a-zA-Z0-9+.\-]{1,31}:[^<>\s]*)(>)`)
)

// unicodeToASCII maps the lookalike characters LLMs habitually emit
// inside URL paths to their ASCII equivalents. Per Briskly's and
// Context-Link.ai's reporting, em-dashes / en-dashes / curly quotes
// in URLs are the dominant breakage mode after split-links — the
// resulting URL works visually but fails on click because the path
// changed.
var unicodeToASCII = map[rune]string{
	'–': "-",   // en-dash
	'—': "-",   // em-dash
	'‘': "'",   // left single quote
	'’': "'",   // right single quote (also apostrophe)
	'“': `"`,   // left double quote
	'”': `"`,   // right double quote
	'…': "...", // ellipsis (rare in URLs, but seen)
}

// applyURLUnicode rewrites Unicode look-alike characters inside URL
// content. Operates on every line regardless of Kind because the
// regexes only match URL-delimited spans — they cannot match in
// fenced code blocks (where there are no link constructs), and even
// if they did the rewrite is safe (ASCII-equivalent).
//
// Catalog code: V7. Evidence: Context-Link.ai "Claude Em-Dash
// Problem"; Briskly "How to Stop AI Em Dashes (Claude, ChatGPT,
// Gemini)" — em-dashes break URLs, filenames, CSV parsing, code.
func applyURLUnicode(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		// Skip code contexts: the regex couldn't match anyway, but
		// skipping is cheaper than running the regex.
		switch lines[i].Kind {
		case LineFenceOpen, LineFenceContent, LineFenceClose, LineIndentedCode:
			continue
		}
		original := lines[i].Text
		mask := inlineCodeMask(original)
		rewritten := rewriteOutsideMask(original, urlInBrackets, mask)
		rewritten = rewriteOutsideMask(rewritten, urlInAutolink, inlineCodeMask(rewritten))
		if rewritten != original {
			lines[i].Text = rewritten
			fired = true
		}
	}
	return lines, fired
}

// rewriteOutsideMask runs re against s and applies the URL-Unicode
// rewrite to every match whose start position is NOT inside a
// CommonMark inline code span (per the inline-code mask). Matches
// inside code spans are left untouched — CommonMark §6.1 specifies
// code-span content as literal, so even an ASCII-equivalent
// substitution is a content change there.
func rewriteOutsideMask(s string, re *regexp.Regexp, mask []bool) string {
	matches := re.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return s
	}
	out := make([]byte, 0, len(s))
	cursor := 0
	for _, m := range matches {
		fullStart, fullEnd := m[0], m[1]
		if fullStart < len(mask) && mask[fullStart] {
			continue
		}
		// Submatches: m[2:4] = group 1, m[4:6] = group 2, m[6:8] = group 3.
		g1 := s[m[2]:m[3]]
		g2 := s[m[4]:m[5]]
		g3 := s[m[6]:m[7]]
		out = append(out, s[cursor:fullStart]...)
		out = append(out, g1...)
		out = append(out, replaceUnicode(g2)...)
		out = append(out, g3...)
		cursor = fullEnd
	}
	out = append(out, s[cursor:]...)
	return string(out)
}

// replaceUnicode walks s and replaces every rune that has an entry in
// unicodeToASCII. Allocates only when at least one replacement is
// needed.
func replaceUnicode(s string) string {
	hasAny := false
	for _, r := range s {
		if _, ok := unicodeToASCII[r]; ok {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return s
	}
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if repl, ok := unicodeToASCII[r]; ok {
			out = append(out, repl...)
			continue
		}
		out = utf8.AppendRune(out, r)
	}
	return string(out)
}
