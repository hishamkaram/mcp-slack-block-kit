package normalizer

import "regexp"

// unicodeBullet matches a prose line whose first non-indent character is a
// non-ASCII bullet glyph followed by horizontal whitespace. LLMs frequently
// emit these (pasted from rich text / Office, or trained on rendered output)
// instead of the CommonMark `-`/`*`/`+` markers. goldmark does not recognize
// them as list markers, so the whole block collapses into one paragraph and
// renders inline in Slack (the items run together on a single line).
//
// Captures: (1) leading indent (≤3 spaces), (2) the content after the marker.
// The glyph and the whitespace after it are replaced with the canonical
// `- ` marker.
//
// The set is restricted to UNAMBIGUOUS bullet glyphs (dedicated bullets +
// filled/hollow geometric shapes) — these effectively never open a prose
// line, so a single occurrence is safely a list item with no peer required.
// Ambiguous markers that often lead ordinary prose as a single line —
// arrows (→ ▶ ► ▸ ➤), the middle dot (·), etc. — are deliberately NOT here;
// they are handled by the goldmark paragraph transformer in
// internal/converter (Layer 1), which only converts them on peer evidence
// (≥2 consistent marker lines). See docs/llm-input-recovery.md, code C11.
var unicodeBullet = regexp.MustCompile(
	`^( {0,3})[` +
		`\x{2022}\x{2023}\x{2043}\x{204C}\x{204D}\x{2219}` + // • ‣ ⁃ ⁌ ⁍ ∙
		`\x{25E6}\x{25AA}\x{25AB}\x{25CF}\x{25CB}` + // ◦ ▪ ▫ ● ○
		`\x{25A0}\x{25A1}\x{25C6}\x{25C7}` + // ■ □ ◆ ◇
		`][ \t]+(.*)$`,
)

// applyUnicodeBullet rewrites line-leading non-ASCII bullet glyphs to the
// canonical CommonMark `- ` marker so goldmark parses a real list instead of a
// run-together paragraph.
//
// No peer guard is needed (unlike C3's `hasAdjacentBulletPeer`): a line that
// starts with a bullet glyph followed by a space is unambiguously a list item
// — prose effectively never opens a line that way, and the `^` anchor already
// excludes mid-line uses like "see • here" or "A · B". Skips code/table lines
// via LineKind.
//
// Output is always a well-formed `- ` marker (dash + single space), so a
// second pass cannot re-fire and C3 never re-touches the line — idempotent.
//
// Catalog code: C11.
func applyUnicodeBullet(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		if lines[i].Kind != LineProse {
			continue
		}
		m := unicodeBullet.FindStringSubmatch(lines[i].Text)
		if m == nil {
			continue
		}
		lines[i].Text = m[1] + "- " + m[2]
		fired = true
	}
	return lines, fired
}
