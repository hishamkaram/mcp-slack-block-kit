// Package normalizer repairs malformed markdown that LLMs commonly emit
// before goldmark parses it. Each repair targets a single pattern
// documented in docs/llm-input-recovery.md with cited evidence (parser
// issues, blog posts, production-grade repair libraries like
// vercel/streamdown + remend).
//
// The pipeline is fence-aware: every prose-level repair skips content
// inside fenced code blocks (``` or ~~~), indented code blocks, and —
// where relevant — table cells. The fence-state walker (fence_state.go)
// classifies every line once and tags repairs with the context they
// need.
//
// Safety invariants pinned by the property tests:
//   - Idempotent: Normalize(Normalize(s)) == Normalize(s).
//   - Length-bounded: len(out) ≤ 4 × len(in) + 1024 for any input.
//   - Code-preserving: bytes inside a well-formed fenced code block are
//     unchanged.
//   - Broadcast-safe: never introduces literal <!channel> / <!here> /
//     <!everyone> that wasn't in the input. (The entity-decode pass —
//     the only repair that could possibly produce a broadcast token —
//     uses a whitelist and re-escape pipeline.)
//
// The pipeline order matters. Whitespace and structural-space repairs
// run first (they shift byte positions), then URL hygiene, then
// split-link repair, then escape adjustments, finally the paragraph-
// level balance repairs for unclosed constructs (because earlier edits
// can change closing-counter math).
package normalizer

