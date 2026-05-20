package normalizer

import "testing"

func TestApplyHTMLEntities_DisabledByDefault(t *testing.T) {
	in := "Tom &amp; Jerry"
	lines := classify(in)
	out, fired := applyHTMLEntities(lines, Options{})
	if fired {
		t.Error("V11 fired when DecodeHTMLEntities=false")
	}
	if reassemble(out) != in {
		t.Errorf("mutated when disabled: %q -> %q", in, reassemble(out))
	}
}

func TestApplyHTMLEntities_TableCases(t *testing.T) {
	opts := Options{DecodeHTMLEntities: true}
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "amp entity",
			in:   "Tom &amp; Jerry",
			want: "Tom & Jerry",
		},
		{
			name: "lt and gt",
			in:   "use &lt;tag&gt; syntax",
			want: "use <tag> syntax",
		},
		{
			name: "quot and apos",
			in:   "she said &quot;hi&apos;there&quot;",
			want: `she said "hi'there"`,
		},
		{
			name: "decimal numeric entity",
			in:   "&#65;&#66;&#67;",
			want: "ABC",
		},
		{
			name: "hex numeric entity",
			in:   "&#x41;&#X42;&#x43;",
			want: "ABC",
		},
		{
			name: "mixed named + numeric",
			in:   "&amp; then &#x21;",
			want: "& then !",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyHTMLEntities(lines, opts)
			if !fired {
				t.Errorf("expected fire on %q", tc.in)
			}
			if got := reassemble(out); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyHTMLEntities_NonWhitelistedUntouched(t *testing.T) {
	// Non-whitelisted named entities (nbsp, copy, mdash, …) must NOT
	// be decoded — silently decoding the full HTML-entity table is
	// too broad a security surface. They flow through to the
	// converter unchanged.
	opts := Options{DecodeHTMLEntities: true}
	cases := []string{
		"this is &nbsp; a space",
		"&copy; 2026",
		"&mdash; em dash",
		"&unknown;",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			lines := classify(in)
			out, fired := applyHTMLEntities(lines, opts)
			if fired {
				t.Errorf("decoded non-whitelisted entity: %q -> %q", in, reassemble(out))
			}
		})
	}
}

func TestApplyHTMLEntities_InsideFenceUnchanged(t *testing.T) {
	opts := Options{DecodeHTMLEntities: true}
	in := "```\nTom &amp; Jerry\n```"
	lines := classify(in)
	out, fired := applyHTMLEntities(lines, opts)
	if fired {
		t.Error("fired inside fenced code")
	}
	if reassemble(out) != in {
		t.Errorf("mutated fenced content")
	}
}

func TestApplyHTMLEntities_InsideInlineCodeUnchanged(t *testing.T) {
	opts := Options{DecodeHTMLEntities: true}
	in := "use `Tom &amp; Jerry` literally"
	lines := classify(in)
	out, fired := applyHTMLEntities(lines, opts)
	if fired {
		t.Errorf("fired inside inline code: %q -> %q", in, reassemble(out))
	}
	if reassemble(out) != in {
		t.Errorf("mutated inline-code content")
	}
}

func TestApplyHTMLEntities_BroadcastTokenRoundtripSafe(t *testing.T) {
	// The point of the broadcast-safety contract: a smuggled
	// `&lt;!channel&gt;` decodes here to `<!channel>`. The
	// downstream sanitizeBroadcasts pass re-escapes it. The
	// normalizer's responsibility is just to perform the decode
	// honestly — the safety property holds at the converter level
	// (covered by mentions_test.go in the parent package).
	opts := Options{DecodeHTMLEntities: true}
	in := "alert &lt;!channel&gt; please"
	lines := classify(in)
	out, fired := applyHTMLEntities(lines, opts)
	if !fired {
		t.Fatal("expected decode")
	}
	got := reassemble(out)
	want := "alert <!channel> please"
	if got != want {
		t.Errorf("decode = %q, want %q", got, want)
	}
}

// TestApplyHTMLEntities_ControlCharsRejected pins the V11 safety
// invariant: numeric entities for C0 controls, DEL, and UTF-16
// surrogate halves must NOT decode. Decoding them would embed
// line-changing bytes (LF/CR/TAB) into a Line's Text — violating
// the package's per-line invariant and the documented idempotence
// contract — or produce invalid UTF-8 (surrogates) — or inject NUL
// bytes that Slack rejects outright.
func TestApplyHTMLEntities_ControlCharsRejected(t *testing.T) {
	opts := Options{DecodeHTMLEntities: true}
	cases := []struct {
		name string
		in   string
	}{
		{name: "NUL (decimal)", in: "a&#0;b"},
		{name: "NUL (hex)", in: "a&#x00;b"},
		{name: "LF (decimal)", in: "a&#10;b"},
		{name: "LF (hex)", in: "a&#x0A;b"},
		{name: "CR (decimal)", in: "a&#13;b"},
		{name: "TAB (decimal)", in: "a&#9;b"},
		{name: "DEL", in: "a&#127;b"},
		{name: "C0 control (BEL)", in: "a&#7;b"},
		{name: "high surrogate", in: "a&#xD800;b"},
		{name: "low surrogate", in: "a&#xDFFF;b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := classify(tc.in)
			out, fired := applyHTMLEntities(lines, opts)
			if fired {
				t.Errorf("V11 decoded a rejected entity: %q -> %q", tc.in, reassemble(out))
			}
			if reassemble(out) != tc.in {
				t.Errorf("mutated input: %q -> %q", tc.in, reassemble(out))
			}
		})
	}
}

// TestApplyHTMLEntities_StructuralSmugglingBlocked confirms that
// even when the input would parse-as-paragraph-plus-blockquote if
// the LF were decoded, V11 leaves the source untouched so goldmark
// sees the original single line.
func TestApplyHTMLEntities_StructuralSmugglingBlocked(t *testing.T) {
	in := "paragraph&#10;> quote"
	out, fired := Normalize(in, Options{DecodeHTMLEntities: true})
	if fired != nil {
		t.Errorf("V11 fired on control-char smuggling attempt: %v", fired)
	}
	if out != in {
		t.Errorf("Normalize mutated structural-smuggle input: %q -> %q", in, out)
	}
}

func TestApplyHTMLEntities_Idempotent(t *testing.T) {
	opts := Options{DecodeHTMLEntities: true}
	in := "Tom &amp; Jerry &#x21;"
	lines := classify(in)
	once, _ := applyHTMLEntities(lines, opts)
	twice, fired := applyHTMLEntities(classify(reassemble(once)), opts)
	if fired {
		t.Error("second pass fired")
	}
	if reassemble(once) != reassemble(twice) {
		t.Error("not idempotent")
	}
}
