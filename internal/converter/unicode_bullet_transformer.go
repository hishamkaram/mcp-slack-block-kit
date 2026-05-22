package converter

import (
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// unicodeBulletTransformer is a goldmark ParagraphTransformer that rescues
// lists an LLM emitted with a non-ASCII bullet marker (•, ◦, ▪, ★, ▶, →, …).
// goldmark recognizes only `-`, `*`, `+` as bullet markers, so such a "list"
// is parsed as a single paragraph and renders inline in Slack (every item on
// one line). This transformer detects a paragraph that is structurally a
// bullet list and rewrites it into a native *ast.List, so the rest of the
// converter (lists.go / markdown_block_emit.go) handles it exactly like a
// real list.
//
// It is the parser-level half of a two-layer fix. The pre-parse normalizer
// (internal/converter/normalizer, code C11) converts a curated set of common
// bullet glyphs to `-` before goldmark runs — those become native lists with
// full nesting/continuation support and surface a C11 warning. This
// transformer is the structural backstop: it catches any remaining bullet
// glyph (by Unicode category, not an allowlist) that reached goldmark as a
// paragraph.
//
// Timing: paragraph transformers run during block parsing, BEFORE inline
// parsing (see goldmark parser.Parse). Operating on the paragraph's raw line
// segments here means each rebuilt list item's text is inline-parsed normally
// in the next phase, so links/bold/emoji inside items work for free.
type unicodeBulletTransformer struct{}

// Transform implements parser.ParagraphTransformer.
func (unicodeBulletTransformer) Transform(p *ast.Paragraph, reader text.Reader, _ parser.Context) {
	parent := p.Parent()
	if parent == nil {
		return
	}
	lines := p.Lines()
	n := lines.Len()
	if n == 0 {
		return
	}
	source := reader.Source()

	// Per-line analysis: find the first bullet line, then require every
	// line from there to the end to be a bullet with the SAME marker rune.
	firstBullet := -1
	var markerRune rune
	contentOffset := make([]int, n)
	for i := 0; i < n; i++ {
		seg := lines.At(i)
		if seg.Padding != 0 {
			// Padded lines only occur in nested contexts where our
			// byte-offset math would be wrong; decline rather than risk
			// corrupting content.
			if firstBullet >= 0 {
				return
			}
			continue
		}
		r, off, ok := bulletMarker(source[seg.Start:seg.Stop])
		if !ok {
			if firstBullet >= 0 {
				// A non-bullet line interrupts the run — the paragraph is
				// mixed content; leave it untouched (safe).
				return
			}
			continue // still in the lead-in
		}
		if firstBullet < 0 {
			firstBullet = i
			markerRune = r
		} else if r != markerRune {
			return // marker changed mid-run — ambiguous, decline
		}
		contentOffset[i] = off
	}

	// Require peer evidence: at least two consecutive bullet lines. A lone
	// symbol-led line is too weak a signal for the broad (category-based)
	// detector and could be ordinary prose. The normalizer handles the
	// confident single-item curated-glyph case.
	if firstBullet < 0 || n-firstBullet < 2 {
		return
	}

	list := ast.NewList('-') // synthetic marker → IsOrdered()==false (bullet)
	list.IsTight = true
	for i := firstBullet; i < n; i++ {
		seg := lines.At(i)
		start := seg.Start + contentOffset[i]
		// Exclude a trailing newline so the item text matches a native
		// list item; left in, the inline parser would render it as a
		// trailing soft-break space ("first " instead of "first").
		stop := seg.Stop
		for stop > start && (source[stop-1] == '\n' || source[stop-1] == '\r') {
			stop--
		}
		content := text.NewSegment(start, stop)
		tb := ast.NewTextBlock()
		segs := text.NewSegments()
		segs.Append(content)
		tb.SetLines(segs)
		item := ast.NewListItem(len("- "))
		item.AppendChild(item, tb)
		list.AppendChild(list, item)
	}

	if firstBullet == 0 {
		// Whole paragraph is the list — replace it.
		parent.InsertBefore(parent, p, list)
		parent.RemoveChild(parent, p)
		return
	}
	// Keep the lead-in lines as the paragraph; the list follows it.
	lead := text.NewSegments()
	for i := 0; i < firstBullet; i++ {
		lead.Append(lines.At(i))
	}
	p.SetLines(lead)
	parent.InsertAfter(parent, p, list)
}

// bulletMarker reports whether a raw source line begins (after ≤3 spaces of
// indent) with a non-ASCII bullet glyph followed by horizontal whitespace.
// On success it returns the marker rune and the byte offset within the line
// where the item content starts.
//
// "Bullet glyph" is defined by Unicode general category — Other-Punctuation
// (Po), Other-Symbol (So), or Math-Symbol (Sm) — restricted to non-ASCII
// runes. This deliberately EXCLUDES dashes (category Pd: en/em-dash, used for
// attributions and ranges), quotation marks (Pi/Pf), and currency (Sc), which
// open prose lines too often to treat as list markers. ASCII `-`/`*`/`+` are
// already handled natively by goldmark.
func bulletMarker(raw []byte) (rune, int, bool) {
	indent := 0
	for indent < len(raw) && indent < 4 && raw[indent] == ' ' {
		indent++
	}
	if indent >= 4 || indent >= len(raw) {
		return 0, 0, false
	}
	r, size := utf8.DecodeRune(raw[indent:])
	if r == utf8.RuneError || r < utf8.RuneSelf || !isBulletGlyph(r) {
		return 0, 0, false
	}
	after := raw[indent+size:]
	ws := 0
	for ws < len(after) && (after[ws] == ' ' || after[ws] == '\t') {
		ws++
	}
	if ws == 0 {
		return 0, 0, false // marker must be followed by whitespace
	}
	return r, indent + size + ws, true
}

// isBulletGlyph reports whether r is a plausible bullet marker by Unicode
// category. See bulletMarker for the rationale behind the included/excluded
// categories.
func isBulletGlyph(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsNumber(r) {
		return false
	}
	return unicode.Is(unicode.Po, r) ||
		unicode.Is(unicode.So, r) ||
		unicode.Is(unicode.Sm, r)
}
