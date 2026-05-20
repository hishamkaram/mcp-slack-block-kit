package normalizer

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// namedEntity matches the whitelist of named HTML entities: the five
// XML predefined entities (&amp; &lt; &gt; &quot; &apos;). Other
// named entities (e.g. &nbsp;, &copy;) are deliberately NOT in the
// set — silently decoding arbitrary HTML entities is a much larger
// security surface than the LLM-output recovery this catalog
// addresses.
var namedEntity = regexp.MustCompile(`&(amp|lt|gt|quot|apos);`)

// numericEntity matches numeric character references: decimal
// (`&#NN;`) and hex (`&#xNN;` / `&#XNN;`). The capture is the
// digits, base-prefixed via the (?i:x)? group.
var numericEntity = regexp.MustCompile(`&#([xX]?)([0-9a-fA-F]+);`)

// namedEntityChars maps the named entities to their decoded character.
var namedEntityChars = map[string]string{
	"amp":  "&",
	"lt":   "<",
	"gt":   ">",
	"quot": `"`,
	"apos": "'",
}

// applyHTMLEntities decodes the whitelisted named entities plus all
// numeric character references in plain-text prose. Skips fenced
// and indented code blocks (their content is literal CommonMark).
// Skips inline code spans (CommonMark §6.1).
//
// Catalog code: V11. Opt-in via Options.DecodeHTMLEntities.
//
// Broadcast-safety contract: after decode, the resulting characters
// flow through the converter's sanitizeBroadcasts pipeline (in
// internal/converter/mentions.go), which entity-escapes `<` / `>` /
// `&` again unless Options.AllowBroadcasts is true. So a payload of
// `&lt;!channel&gt;` decodes here to `<!channel>` and then re-escapes
// to `&lt;!channel&gt;` at the rich_text emission stage — the
// broadcast token cannot survive the round-trip and reach Slack
// unless the caller has already opted out of broadcast safety
// entirely. This makes V11 safe to enable.
func applyHTMLEntities(lines []Line, opts Options) ([]Line, bool) {
	if !opts.DecodeHTMLEntities {
		return lines, false
	}
	var fired bool
	for i := range lines {
		switch lines[i].Kind {
		case LineFenceOpen, LineFenceContent, LineFenceClose, LineIndentedCode:
			continue
		}
		original := lines[i].Text
		rewritten, didFire := decodeEntities(original)
		if didFire {
			lines[i].Text = rewritten
			fired = true
		}
	}
	return lines, fired
}

// decodeEntities decodes whitelisted entities in s, skipping any
// occurrence inside a backtick inline code span (per
// inlineCodeMask).
func decodeEntities(s string) (string, bool) {
	if !strings.ContainsRune(s, '&') {
		return s, false
	}
	mask := inlineCodeMask(s)
	var (
		fired   bool
		out     []byte
		cursor  int
		matches [][]int
	)
	// Collect named-entity matches.
	matches = namedEntity.FindAllStringSubmatchIndex(s, -1)
	for _, m := range matches {
		fullStart, fullEnd := m[0], m[1]
		if fullStart < len(mask) && mask[fullStart] {
			continue
		}
		name := s[m[2]:m[3]]
		out = append(out, s[cursor:fullStart]...)
		out = append(out, namedEntityChars[name]...)
		cursor = fullEnd
		fired = true
	}
	// Then numeric-entity matches in the partially-rewritten string.
	// Build the named-rewrite intermediate first so positions
	// stay consistent.
	intermediate := s
	if fired {
		out = append(out, s[cursor:]...)
		intermediate = string(out)
		out = out[:0]
		cursor = 0
		mask = inlineCodeMask(intermediate)
	}
	numMatches := numericEntity.FindAllStringSubmatchIndex(intermediate, -1)
	for _, m := range numMatches {
		fullStart, fullEnd := m[0], m[1]
		if fullStart < len(mask) && mask[fullStart] {
			continue
		}
		base := 10
		if intermediate[m[2]:m[3]] != "" {
			base = 16
		}
		digits := intermediate[m[4]:m[5]]
		n, err := strconv.ParseInt(digits, base, 32)
		if err != nil || n < 0 || n > 0x10FFFF {
			continue
		}
		out = append(out, intermediate[cursor:fullStart]...)
		var buf [4]byte
		nbytes := utf8.EncodeRune(buf[:], rune(n))
		out = append(out, buf[:nbytes]...)
		cursor = fullEnd
		fired = true
	}
	if !fired {
		return s, false
	}
	out = append(out, intermediate[cursor:]...)
	return string(out), true
}
