package normalizer

import (
	"reflect"
	"testing"
)

// TestClassify_TableCases pins the line-classifier behavior every
// repair function depends on. False classifications would cause
// repairs to fire inside code (corrupting it) or skip valid prose.
func TestClassify_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []LineKind
	}{
		{
			name: "single prose line",
			in:   "hello",
			want: []LineKind{LineProse},
		},
		{
			name: "fenced code block",
			in:   "before\n```\ncode\n```\nafter",
			want: []LineKind{
				LineProse, LineFenceOpen, LineFenceContent,
				LineFenceClose, LineProse,
			},
		},
		{
			name: "fenced code block with language",
			in:   "```go\nfmt.Println(\"hi\")\n```",
			want: []LineKind{LineFenceOpen, LineFenceContent, LineFenceClose},
		},
		{
			name: "tilde fence",
			in:   "~~~\nx\n~~~",
			want: []LineKind{LineFenceOpen, LineFenceContent, LineFenceClose},
		},
		{
			name: "mixed fence chars do not close each other",
			in:   "```\nx\n~~~\ny\n```",
			want: []LineKind{
				LineFenceOpen, LineFenceContent, LineFenceContent,
				LineFenceContent, LineFenceClose,
			},
		},
		{
			name: "blank lines",
			in:   "a\n\nb",
			want: []LineKind{LineProse, LineBlank, LineProse},
		},
		{
			name: "indented code after blank",
			in:   "para\n\n    code\n    more",
			want: []LineKind{LineProse, LineBlank, LineIndentedCode, LineIndentedCode},
		},
		{
			name: "GFM table",
			in:   "| h1 | h2 |\n|----|----|\n| a  | b  |",
			want: []LineKind{LineTable, LineTable, LineTable},
		},
		{
			name: "table without leading pipes",
			in:   "h1 | h2\n---|---\na | b",
			want: []LineKind{LineTable, LineTable, LineTable},
		},
		{
			name: "table ends at blank line",
			in:   "| h |\n|---|\n| a |\n\nback to prose",
			want: []LineKind{LineTable, LineTable, LineTable, LineBlank, LineProse},
		},
		{
			name: "indented under 4 spaces stays prose",
			in:   "para\n\n   only 3 spaces",
			want: []LineKind{LineProse, LineBlank, LineProse},
		},
		{
			name: "tab counts as indented code",
			in:   "para\n\n\tcode",
			want: []LineKind{LineProse, LineBlank, LineIndentedCode},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			got := make([]LineKind, len(lines))
			for i := range lines {
				got[i] = lines[i].Kind
			}
			if !reflect.DeepEqual(tc.want, got) {
				t.Errorf("classify(%q) kind = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestClassify_FenceLangCaptured pins that the info-string from a
// fenced opener is propagated to every content line. Repairs that
// inspect language (e.g. V4) rely on this.
func TestClassify_FenceLangCaptured(t *testing.T) {
	lines := classify("```go\nfmt.Println(\"hi\")\n```")
	if lines[1].FenceLang != "go" {
		t.Errorf("FenceLang on content line = %q, want %q", lines[1].FenceLang, "go")
	}
}

// TestReassemble_Roundtrip pins the invariant the orchestrator relies
// on: reassemble(classify(s)) == s for any s.
func TestReassemble_Roundtrip(t *testing.T) {
	cases := []string{
		"",
		"hello",
		"a\nb\nc",
		"a\n\nb\n\nc\n",
		"```\ncode\n```\nprose",
		"trailing newline\n",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got := reassemble(classify(in))
			if got != in {
				t.Errorf("reassemble(classify(%q)) = %q, want %q", in, got, in)
			}
		})
	}
}
