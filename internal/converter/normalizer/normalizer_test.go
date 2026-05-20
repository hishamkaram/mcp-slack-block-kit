package normalizer

import (
	"reflect"
	"testing"
)

// TestNormalize_EmptyInput exercises the early-return path. Returning
// nil (not an empty slice) for the fired codes matches the contract
// the converter relies on for omitempty serialization.
func TestNormalize_EmptyInput(t *testing.T) {
	out, fired := Normalize("", Options{})
	if out != "" {
		t.Errorf("Normalize(empty) text = %q, want empty", out)
	}
	if fired != nil {
		t.Errorf("Normalize(empty) fired = %v, want nil", fired)
	}
}

// TestNormalize_NoRepairsNeeded_PassesThrough proves the skeleton's
// no-op behavior: well-formed input flows through unchanged with no
// codes reported. Subsequent commits add per-pattern tests that prove
// concrete repairs flip codes on; this test pins the negative case so
// regressions show up loudly.
func TestNormalize_NoRepairsNeeded_PassesThrough(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{name: "single line prose", in: "Hello, world."},
		{name: "paragraph with link", in: "See [docs](https://example.com) for more."},
		{name: "fenced code block", in: "```go\nfmt.Println(\"hi\")\n```"},
		{name: "list with bold", in: "- **a**\n- **b**\n"},
		{name: "header above paragraph", in: "# Heading\n\nbody text\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, fired := Normalize(tc.in, Options{})
			if out != tc.in {
				t.Errorf("Normalize(%q) text = %q, want %q", tc.in, out, tc.in)
			}
			if fired != nil {
				t.Errorf("fired = %v, want nil for already-well-formed input", fired)
			}
		})
	}
}

// TestNormalize_OptionsCarriedThrough proves the Options struct is
// threaded into the pipeline. The skeleton has no opt-in repairs yet,
// so the test asserts no panic and no code emission with each flag
// set; future commits extend this with positive assertions per flag.
func TestNormalize_OptionsCarriedThrough(t *testing.T) {
	cases := []Options{
		{},
		{DecodeHTMLEntities: true},
		{RepairMismatchedEmphasis: true},
		{DecodeHTMLEntities: true, RepairMismatchedEmphasis: true},
	}
	for _, opts := range cases {
		_, _ = Normalize("plain text", opts)
	}
}

// TestFiredSet_DeduplicatesPreservingOrder pins the firedSet contract
// the orchestrator relies on. Repairs can fire multiple times across
// the input (e.g. V8 split-link in two different lines); the returned
// codes slice must list each code once, in first-seen order.
func TestFiredSet_DeduplicatesPreservingOrder(t *testing.T) {
	s := newFiredSet()
	if s.any() {
		t.Fatal("new firedSet reports any()=true")
	}
	if got := s.codes(); got != nil {
		t.Fatalf("empty firedSet codes() = %v, want nil", got)
	}

	s.add("V8")
	s.add("C3")
	s.add("V8") // dedup
	s.add("V5")
	s.add("C3") // dedup

	want := []string{"V8", "C3", "V5"}
	if got := s.codes(); !reflect.DeepEqual(want, got) {
		t.Errorf("codes() = %v, want %v", got, want)
	}
	if !s.any() {
		t.Error("any() reports false after adds")
	}
}
