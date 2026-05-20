package normalizer

// applyTildeInWord escapes a single `~` flanking word characters
// (`20~25` → `20\~25`) so downstream parsers don't treat the stray
// tilde as a broken GFM strikethrough opener. Skips fenced and
// indented code blocks. Skips lines already containing the doubled
// `~~` strikethrough form (preserves authorial intent).
//
// Implementation: a single byte-level pass that finds all
// non-overlapping `<word>~<word>` triplets and inserts a backslash
// before the tilde. We hand-roll the scan rather than using
// regexp.ReplaceAllString because the regex engine emits
// non-overlapping matches; with overlapping word boundaries (e.g.
// `0~0~0`) it produces output that requires another pass — breaking
// idempotence. The hand walker advances by 2 bytes after each match
// so the trailing word char is reused as the leading char of the
// next candidate match, exactly matching what "fix every stray
// tilde" should do.
//
// Catalog code: C1. Evidence: Streamdown 2.5 changelog
// (`singleTilde` option); Claude Code issue #19251 "Markdown
// renderer treats single tilde as strikethrough".
func applyTildeInWord(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		switch lines[i].Kind {
		case LineFenceOpen, LineFenceContent, LineFenceClose, LineIndentedCode:
			continue
		}
		// Skip if the line already has `~~` — author meant strikethrough.
		if hasDoubleTilde(lines[i].Text) {
			continue
		}
		original := lines[i].Text
		rewritten := escapeStrayTildes(original)
		if rewritten != original {
			lines[i].Text = rewritten
			fired = true
		}
	}
	return lines, fired
}

// escapeStrayTildes walks s and inserts a `\` before every `~` that
// is flanked on both sides by an ASCII word character (letter or
// digit) AND is not already preceded by a backslash. The walker
// advances byte-by-byte so overlapping triplets (`0~0~0`) all get
// caught in one pass — making the function idempotent.
func escapeStrayTildes(s string) string {
	if len(s) < 3 {
		return s
	}
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '~' && i > 0 && i+1 < len(s) &&
			isWordByte(s[i-1]) && isWordByte(s[i+1]) {
			if out == nil {
				out = make([]byte, 0, len(s)+8)
				out = append(out, s[:i]...)
			}
			// Don't double-escape an already-escaped tilde.
			if len(out) > 0 && out[len(out)-1] == '\\' {
				out = append(out, '~')
				continue
			}
			out = append(out, '\\', '~')
			continue
		}
		if out != nil {
			out = append(out, s[i])
		}
	}
	if out == nil {
		return s
	}
	return string(out)
}

// isWordByte reports whether b is an ASCII word character (letter or
// digit). Mirrors the regex character class `[A-Za-z0-9]` without
// allocating a regex engine.
func isWordByte(b byte) bool {
	switch {
	case b >= '0' && b <= '9':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= 'a' && b <= 'z':
		return true
	}
	return false
}

// hasDoubleTilde reports whether s contains `~~` (the GFM
// strikethrough delimiter). When present we assume the author used
// strikethrough intentionally and skip the single-tilde repair on
// this line.
func hasDoubleTilde(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '~' && s[i+1] == '~' {
			return true
		}
	}
	return false
}
