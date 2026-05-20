package normalizer

import "testing"

func TestApplyBRTag_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple br",
			in:   "line one<br>line two",
			want: "line one\nline two",
		},
		{
			name: "self-closing br",
			in:   "line one<br/>line two",
			want: "line one\nline two",
		},
		{
			name: "br with space",
			in:   "line one<br />line two",
			want: "line one\nline two",
		},
		{
			name: "uppercase br",
			in:   "line one<BR>line two",
			want: "line one\nline two",
		},
		{
			name: "multiple br in one line",
			in:   "a<br>b<br>c",
			want: "a\nb\nc",
		},
		{
			name: "br in table cell becomes space",
			in:   "| h |\n|---|\n| a<br>b |",
			want: "| h |\n|---|\n| a b |",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyBRTag(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyBRTag_NoBRUnchanged(t *testing.T) {
	in := "plain text with no br tags here"
	lines := classify(in)
	out, fired := applyBRTag(lines, Options{})
	if fired {
		t.Error("fired on input without br tags")
	}
	if reassemble(out) != in {
		t.Error("mutated input")
	}
}

func TestApplyBRTag_InsideFenceUnchanged(t *testing.T) {
	in := "```html\nline<br>break\n```"
	lines := classify(in)
	out, fired := applyBRTag(lines, Options{})
	if fired {
		t.Error("fired inside fence")
	}
	if reassemble(out) != in {
		t.Error("mutated fenced content")
	}
}

func TestApplyBRTag_Idempotent(t *testing.T) {
	in := "a<br>b<br/>c"
	lines := classify(in)
	once, _ := applyBRTag(lines, Options{})
	twice, fired := applyBRTag(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
