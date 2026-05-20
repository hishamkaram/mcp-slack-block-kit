package normalizer

import "testing"

func TestApplyBulletNoSpace_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "two-item dash list",
			in:   "-first\n-second",
			want: "- first\n- second",
		},
		{
			name: "asterisk markers",
			in:   "*one\n*two\n*three",
			want: "* one\n* two\n* three",
		},
		{
			name: "plus markers",
			in:   "+a\n+b",
			want: "+ a\n+ b",
		},
		{
			name: "indented list",
			in:   "  -nested\n  -nested2",
			want: "  - nested\n  - nested2",
		},
		{
			name: "mixed well-formed and malformed",
			in:   "- first\n-second\n- third",
			want: "- first\n- second\n- third",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyBulletNoSpace(lines, Options{})
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

func TestApplyBulletNoSpace_FalsePositiveGuard(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{name: "unary minus number", in: "-1 means undefined\n-2 is also valid"},
		{name: "negative range", in: "-5 to -10"},
		{name: "single line no peer", in: "-loner"},
		{name: "already well-formed", in: "- a\n- b"},
		{name: "code line with dash", in: "    -code dash"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyBulletNoSpace(lines, Options{})
			if fired {
				t.Errorf("guard failed: fired on %q -> %q", tc.in, reassemble(out))
			}
			if reassemble(out) != tc.in {
				t.Errorf("mutated %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

func TestApplyBulletNoSpace_InsideFenceUnchanged(t *testing.T) {
	in := "```\n-not a bullet\n-still not\n```"
	lines := classify(in)
	out, fired := applyBulletNoSpace(lines, Options{})
	if fired {
		t.Error("fired inside fence")
	}
	if reassemble(out) != in {
		t.Errorf("mutated %q", in)
	}
}

func TestApplyBulletNoSpace_Idempotent(t *testing.T) {
	in := "-x\n-y\n-z"
	lines := classify(in)
	once, _ := applyBulletNoSpace(lines, Options{})
	twice, fired := applyBulletNoSpace(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Errorf("not idempotent")
	}
}
