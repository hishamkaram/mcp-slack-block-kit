package normalizer

import "regexp"

// numberedNoSpace matches a line that starts with an ordered-list
// marker (1–9 digits + `.` or `)`) immediately followed by a non-space
// character. CommonMark §5.2 caps the number at 9 digits per the spec.
//
// Captures: (1) leading indent, (2) the digit run, (3) the marker char
// (`.` or `)`), (4) the first content character.
var numberedNoSpace = regexp.MustCompile(`^( {0,3})(\d{1,9})([.)])([^\s])`)

// applyNumberedNoSpace inserts a space after an ordered-list marker
// when missing. Fires only when the line is prose context AND an
// adjacent line opens with the same marker shape — the same
// peer-presence guard the bullet repair uses to avoid false positives
// on numeric labels like `1.5GB`.
//
// Catalog code: C4. Evidence: CommonMark §5.2; markdownlint flags
// missing-space patterns generically.
func applyNumberedNoSpace(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		if lines[i].Kind != LineProse {
			continue
		}
		m := numberedNoSpace.FindStringSubmatch(lines[i].Text)
		if m == nil {
			continue
		}
		if !hasAdjacentNumberedPeer(lines, i, m[1], m[3][0]) {
			continue
		}
		head := m[1] + m[2] + m[3]
		lines[i].Text = head + " " + lines[i].Text[len(head):]
		fired = true
	}
	return lines, fired
}

// hasAdjacentNumberedPeer checks whether the line at index i has a
// sibling ordered-list-item line above or below using the same indent
// and marker character.
func hasAdjacentNumberedPeer(lines []Line, i int, indent string, marker byte) bool {
	check := func(j int) bool {
		if j < 0 || j >= len(lines) || lines[j].Kind != LineProse {
			return false
		}
		text := lines[j].Text
		if len(text) < len(indent)+2 {
			return false
		}
		if text[:len(indent)] != indent {
			return false
		}
		// Walk the digit run.
		k := len(indent)
		digitCount := 0
		for k < len(text) && text[k] >= '0' && text[k] <= '9' && digitCount < 9 {
			k++
			digitCount++
		}
		if digitCount == 0 || k >= len(text) || text[k] != marker {
			return false
		}
		k++
		// Either followed by space (well-formed peer) or non-space
		// content (peer also needs repair).
		if k < len(text) {
			next := text[k]
			if next == ' ' || next == '\t' {
				return true
			}
			return true // peer also malformed
		}
		return false
	}
	return check(i-1) || check(i+1)
}
