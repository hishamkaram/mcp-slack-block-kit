package normalizer

import "regexp"

// bulletNoSpace matches a line that starts with a bullet marker
// followed immediately by a non-space, non-digit character. The
// non-digit guard is critical: lines like `-1 means undefined` should
// pass through (the `-` is the unary minus, not a bullet marker).
//
// Captures: (1) leading indent (≤3 spaces), (2) the bullet marker
// (`-`, `*`, or `+`), (3) the first content character.
var bulletNoSpace = regexp.MustCompile(`^( {0,3})([-*+])([^\s\d])`)

// applyBulletNoSpace inserts a space after a bullet marker when
// missing. Fires only when (a) the line is prose context, (b) the
// next non-space character after the marker is neither whitespace
// nor a digit, and (c) at least one adjacent line also opens with
// the same marker (which prevents single-line false positives like
// `-update` in the middle of a paragraph).
//
// Catalog code: C3. Evidence: CommonMark §5.2 list-marker rules
// require a space after the marker; markdownlint flags this.
func applyBulletNoSpace(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		if lines[i].Kind != LineProse {
			continue
		}
		m := bulletNoSpace.FindStringSubmatch(lines[i].Text)
		if m == nil {
			continue
		}
		if !hasAdjacentBulletPeer(lines, i, m[1], m[2][0]) {
			continue
		}
		lines[i].Text = m[1] + m[2] + " " + lines[i].Text[len(m[1])+len(m[2]):]
		fired = true
	}
	return lines, fired
}

// hasAdjacentBulletPeer checks whether the line at index i has a
// sibling list-item line above or below using the same marker and
// indent. Adjacency stops at blank lines (which end a list anyway).
// Requires the peer to be already well-formed: `<indent><marker><sp>...`
// or also missing the space (`<indent><marker><non-space>`).
func hasAdjacentBulletPeer(lines []Line, i int, indent string, marker byte) bool {
	check := func(j int) bool {
		if j < 0 || j >= len(lines) {
			return false
		}
		if lines[j].Kind != LineProse {
			return false
		}
		text := lines[j].Text
		if len(text) < len(indent)+1 {
			return false
		}
		if text[:len(indent)] != indent {
			return false
		}
		if text[len(indent)] != marker {
			return false
		}
		// Either followed by space (well-formed peer) or another
		// non-space, non-digit (peer also needs the same repair).
		if len(text) > len(indent)+1 {
			next := text[len(indent)+1]
			if next == ' ' || next == '\t' {
				return true
			}
			if next < '0' || next > '9' {
				return true
			}
		}
		return false
	}
	return check(i-1) || check(i+1)
}
