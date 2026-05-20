package normalizer

import "testing"

func TestApplyURLUnicode_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "en-dash in markdown link URL",
			in:   "[docs](https://example.com/v2–migration)",
			want: "[docs](https://example.com/v2-migration)",
		},
		{
			name: "em-dash in markdown link URL",
			in:   "[docs](https://example.com/v2—migration)",
			want: "[docs](https://example.com/v2-migration)",
		},
		{
			name: "curly apostrophe in URL",
			in:   "[user](https://example.com/it’s-fine)",
			want: "[user](https://example.com/it's-fine)",
		},
		{
			name: "curly double quote in URL query",
			in:   `[search](https://example.com/q?s=“quoted”)`,
			want: `[search](https://example.com/q?s="quoted")`,
		},
		{
			name: "autolink em-dash",
			in:   "<https://example.com/foo—bar>",
			want: "<https://example.com/foo-bar>",
		},
		{
			name: "multiple replacements in one URL",
			in:   "[x](https://x.com/a–b—c'd)",
			want: "[x](https://x.com/a-b-c'd)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyURLUnicode(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyURLUnicode_FalsePositiveGuard(t *testing.T) {
	// Em-dashes in prose must NOT be rewritten — that's a legitimate
	// typographic choice; rewriting it would corrupt the author's
	// content.
	cases := []struct {
		name string
		in   string
	}{
		{name: "em-dash in prose", in: "see the spec — it's clear"},
		{name: "curly quotes in prose", in: `she said "yes"`},
		{name: "well-formed ASCII URL", in: "[x](https://x.com/a-b)"},
		{name: "em-dash in code span", in: "`use --flag — see docs`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyURLUnicode(lines, Options{})
			if fired {
				t.Errorf("guard failed: fired on %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

func TestApplyURLUnicode_InsideFenceUnchanged(t *testing.T) {
	in := "```\n[a](https://x.com/v2–migration)\n```"
	lines := classify(in)
	out, fired := applyURLUnicode(lines, Options{})
	if fired {
		t.Error("fired inside fence")
	}
	if reassemble(out) != in {
		t.Errorf("mutated fenced block")
	}
}

func TestApplyURLUnicode_Idempotent(t *testing.T) {
	in := "[a](https://x.com/v2–v3 — final)"
	lines := classify(in)
	once, _ := applyURLUnicode(lines, Options{})
	twice, fired := applyURLUnicode(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
