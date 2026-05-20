package normalizer

import "testing"

// FuzzNormalize confirms the normalizer never panics, terminates,
// and stays length-bounded across arbitrary input. Seed corpus
// targets each catalog code's failure shape so the fuzzer starts
// near interesting inputs and explores outward.
//
// Run locally:  go test -run=^$ -fuzz=FuzzNormalize ./internal/converter/normalizer/
// Run in CI:    make fuzz  (30s budget; see Makefile)
func FuzzNormalize(f *testing.F) {
	seeds := []string{
		"",
		"plain prose",
		"# Heading\n- [a]\n(b)",              // V8
		"#Heading without space",             // V5
		"-item without space",                // C3
		"1.no space",                         // C4
		"[link](https://x.com/v2–migration)", // V7
		"trailing whitespace   \n",           // C5/C9
		"20~25°C",                            // C1
		"H1 | H2\n---|---\na | b",            // C6 borderless table
		"text<br>line two",                   // R8
		"&amp; &lt; &gt;",                    // V11 (opt-in)
		"**unclosed",                         // V1
		"`open inline",                       // V2
		"```go\nfunc x()",                    // V3
		"```go fmt.Println(\"hi\")```",       // V4
		"**bold*",                            // V6
		"<!channel>",                         // broadcast — must never be introduced
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, src string) {
		out1, _ := Normalize(src, Options{})
		out2, _ := Normalize(out1, Options{})
		if out1 != out2 {
			t.Fatalf("not idempotent:\n once: %q\ntwice: %q", out1, out2)
		}
		if len(out1) > 4*len(src)+1024 {
			t.Fatalf("unbounded growth: in=%d out=%d", len(src), len(out1))
		}

		// Same shape with the opt-in flags toggled — fuzzing must
		// cover those paths too.
		_, _ = Normalize(src, Options{DecodeHTMLEntities: true, RepairMismatchedEmphasis: true})
	})
}
