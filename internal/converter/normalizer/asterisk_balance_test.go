package normalizer

import "testing"

func TestApplyAsteriskBalance_DisabledByDefault(t *testing.T) {
	in := "**italic*"
	lines := classify(in)
	out, fired := applyAsteriskBalance(lines, Options{})
	if fired {
		t.Error("fired when RepairMismatchedEmphasis=false")
	}
	if reassemble(out) != in {
		t.Errorf("mutated when disabled: %q -> %q", in, reassemble(out))
	}
}

func TestApplyAsteriskBalance_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "bold-open italic-close collapses to italic",
			in:   "**italic*",
			want: "*italic*",
		},
		{
			name: "italic-open bold-close collapses to italic",
			in:   "*italic**",
			want: "*italic*",
		},
		{
			name: "two pairs in one paragraph",
			in:   "**a* and *b**",
			want: "*a* and *b*",
		},
	}
	opts := Options{RepairMismatchedEmphasis: true}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyAsteriskBalance(lines, opts)
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyAsteriskBalance_FalsePositiveGuard(t *testing.T) {
	opts := Options{RepairMismatchedEmphasis: true}
	cases := []struct {
		name string
		in   string
	}{
		{name: "well-formed bold", in: "**bold**"},
		{name: "well-formed italic", in: "*italic*"},
		{name: "bold-italic triple unchanged", in: "***word***"},
		{name: "inner asterisk content unchanged", in: "**a*b*c**"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyAsteriskBalance(lines, opts)
			if fired {
				t.Errorf("guard failed: fired on %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

func TestApplyAsteriskBalance_InsideFenceUnchanged(t *testing.T) {
	opts := Options{RepairMismatchedEmphasis: true}
	in := "```\n**italic*\n```"
	lines := classify(in)
	out, fired := applyAsteriskBalance(lines, opts)
	if fired {
		t.Error("fired inside fence")
	}
	if reassemble(out) != in {
		t.Error("mutated fenced content")
	}
}

func TestApplyAsteriskBalance_Idempotent(t *testing.T) {
	opts := Options{RepairMismatchedEmphasis: true}
	in := "**italic*"
	lines := classify(in)
	once, _ := applyAsteriskBalance(lines, opts)
	twice, fired := applyAsteriskBalance(classify(reassemble(once)), opts)
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
