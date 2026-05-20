package normalizer

import (
	"regexp"
	"strings"
)

// Line is a single source line plus the parse state that line lives in.
// The orchestrator builds a []Line once per Normalize call; every
// pattern repair iterates the slice and consults LineKind / FenceLang
// before deciding whether to touch the bytes.
//
// We track state per-line (not per-byte) because every pattern in the
// catalog operates at line granularity or above. A byte-level tokenizer
// would be more accurate but adds cost without buying us anything.
type Line struct {
	// Text is the line content with the trailing newline stripped.
	Text string

	// Kind tells the repair functions which CommonMark construct this
	// line sits in. The orchestrator may have rewritten Text after
	// classification — Kind reflects the *original* classification so
	// repairs can decide based on the pre-edit context.
	Kind LineKind

	// FenceLang is the language tag from the opening fence line, only
	// populated for lines inside a fenced code block (Kind ==
	// LineFenceContent). Empty for non-fence lines.
	FenceLang string
}

// LineKind classifies a line's role in the source. The set is
// intentionally minimal — every pattern in the LLM-input recovery
// catalog needs to know only "is this line inside code?" and "is
// this line in a table?".
type LineKind int

const (
	// LineProse is the default — a line that is not inside any
	// special CommonMark block.
	LineProse LineKind = iota

	// LineFenceOpen marks the opening fence (``` or ~~~) of a fenced
	// code block. Repairs MUST NOT alter this line — changing the
	// fence count would corrupt the block boundary detection.
	LineFenceOpen

	// LineFenceContent marks a line inside a fenced code block.
	// Every repair must skip these lines unless it's explicitly
	// scoped to fence repair (V3, V4).
	LineFenceContent

	// LineFenceClose marks the closing fence.
	LineFenceClose

	// LineIndentedCode marks a 4-space-indented code block line
	// (CommonMark §4.4). Skipped by every prose-level repair.
	LineIndentedCode

	// LineTable marks a line inside a GFM table (header, delimiter,
	// or data row). Some repairs (R8 `<br>` replacement) behave
	// differently inside tables.
	LineTable

	// LineBlank marks a blank line.
	LineBlank
)

// fenceOpen matches an opening fenced code block line: 0–3 leading
// spaces, then 3+ backticks or tildes, then an optional info string.
// The capture groups are (indent)(fence-char-run)(info-string).
//
// CommonMark §4.5: "A fenced code block begins with a code fence,
// indented no more than three spaces."
var fenceOpen = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})(.*)$")

// tableDelimiter matches the GFM table separator row, e.g.
// `| :---: | ---: |` or `---|---`. Used to detect that the previous
// line was a table header and the current run is a table body.
var tableDelimiter = regexp.MustCompile(
	`^\s*\|?\s*:?-{3,}:?(\s*\|\s*:?-{3,}:?)*\s*\|?\s*$`,
)

// classify walks the source once and tags every line with its parse
// state. Single-pass, O(n). The returned slice does not include the
// trailing newline on any line; the orchestrator reattaches them via
// reassemble.
func classify(src string) []Line {
	raw := strings.Split(src, "\n")
	out := make([]Line, len(raw))

	var (
		inFence       bool
		fenceChar     byte // '`' or '~'
		fenceLen      int
		fenceLang     string
		inIndentCode  bool
		prevBlank     = true // before first line, treat as blank for indent-code start
		tableHeaderAt = -1
	)

	for i, line := range raw {
		out[i].Text = line

		// Fence state takes priority over everything else.
		if inFence {
			if m := fenceOpen.FindStringSubmatch(line); m != nil &&
				len(m[2]) >= fenceLen && m[2][0] == fenceChar &&
				strings.TrimSpace(m[3]) == "" {
				// Matching closer — same char family, ≥ same length,
				// no info string.
				out[i].Kind = LineFenceClose
				inFence = false
				fenceChar = 0
				fenceLen = 0
				fenceLang = ""
			} else {
				out[i].Kind = LineFenceContent
				out[i].FenceLang = fenceLang
			}
			prevBlank = false
			continue
		}

		// Fence opener check (only outside a fence).
		if m := fenceOpen.FindStringSubmatch(line); m != nil {
			out[i].Kind = LineFenceOpen
			inFence = true
			fenceChar = m[2][0]
			fenceLen = len(m[2])
			fenceLang = strings.TrimSpace(m[3])
			out[i].FenceLang = fenceLang
			prevBlank = false
			continue
		}

		// Blank line.
		if strings.TrimSpace(line) == "" {
			out[i].Kind = LineBlank
			inIndentCode = false
			tableHeaderAt = -1
			prevBlank = true
			continue
		}

		// Indented code: only starts after a blank line. List-item
		// continuation indent is its own pattern; we conservatively
		// don't classify those as indented code here (lists handle
		// their own continuation).
		if prevBlank && hasIndentedCodePrefix(line) {
			out[i].Kind = LineIndentedCode
			inIndentCode = true
			prevBlank = false
			continue
		}
		if inIndentCode && hasIndentedCodePrefix(line) {
			out[i].Kind = LineIndentedCode
			prevBlank = false
			continue
		}
		inIndentCode = false

		// Table detection: a delimiter row immediately following a
		// non-blank line opens a table that runs until the next blank
		// line.
		if tableDelimiter.MatchString(line) && i > 0 && out[i-1].Kind == LineProse {
			out[i].Kind = LineTable
			// Backfill the header row as part of the table.
			out[i-1].Kind = LineTable
			tableHeaderAt = i - 1
			prevBlank = false
			continue
		}
		if tableHeaderAt >= 0 {
			out[i].Kind = LineTable
			prevBlank = false
			continue
		}

		out[i].Kind = LineProse
		prevBlank = false
	}

	return out
}

// hasIndentedCodePrefix reports whether the line begins with at least
// 4 spaces (or a single tab) of indentation.
func hasIndentedCodePrefix(line string) bool {
	if strings.HasPrefix(line, "\t") {
		return true
	}
	return strings.HasPrefix(line, "    ")
}

// reassemble joins a []Line back into a single string with newlines.
// Idempotent with classify: reassemble(classify(s)) == s.
func reassemble(lines []Line) string {
	parts := make([]string, len(lines))
	for i := range lines {
		parts[i] = lines[i].Text
	}
	return strings.Join(parts, "\n")
}
