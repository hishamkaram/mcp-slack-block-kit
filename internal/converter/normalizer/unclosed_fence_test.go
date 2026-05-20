package normalizer

import (
	"strings"
	"testing"
)

func TestApplyUnclosedFence_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple backtick fence at EOF",
			in:   "```\nfmt.Println(\"hi\")",
			want: "```\nfmt.Println(\"hi\")\n```",
		},
		{
			name: "fence with language",
			in:   "```go\nfunc x()",
			want: "```go\nfunc x()\n```",
		},
		{
			name: "tilde fence",
			in:   "~~~\nx",
			want: "~~~\nx\n~~~",
		},
		{
			name: "indented fence (≤3 spaces)",
			in:   "  ```\n  x",
			want: "  ```\n  x\n  ```",
		},
		{
			name: "long fence (4+ backticks) gets matching closer",
			in:   "````\nfunc x()",
			want: "````\nfunc x()\n````",
		},
		{
			name: "unclosed fence followed by prose still consumed but repaired",
			in:   "```go\nfmt.Println(\"hi\")\n\nrest of prose",
			want: "```go\nfmt.Println(\"hi\")\n\nrest of prose\n```",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyUnclosedFence(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyUnclosedFence_ClosedFenceNoOp(t *testing.T) {
	cases := []string{
		"```\nx\n```",
		"```go\nfunc x()\n```\n",
		"~~~\nx\n~~~",
		"no fences at all here",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			lines := classify(in)
			out, fired := applyUnclosedFence(lines, Options{})
			if fired {
				t.Errorf("fired on already-closed input %q", in)
			}
			if reassemble(out) != in {
				t.Errorf("mutated %q -> %q", in, reassemble(out))
			}
		})
	}
}

func TestApplyUnclosedFence_Idempotent(t *testing.T) {
	in := "```go\nfunc x()"
	lines := classify(in)
	once, _ := applyUnclosedFence(lines, Options{})
	twice, fired := applyUnclosedFence(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired (idempotence violation)")
	}
	if reassemble(once) != reassemble(twice) {
		t.Errorf("not idempotent")
	}
}

func TestApplyUnclosedFence_DoesNotConsumeRestOfDocument(t *testing.T) {
	// End-to-end through Normalize: verify the prose after the
	// fenced block actually emerges as separate content rather than
	// being swallowed by the orphan fence.
	in := "```go\nfmt.Println(\"hi\")\n\nrest of prose continues here"
	out, fired := Normalize(in, Options{})
	if !contains(fired, "V3") {
		t.Errorf("expected V3 in fired codes, got %v", fired)
	}
	if !strings.Contains(out, "rest of prose continues here") {
		t.Errorf("prose was swallowed: %q", out)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
