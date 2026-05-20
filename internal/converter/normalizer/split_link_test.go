package normalizer

import "testing"

// TestApplySplitLink_TableCases covers V8 split-link repair across the
// observed LLM-emission variants plus the false-positive guard.
func TestApplySplitLink_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple LF split",
			in:   "[click here]\n(https://example.com)",
			want: "[click here](https://example.com)",
		},
		{
			name: "CRLF split",
			in:   "[click here]\r\n(https://example.com)",
			want: "[click here](https://example.com)",
		},
		{
			name: "leading whitespace on URL line",
			in:   "[click]\n   (https://example.com)",
			want: "[click](https://example.com)",
		},
		{
			name: "trailing whitespace on label line",
			in:   "[click]   \n(https://example.com)",
			want: "[click](https://example.com)",
		},
		{
			name: "paragraph break between bracket and parens",
			in:   "[click]\n\n(https://example.com)",
			want: "[click](https://example.com)",
		},
		{
			name: "inside list item",
			in:   "- [Project Folder]\n(https://drive.google.com/drive/folders/abc)",
			want: "- [Project Folder](https://drive.google.com/drive/folders/abc)",
		},
		{
			name: "preserves trailing content after URL",
			in:   "[click]\n(https://example.com) and more text",
			want: "[click](https://example.com) and more text",
		},
		{
			name: "mailto URL",
			in:   "[email me]\n(mailto:alice@example.com)",
			want: "[email me](mailto:alice@example.com)",
		},
		{
			name: "two split links one after another",
			in:   "- [first]\n(https://a.com/x)\n- [second]\n(https://b.com/y)",
			want: "- [first](https://a.com/x)\n- [second](https://b.com/y)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applySplitLink(lines, Options{})
			if !fired {
				t.Errorf("expected fired=true for %q", tc.in)
			}
			got := reassemble(out)
			if got != tc.want {
				t.Errorf("applySplitLink(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestApplySplitLink_FalsePositiveGuard pins inputs that look similar
// but must NOT be repaired. A `]` followed by `(` in prose is common
// in English; the URL-ish guard prevents collapsing them.
func TestApplySplitLink_FalsePositiveGuard(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{
			name: "prose parenthetical with no URL chars",
			in:   "see [Bar]\n(actually Baz)",
		},
		{
			name: "parenthetical with space inside",
			in:   "[label]\n(some text here)",
		},
		{
			name: "well-formed link unchanged",
			in:   "[a](https://b.com)",
		},
		{
			name: "single line no break",
			in:   "[a] (b)", // not a link per CommonMark either, but we don't touch it
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applySplitLink(lines, Options{})
			if fired {
				t.Errorf("guard failed: applySplitLink(%q) fired", tc.in)
			}
			got := reassemble(out)
			if got != tc.in {
				t.Errorf("applySplitLink(%q) mutated to %q", tc.in, got)
			}
		})
	}
}

// TestApplySplitLink_InsideFenceUnchanged confirms the repair skips
// code blocks. A literal `[a]\n(b)` inside a fenced code block must
// pass through unchanged so the rendered code stays accurate.
func TestApplySplitLink_InsideFenceUnchanged(t *testing.T) {
	in := "```\n[a]\n(https://example.com)\n```"
	lines := classify(in)
	out, fired := applySplitLink(lines, Options{})
	if fired {
		t.Errorf("fired inside fenced code block")
	}
	got := reassemble(out)
	if got != in {
		t.Errorf("mutated fenced code block:\n in: %q\nout: %q", in, got)
	}
}

// TestApplySplitLink_Idempotent runs the repair twice and confirms the
// second pass is a no-op.
func TestApplySplitLink_Idempotent(t *testing.T) {
	in := "- [Project Folder]\n(https://drive.google.com/drive/folders/abc)"
	lines := classify(in)
	once, _ := applySplitLink(lines, Options{})
	twice, fired := applySplitLink(classify(reassemble(once)), Options{})
	if fired {
		t.Errorf("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Errorf("not idempotent:\nonce:  %q\ntwice: %q",
			reassemble(once), reassemble(twice))
	}
}
