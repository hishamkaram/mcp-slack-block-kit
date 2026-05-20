package normalizer

import "strings"

// applyAsteriskBalance fixes mismatched asterisk emphasis like
// `**italic*` (2-open, 1-close) or `*bold**` (1-open, 2-close) by
// normalizing both sides to the SMALLER count. Operates per
// paragraph: for each adjacent open/close emphasis-marker pair, if
// the counts differ we collapse both to the minimum so the inner
// span renders as the correct emphasis level.
//
// This is the catalog's trickiest pattern — false positives corrupt
// otherwise-valid prose with deliberate asterisk asymmetry. Default
// off; the caller opts in via Options.RepairMismatchedEmphasis.
//
// Algorithm (per paragraph):
//  1. Find adjacent emphasis markers separated by word content.
//  2. If their lengths differ AND both are entirely asterisks AND
//     the inner content has no other asterisks, collapse both to
//     the smaller count.
//  3. Iterate until no more changes — but cap at 4 passes per
//     paragraph to keep complexity bounded.
//
// Catalog code: V6. Evidence: Unmarkdown "Why Do Asterisks Appear
// When I Paste from ChatGPT"; CleanTextTools "Why Does ChatGPT Add
// Asterisks"; remend npm.
//
// Skips fenced and indented code blocks.
func applyAsteriskBalance(lines []Line, opts Options) ([]Line, bool) {
	if !opts.RepairMismatchedEmphasis {
		return lines, false
	}
	var (
		fired     bool
		paragraph []int
	)
	flush := func() {
		if len(paragraph) == 0 {
			return
		}
		var b strings.Builder
		for i, idx := range paragraph {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(lines[idx].Text)
		}
		original := b.String()
		rewritten := balanceAsteriskPairs(original)
		if rewritten != original {
			parts := strings.Split(rewritten, "\n")
			for i, idx := range paragraph {
				if i < len(parts) {
					lines[idx].Text = parts[i]
				}
			}
			fired = true
		}
		paragraph = nil
	}
	for i := range lines {
		if lines[i].Kind == LineProse {
			paragraph = append(paragraph, i)
			continue
		}
		flush()
	}
	flush()
	return lines, fired
}

// balanceAsteriskPairs collapses asterisk-run pairs in the paragraph
// to their smaller count, but only for paragraphs whose runs cleanly
// partition into adjacent open/close pairs with no other asterisks in
// the inner content. Conservative: leaves runs ≥3 alone (bold-italic)
// and bails entirely when the paragraph has an odd number of runs.
//
// Concretely, a paragraph like `**italic*` (2 runs) collapses, and
// `**a* and *b**` (4 runs, both pairs cleanly separated by non-
// asterisk content) collapses both pairs. A paragraph like
// `**a*b*c**` (4 runs but the "inner" of the first pair contains
// a separator-like single asterisk that's actually one of the four
// runs) is rejected because each pair's inner span isn't cleanly
// asterisk-free relative to the OUTER pair's claim on the same
// region — we can't safely disambiguate without a full CommonMark
// emphasis parse, so we leave it alone.
func balanceAsteriskPairs(s string) string {
	runs := asteriskRuns(s)
	if len(runs) < 2 || len(runs)%2 != 0 {
		return s
	}
	// Reject if any run is ≥3 (bold-italic territory).
	for _, r := range runs {
		if r.length >= 3 {
			return s
		}
	}
	// Reject if the outermost (first..last) span has any asterisks
	// inside the gaps between the inner pair boundaries — that's
	// the `**a*b*c**` shape where the inner content visually has
	// literal asterisks the author wants preserved.
	if hasInnerAsteriskOverlap(s, runs) {
		return s
	}
	out := []byte(s)
	delta := 0
	for i := 0; i+1 < len(runs); i += 2 {
		open := runs[i]
		closeR := runs[i+1]
		if open.length == closeR.length {
			continue
		}
		innerStart := open.end + delta
		innerEnd := closeR.start + delta
		if innerStart >= innerEnd {
			continue
		}
		inner := string(out[innerStart:innerEnd])
		if strings.Contains(inner, "*") || strings.TrimSpace(inner) == "" {
			continue
		}
		target := open.length
		if closeR.length < target {
			target = closeR.length
		}
		newOpen := strings.Repeat("*", target)
		openStartAdj := open.start + delta
		openEndAdj := open.end + delta
		out = append(out[:openStartAdj], append([]byte(newOpen), out[openEndAdj:]...)...)
		delta += target - open.length
		newClose := strings.Repeat("*", target)
		closeStartAdj := closeR.start + delta
		closeEndAdj := closeR.end + delta
		out = append(out[:closeStartAdj], append([]byte(newClose), out[closeEndAdj:]...)...)
		delta += target - closeR.length
	}
	return string(out)
}

// hasInnerAsteriskOverlap reports whether the content between
// consecutive pairs contains non-whitespace text that suggests the
// runs aren't cleanly partitioned. Specifically: if pair (2k, 2k+1)
// is followed by pair (2k+2, 2k+3) with NO whitespace between
// runs[2k+1] and runs[2k+2], the author likely meant one span with
// inner asterisks (e.g. `**a*b*c**`). Reject the whole paragraph in
// that case.
func hasInnerAsteriskOverlap(s string, runs []asteriskRun) bool {
	for i := 1; i+1 < len(runs); i += 2 {
		between := s[runs[i].end:runs[i+1].start]
		if !strings.ContainsAny(between, " \t") {
			return true
		}
	}
	return false
}
