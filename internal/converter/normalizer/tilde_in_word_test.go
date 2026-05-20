package normalizer

import "testing"

func TestApplyTildeInWord_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "temperature range",
			in:   "20~25°C",
			want: `20\~25°C`,
		},
		{
			name: "digit-digit",
			in:   "1~2",
			want: `1\~2`,
		},
		{
			name: "letter-letter",
			in:   "a~b approximately",
			want: `a\~b approximately`,
		},
		{
			name: "multiple occurrences",
			in:   "0~1 and 9~10",
			want: `0\~1 and 9\~10`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyTildeInWord(lines, Options{})
			if !fired {
				t.Errorf("expected fired for %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyTildeInWord_FalsePositiveGuard(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{name: "GFM strikethrough preserved", in: "~~deleted~~ text"},
		{name: "GFM strike with single-tilde nearby", in: "~~old~~ and 20~25"},
		{name: "tilde flanked by space", in: "approximately ~ 25"},
		{name: "tilde at line start", in: "~bookmark"},
		{name: "tilde at line end", in: "trailing~"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyTildeInWord(lines, Options{})
			if fired {
				t.Errorf("guard failed: fired on %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

func TestApplyTildeInWord_InsideFenceUnchanged(t *testing.T) {
	in := "```\n20~25\n```"
	lines := classify(in)
	out, fired := applyTildeInWord(lines, Options{})
	if fired {
		t.Error("fired inside fence")
	}
	if reassemble(out) != in {
		t.Error("mutated fenced block")
	}
}

func TestApplyTildeInWord_Idempotent(t *testing.T) {
	in := "20~25 and 50~75"
	lines := classify(in)
	once, _ := applyTildeInWord(lines, Options{})
	twice, fired := applyTildeInWord(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
