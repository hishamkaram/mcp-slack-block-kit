package normalizer

import "testing"

func TestApplyUnicodeBullet_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "bullet two-item list",
			in:   "• a\n• b",
			want: "- a\n- b",
		},
		{
			name: "white bullet single item (no peer needed)",
			in:   "◦ x",
			want: "- x",
		},
		{
			name: "small black square markers",
			in:   "▪ a\n▪ b\n▪ c",
			want: "- a\n- b\n- c",
		},
		{
			name: "black circle markers",
			in:   "● item one\n● item two",
			want: "- item one\n- item two",
		},
		{
			name: "black square single item (unambiguous, no peer)",
			in:   "■ done",
			want: "- done",
		},
		{
			name: "indented bullets keep indent",
			in:   "  • nested\n  • nested2",
			want: "  - nested\n  - nested2",
		},
		{
			name: "lead-in line stays, bullets converted (production repro)",
			in:   "Project setup complete:\n• HubSpot Deal: x\n• Drive Folder: y\n• Master Document: z",
			want: "Project setup complete:\n- HubSpot Deal: x\n- Drive Folder: y\n- Master Document: z",
		},
		{
			name: "tab after marker",
			in:   "•\titem",
			want: "- item",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyUnicodeBullet(lines, Options{})
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

func TestApplyUnicodeBullet_FalsePositiveGuard(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{name: "bullet glyph mid-line", in: "see • here for details"},
		{name: "middle dot mid-line", in: "A · B is a product"},
		{name: "glyph inside a word", in: "ca•fe latte"},
		{name: "already dash list", in: "- a\n- b"},
		{name: "asterisk markers untouched", in: "* a\n* b"},
		{name: "no whitespace after glyph", in: "•word stuck to marker"},
		{name: "ascii hyphen not in set", in: "- plain dash"},
		// Ambiguous markers are left to the peer-requiring transformer,
		// not converted unconditionally by the normalizer.
		{name: "arrow marker left to transformer", in: "→ See the docs for details"},
		{name: "middle dot left to transformer", in: "· a lone interpunct line"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyUnicodeBullet(lines, Options{})
			if fired {
				t.Errorf("guard failed: fired on %q -> %q", tc.in, reassemble(out))
			}
			if reassemble(out) != tc.in {
				t.Errorf("mutated %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

func TestApplyUnicodeBullet_InsideFenceUnchanged(t *testing.T) {
	in := "```\n• not a bullet\n• still not\n```"
	lines := classify(in)
	out, fired := applyUnicodeBullet(lines, Options{})
	if fired {
		t.Error("fired inside fence")
	}
	if reassemble(out) != in {
		t.Errorf("mutated %q", in)
	}
}

func TestApplyUnicodeBullet_Idempotent(t *testing.T) {
	in := "• x\n• y\n• z"
	once, _ := applyUnicodeBullet(classify(in), Options{})
	twice, fired := applyUnicodeBullet(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Errorf("not idempotent: %q vs %q", reassemble(once), reassemble(twice))
	}
}