// Normalize repairs LLM-emitted markdown malformations and returns the
// repaired source together with the codes of every repair that fired
// (e.g. "V8", "C3"). The caller — typically converter.Renderer — can
// surface the codes verbatim to its own warnings channel so end users
// know which repairs the library applied.
//
// The returned slice is deduplicated and stable across runs; the order
// is the pipeline order, not the input order.
//
// Pure-string in/out. Idempotent on already-well-formed inputs (the
// returned []string is empty when no repair fires). O(n) on input
// length. Safe to call concurrently.
//
// This release ships an empty pipeline — the entry point exists so
// downstream wiring can land first. Subsequent commits add concrete
// repairs one at a time.
func Normalize(src string, opts Options) (string, []string) {
	if src == "" {
		return "", nil
	}

	lines := classify(src)
	fired := newFiredSet()

	// Pipeline order. Catalog codes (V*, C*, R*) refer to entries in
	// docs/llm-input-recovery.md. Order matters:
	//   - Structural rewrites that change line count run FIRST, so
	//     later line-local repairs see the final line layout.
	//   - Paragraph-level balance for unclosed constructs runs LAST,
	//     so earlier edits don't perturb closing counts.
	//
	// Structural rewrites that change line count run first.
	// br_tag can split a single prose line into multiple lines, so it
	// runs before any line-pair-merging repair.
	if l, ok := applyBRTag(lines, opts); ok { // R8
		lines = l
		fired.add("R8")
	}
	// Split-link merge collapses `[label]\n(url)` pairs. MUST precede
	// the list-marker repairs (C3, C4) — without it, the post-merge
	// line would become a sibling list-item for the next `-` line and
	// the C3 peer check would re-fire on a second Normalize pass,
	// breaking idempotence.
	if l, ok := applySplitLink(lines, opts); ok { // V8
		lines = l
		fired.add("V8")
	}
	// Whitespace hygiene next: trailing whitespace can throw off
	// every line-pattern repair below (regex anchors, peer checks).
	if l, ok := applyTrailingWhitespace(lines, opts); ok { // C5/C9
		lines = l
		fired.add("C5")
	}
	// Line-local space repairs.
	if l, ok := applyATXHeaderSpace(lines, opts); ok { // V5
		lines = l
		fired.add("V5")
	}
	if l, ok := applyBulletNoSpace(lines, opts); ok { // C3
		lines = l
		fired.add("C3")
	}
	if l, ok := applyNumberedNoSpace(lines, opts); ok { // C4
		lines = l
		fired.add("C4")
	}
	// URL hygiene and inline structure.
	if l, ok := applyURLUnicode(lines, opts); ok { // V7
		lines = l
		fired.add("V7")
	}
	if l, ok := applyTildeInWord(lines, opts); ok { // C1
		lines = l
		fired.add("C1")
	}
	if l, ok := applyBorderlessTable(lines, opts); ok { // C6
		lines = l
		fired.add("C6")
	}
	// Paragraph-level balance for unclosed constructs. MUST run last
	// because earlier line-pattern repairs (V8 merge, R8 split, C5
	// strip) change line boundaries and content lengths that the
	// balance counters consume. V4 runs before V3 because V4 may
	// produce a properly-fenced block out of a one-line lump, which
	// V3 would otherwise misclassify as an unclosed fence.
	if l, ok := applyFenceLangNoNewline(lines, opts); ok { // V4
		lines = l
		fired.add("V4")
	}
	if l, ok := applyUnclosedFence(lines, opts); ok { // V3
		lines = l
		fired.add("V3")
	}
	// V6 runs BEFORE V1: collapsing mismatched pairs to the smaller
	// count can turn an unmatched **opener into a matched *italic*,
	// removing work for V1. Gated by Options.RepairMismatchedEmphasis.
	if l, ok := applyAsteriskBalance(lines, opts); ok { // V6
		lines = l
		fired.add("V6")
	}
	// V1 runs BEFORE V2 in pipeline order: an emphasis appender that
	// turns a line ending in `` ` `` into one ending in `*` would
	// otherwise re-arm V2 on the next pass, breaking idempotence.
	// (Pinned by FuzzNormalize against inputs like `*0000000` `.)
	if l, ok := applyUnclosedEmphasis(lines, opts); ok { // V1
		lines = l
		fired.add("V1")
	}
	if l, ok := applyUnclosedInlineCode(lines, opts); ok { // V2
		lines = l
		fired.add("V2")
	}
	//
	// Future pipeline stages (placeholder list; concrete repairs land
	// in subsequent commits):
	//
	// URL hygiene:
	//   V7    smart quotes / em-dashes in URLs
	//
	// Inline structure:
	//   C5/C9 trailing whitespace
	//   C1    single-tilde-in-word escape
	//   C6    borderless table edge-pipe insertion
	//   R8    <br> → \n outside table cells
	//
	// Optional, opt-in:
	//   V11   HTML entity decode (opts.DecodeHTMLEntities)
	//   V6    mismatched-asterisks repair (opts.RepairMismatchedEmphasis)
	//
	// Paragraph-level balance (must run last; earlier edits change
	// closing counts):
	//   V2    unclosed inline code
	//   V3    unclosed fenced code
	//   V4    fence-with-language-no-newline
	//   V1    unclosed emphasis

	if !fired.any() {
		// No repair fired. Returning the original string preserves
		// any trailing bytes that classify+reassemble would normalize
		// (e.g. a missing trailing newline).
		return src, nil
	}
	return reassemble(lines), fired.codes()
}

// firedSet is an ordered, deduplicated set of catalog codes. Insertion
// order is preserved so the returned slice matches the pipeline order.
type firedSet struct {
	seen  map[string]struct{}
	order []string
}

func newFiredSet() *firedSet {
	return &firedSet{seen: make(map[string]struct{}, 4)}
}

// add records that the named repair fired. Idempotent: a code added
// twice appears once in the returned slice.
func (f *firedSet) add(code string) {
	if _, ok := f.seen[code]; ok {
		return
	}
	f.seen[code] = struct{}{}
	f.order = append(f.order, code)
}

// any reports whether at least one repair has fired.
func (f *firedSet) any() bool { return len(f.order) > 0 }

// codes returns the deduplicated list of fired codes in pipeline
// order. Returns nil (not an empty slice) when nothing fired, matching
// the public Normalize contract.
func (f *firedSet) codes() []string {
	if len(f.order) == 0 {
		return nil
	}
	out := make([]string, len(f.order))
	copy(out, f.order)
	return out
}
