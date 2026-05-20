package normalizer

import "testing"

func TestApplyTrailingWhitespace_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single line trailing spaces",
			in:   "hello   ",
			want: "hello",
		},
		{
			name: "trailing tab",
			in:   "hello\t",
			want: "hello",
		},
		{
			name: "trailing mixed",
			in:   "hello \t \t",
			want: "hello",
		},
		{
			name: "list item trailing space",
			in:   "- a   \n- b",
			want: "- a\n- b",
		},
		{
			name: "table delimiter trailing space",
			in:   "| h |\n|---|   \n| a |",
			want: "| h |\n|---|\n| a |",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyTrailingWhitespace(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyTrailingWhitespace_HardBreakPreserved(t *testing.T) {
	// CommonMark §6.7: two trailing spaces = hard line break.
	// Our repair must NOT collapse them.
	in := "line one  \nline two"
	lines := classify(in)
	out, fired := applyTrailingWhitespace(lines, Options{})
	if fired {
		t.Error("fired on hard-break line")
	}
	if reassemble(out) != in {
		t.Errorf("hard break corrupted: %q -> %q", in, reassemble(out))
	}
}

func TestApplyTrailingWhitespace_NoOpOnClean(t *testing.T) {
	in := "no trailing\nwhitespace here\nat all"
	lines := classify(in)
	out, fired := applyTrailingWhitespace(lines, Options{})
	if fired {
		t.Error("fired on clean input")
	}
	if reassemble(out) != in {
		t.Errorf("mutated clean input")
	}
}

func TestApplyTrailingWhitespace_Idempotent(t *testing.T) {
	in := "a    \nb\t\nc  "
	lines := classify(in)
	once, _ := applyTrailingWhitespace(lines, Options{})
	twice, fired := applyTrailingWhitespace(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
