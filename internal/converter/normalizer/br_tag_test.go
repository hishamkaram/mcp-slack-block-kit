package normalizer

import (
	"strings"
	"testing"
)

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

// TestApplyBRTag_InsideInlineCodeUnchanged pins the merged_bug_014
// regression: a `<br>` inside a backtick code span (CommonMark §6.1
// literal content) must NOT be split out.
func TestApplyBRTag_InsideInlineCodeUnchanged(t *testing.T) {
	cases := []string{
		"Use `<br>` for HTML breaks",
		"docs: `Foo<br>Bar` is literal",
		"mixed `<br>` and out-of-code <br> repaired only out",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			lines := classify(in)
			out, fired := applyBRTag(lines, Options{})
			// `Use \`<br>\` for HTML breaks` — the <br> inside
			// backticks must survive. The first two cases have no
			// out-of-code <br>, so no fire at all.
			if strings.Contains(in, "out-of-code") {
				if !fired {
					t.Error("expected fire on out-of-code <br>")
				}
				if !strings.Contains(reassemble(out), "`<br>`") {
					t.Errorf("inline-code <br> was modified: %q", reassemble(out))
				}
			} else {
				if fired {
					t.Errorf("fired on input with <br> only inside code: %q -> %q",
						in, reassemble(out))
				}
				if reassemble(out) != in {
					t.Errorf("mutated input: %q -> %q", in, reassemble(out))
				}
			}
		})
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
