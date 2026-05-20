package normalizer

import "testing"

func TestApplyBorderlessTable_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "fully borderless",
			in:   "h1 | h2\n---|---\na | b",
			want: "| h1 | h2 |\n| ---|--- |\n| a | b |",
		},
		{
			name: "missing leading pipe only",
			in:   "h1 | h2 |\n---|---|\na | b |",
			want: "| h1 | h2 |\n| ---|---|\n| a | b |",
		},
		{
			name: "missing trailing pipe only",
			in:   "| h1 | h2\n|---|---\n| a | b",
			want: "| h1 | h2 |\n|---|--- |\n| a | b |",
		},
		{
			name: "fully borderless preserves indent",
			in:   "  h1 | h2\n  ---|---\n  a | b",
			want: "  | h1 | h2 |\n  | ---|--- |\n  | a | b |",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyBorderlessTable(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyBorderlessTable_FullyBorderedNoOp(t *testing.T) {
	in := "| h1 | h2 |\n|----|----|\n| a  | b  |"
	lines := classify(in)
	out, fired := applyBorderlessTable(lines, Options{})
	if fired {
		t.Error("fired on already-bordered table")
	}
	if reassemble(out) != in {
		t.Error("mutated well-formed table")
	}
}

func TestApplyBorderlessTable_NotATableUnchanged(t *testing.T) {
	in := "prose with | pipe character"
	lines := classify(in)
	out, fired := applyBorderlessTable(lines, Options{})
	if fired {
		t.Error("fired on prose")
	}
	if reassemble(out) != in {
		t.Error("mutated prose")
	}
}

func TestApplyBorderlessTable_Idempotent(t *testing.T) {
	in := "h1 | h2\n---|---\na | b"
	lines := classify(in)
	once, _ := applyBorderlessTable(lines, Options{})
	twice, fired := applyBorderlessTable(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
