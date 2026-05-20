package normalizer

import "testing"

func TestApplyUnclosedEmphasis_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "trailing unclosed bold",
			in:   "Here is **bold text without closing",
			want: "Here is **bold text without closing**",
		},
		{
			name: "trailing unclosed italic",
			in:   "Here is *italic text",
			want: "Here is *italic text*",
		},
		{
			name: "multi-line paragraph unclosed bold",
			in:   "First line\nthen **second line opens bold",
			want: "First line\nthen **second line opens bold**",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyUnclosedEmphasis(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyUnclosedEmphasis_BalancedNoOp(t *testing.T) {
	cases := []string{
		"plain prose",
		"**balanced bold**",
		"*balanced italic*",
		"mix of **a** and *b* in one line",
		// Triple asterisk bold-italic is left to V6.
		"***triple***",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			lines := classify(in)
			out, fired := applyUnclosedEmphasis(lines, Options{})
			if fired {
				t.Errorf("fired on balanced %q -> %q", in, reassemble(out))
			}
		})
	}
}

func TestApplyUnclosedEmphasis_InsideFenceUnchanged(t *testing.T) {
	in := "```\n**unclosed inside fence\n```"
	lines := classify(in)
	out, fired := applyUnclosedEmphasis(lines, Options{})
	if fired {
		t.Error("fired inside fence")
	}
	if reassemble(out) != in {
		t.Error("mutated fenced content")
	}
}

func TestApplyUnclosedEmphasis_Idempotent(t *testing.T) {
	in := "**unclosed bold"
	lines := classify(in)
	once, _ := applyUnclosedEmphasis(lines, Options{})
	twice, fired := applyUnclosedEmphasis(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
