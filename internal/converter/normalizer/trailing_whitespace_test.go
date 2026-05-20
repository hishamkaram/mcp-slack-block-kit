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
	// CommonMark §6.7: a hard line break is preceded by TWO OR MORE
	// trailing spaces. Our repair must NOT collapse any such suffix.
	cases := []struct {
		name string
		in   string
	}{
		{name: "2 spaces", in: "line one  \nline two"},
		{name: "3 spaces", in: "line one   \nline two"},
		{name: "4 spaces", in: "line one    \nline two"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyTrailingWhitespace(lines, Options{})
			if fired {
				t.Error("fired on hard-break line")
			}
			if reassemble(out) != tc.in {
				t.Errorf("hard break corrupted: %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

// TestApplyTrailingWhitespace_InsideFenceUnchanged pins the
// code-preserving invariant for C5. Trailing whitespace inside a
// fenced code block is part of the literal content per CommonMark
// §4.5 — stripping it would corrupt e.g. Python doctest fixtures
// where trailing space is semantically significant.
func TestApplyTrailingWhitespace_InsideFenceUnchanged(t *testing.T) {
	cases := []string{
		"```python\nprint('hi')   \n```",
		"```\nline1   \nline2  \n```",
		"```\n  indented_with_trailing   \n```",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			lines := classify(in)
			out, fired := applyTrailingWhitespace(lines, Options{})
			if fired {
				t.Errorf("fired inside fence on %q", in)
			}
			if got := reassemble(out); got != in {
				t.Errorf("mutated fenced content:\n in: %q\nout: %q", in, got)
			}
		})
	}
}

// TestApplyTrailingWhitespace_InsideIndentedCodeUnchanged covers
// the indented-code case (CommonMark §4.4).
func TestApplyTrailingWhitespace_InsideIndentedCodeUnchanged(t *testing.T) {
	in := "para\n\n    indented_code   \n    second_line  "
	lines := classify(in)
	out, fired := applyTrailingWhitespace(lines, Options{})
	if fired {
		t.Error("fired inside indented code")
	}
	if reassemble(out) != in {
		t.Errorf("indented code corrupted")
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
