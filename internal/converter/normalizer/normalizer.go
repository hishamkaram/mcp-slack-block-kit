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

	// Pipeline order. Each step is a no-op until its concrete repair
	// lands in a later commit; the comment block above the call cites
	// the catalog code so reviewers can trace the design to the
	// evidence.
	//
	// Structural-space repairs first (they shift byte offsets):
	//   C5/C9 trailing whitespace
	//   V5    ATX header without space
	//   C3    bullet without space
	//   C4    numbered list without space
	//
	// URL hygiene:
	//   V7    smart quotes / em-dashes in URLs
	//
	// Inline structure:
	//   V8    split-link `[label]\n(url)` → `[label](url)`
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

	_ = opts // referenced by future repairs

	if !fired.any() {
		// No repair fired and the slice was untouched. Returning the
		// original string avoids the round-trip cost of reassemble.
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
