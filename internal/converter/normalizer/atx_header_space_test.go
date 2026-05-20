package normalizer

import "testing"

func TestApplyATXHeaderSpace_TableCases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "h1 no space",
			in:   "#Migration Overview",
			want: "# Migration Overview",
		},
		{
			name: "h2 no space",
			in:   "##Section Two",
			want: "## Section Two",
		},
		{
			name: "h6 no space",
			in:   "######Deep",
			want: "###### Deep",
		},
		{
			name: "h1 with leading indent",
			in:   "  #Indented Heading",
			want: "  # Indented Heading",
		},
		{
			name: "multi-line document fixes only header lines",
			in:   "#Title\n\nbody text\n#Subtitle",
			want: "# Title\n\nbody text\n# Subtitle",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyATXHeaderSpace(lines, Options{})
			if !fired {
				t.Errorf("expected fired=true for %q", tc.in)
			}
			got := reassemble(out)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyATXHeaderSpace_FalsePositiveGuard(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{name: "already well-formed", in: "# Title"},
		{name: "hashtag mid-paragraph", in: "see #hashtag for more"},
		{name: "indent ≥4 spaces becomes code", in: "    #not a heading"},
		{name: "7+ hashes is not a heading", in: "#######tooMany"},
		{name: "all hashes line (could be closer)", in: "###"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyATXHeaderSpace(lines, Options{})
			if fired {
				t.Errorf("guard failed: fired on %q -> %q", tc.in, reassemble(out))
			}
			if reassemble(out) != tc.in {
				t.Errorf("mutated %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

func TestApplyATXHeaderSpace_InsideFenceUnchanged(t *testing.T) {
	in := "```\n#NotAHeader\n```"
	lines := classify(in)
	out, fired := applyATXHeaderSpace(lines, Options{})
	if fired {
		t.Errorf("fired inside fence")
	}
	if reassemble(out) != in {
		t.Errorf("mutated %q -> %q", in, reassemble(out))
	}
}

func TestApplyATXHeaderSpace_Idempotent(t *testing.T) {
	in := "#A\n##B\n###C"
	lines := classify(in)
	once, _ := applyATXHeaderSpace(lines, Options{})
	twice, fired := applyATXHeaderSpace(classify(reassemble(once)), Options{})
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Errorf("not idempotent")
	}
}
