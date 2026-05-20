package normalizer

import "testing"

func TestApplyFenceLangNoNewline_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "go on one line",
			in:   "```go fmt.Println(\"hi\")```",
			want: "```go\nfmt.Println(\"hi\")\n```",
		},
		{
			name: "python on one line",
			in:   "```python print(\"hi\")```",
			want: "```python\nprint(\"hi\")\n```",
		},
		{
			name: "json on one line",
			in:   "```json {\"a\": 1}```",
			want: "```json\n{\"a\": 1}\n```",
		},
		{
			name: "uppercase language normalized",
			in:   "```GO fmt.Println(\"hi\")```",
			want: "```GO\nfmt.Println(\"hi\")\n```",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyFenceLangNoNewline(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyFenceLangNoNewline_FalsePositiveGuard(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{
			name: "unknown language passes through",
			in:   "```hello world```",
		},
		{
			name: "well-formed multi-line",
			in:   "```go\nfunc x()\n```",
		},
		{
			name: "no fence",
			in:   "just some prose",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyFenceLangNoNewline(lines, Options{})
			if fired {
				t.Errorf("guard failed: fired on %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

func TestApplyFenceLangNoNewline_Idempotent(t *testing.T) {
	in := "```go fmt.Println(\"hi\")```"
	lines := classify(in)
	once, _ := applyFenceLangNoNewline(lines, Options{})
	twice, fired := applyFenceLangNoNewline(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
