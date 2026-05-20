package normalizer

import "testing"

func TestApplyUnclosedInlineCode_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single open backtick",
			in:   "Use `myFunction to do X",
			want: "Use `myFunction to do X`",
		},
		{
			name: "two open then one unclosed",
			in:   "Use `foo` and `bar to mean baz",
			want: "Use `foo` and `bar to mean baz`",
		},
		{
			name: "open double backtick",
			in:   "show ``a`b without closing",
			want: "show ``a`b without closing``",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyUnclosedInlineCode(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyUnclosedInlineCode_BalancedNoOp(t *testing.T) {
	cases := []string{
		"no backticks here",
		"`balanced` example",
		"`a` `b` `c`",
		"``inner ` outer``",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			lines := classify(in)
			out, fired := applyUnclosedInlineCode(lines, Options{})
			if fired {
				t.Errorf("fired on balanced input %q", in)
			}
			if reassemble(out) != in {
				t.Errorf("mutated %q", in)
			}
		})
	}
}

func TestApplyUnclosedInlineCode_InsideFenceUnchanged(t *testing.T) {
	in := "```\n`unclosed inside fence\n```"
	lines := classify(in)
	out, fired := applyUnclosedInlineCode(lines, Options{})
	if fired {
		t.Error("fired inside fence")
	}
	if reassemble(out) != in {
		t.Error("mutated fenced content")
	}
}

func TestApplyUnclosedInlineCode_Idempotent(t *testing.T) {
	in := "use `foo to do bar"
	lines := classify(in)
	once, _ := applyUnclosedInlineCode(lines, Options{})
	twice, fired := applyUnclosedInlineCode(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
