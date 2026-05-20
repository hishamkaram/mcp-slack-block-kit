package normalizer

import "testing"

func TestApplyNumberedNoSpace_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "period marker",
			in:   "1.first\n2.second",
			want: "1. first\n2. second",
		},
		{
			name: "paren marker",
			in:   "1)first\n2)second",
			want: "1) first\n2) second",
		},
		{
			name: "multi-digit",
			in:   "10.ten\n11.eleven",
			want: "10. ten\n11. eleven",
		},
		{
			name: "mixed malformed and well-formed",
			in:   "1. a\n2.b\n3. c",
			want: "1. a\n2. b\n3. c",
		},
		{
			name: "indented",
			in:   "  1.a\n  2.b",
			want: "  1. a\n  2. b",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyNumberedNoSpace(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			got := reassemble(out)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyNumberedNoSpace_FalsePositiveGuard(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{name: "version with no peer", in: "1.5GB of RAM"},
		{name: "already well-formed", in: "1. a\n2. b"},
		{name: "single line no peer", in: "1.lonely"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyNumberedNoSpace(lines, Options{})
			if fired {
				t.Errorf("guard failed on %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

func TestApplyNumberedNoSpace_InsideFenceUnchanged(t *testing.T) {
	in := "```\n1.not a list\n2.still not\n```"
	lines := classify(in)
	out, fired := applyNumberedNoSpace(lines, Options{})
	if fired {
		t.Error("fired inside fence")
	}
	if reassemble(out) != in {
		t.Errorf("mutated %q", in)
	}
}

func TestApplyNumberedNoSpace_Idempotent(t *testing.T) {
	in := "1.a\n2.b\n3.c"
	lines := classify(in)
	once, _ := applyNumberedNoSpace(lines, Options{})
	twice, fired := applyNumberedNoSpace(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
