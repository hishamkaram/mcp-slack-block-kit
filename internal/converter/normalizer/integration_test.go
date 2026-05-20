package normalizer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestNormalize_ScreenshotInput_RepairsAllExpectedPatterns exercises
// the full pipeline against the bot output reproduced from the user's
// production Slack screenshot. The raw input mixes four
// catalog-confirmed malformations on the same payload — exactly the
// kind of compound case that demonstrates the pipeline's correctness.
//
// Expected repairs (codes appear once each, in pipeline order):
//   - V5 (##✅ / ###HubSpot etc. — header without space)
//   - C3 (-**Deal Name:** — bullet without space)
//   - V8 (- [View Deal]\n(https://...) — split link inside list item)
//
// C4 (numbered list no space) is not present in this input.
//
// After normalization the source must:
//   - render every `### Section` header with a space after the hashes,
//   - render every list item with `- ` between marker and content,
//   - put every Google Docs link on a single line with the URL inline.
func TestNormalize_ScreenshotInput_RepairsAllExpectedPatterns(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "screenshot_hubspot_summary.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	out, fired := Normalize(string(raw), Options{})

	// Repair codes — order is pipeline order, not input order.
	wantCodes := []string{"V5", "C3", "V8"}
	gotCodes := append([]string(nil), fired...)
	sort.Strings(wantCodes)
	sort.Strings(gotCodes)
	if !equalStringSlices(wantCodes, gotCodes) {
		t.Errorf("repair codes = %v, want %v", fired, []string{"V5", "C3", "V8"})
	}

	// Headers fixed.
	for _, frag := range []string{
		"## ✅ Project Setup Complete",
		"### HubSpot Deal",
		"### 🟢 Google Drive",
		"### 🔵 Line Item",
	} {
		if !strings.Contains(out, frag) {
			t.Errorf("missing header repair fragment: %q", frag)
		}
	}

	// Bullets fixed.
	for _, frag := range []string{
		"- **Deal Name:**",
		"- **Owner:**",
		"- **Product:**",
	} {
		if !strings.Contains(out, frag) {
			t.Errorf("missing bullet repair fragment: %q", frag)
		}
	}

	// Split links collapsed onto one line.
	for _, frag := range []string{
		"[View Deal](https://app.hubspot.com/contacts/6727898/record/0-3/60403877913)",
		"[Project Folder](https://drive.google.com/drive/folders/1GfYbOAuc5QJL14obsYRDoBzQut6T2WPK)",
		"[Project Master Document](https://docs.google.com/document/d/1xXgAyAvL_TCG-UFZjLWOndD9C1tbtJUBjyR_5jqnqzsmAso/edit?usp=drivesdk)",
		"[Proposal Deck](https://docs.google.com/presentation/d/17DUhwZZUYc2PnKKUhrlypnwueL9VQo0N0hk6cykmkew/edit?usp=drivesdk)",
		"[View Line Item](https://app.hubspot.com/line-items/6727898/deal/60403877913)",
	} {
		if !strings.Contains(out, frag) {
			t.Errorf("missing collapsed link: %q", frag)
		}
	}

	// Idempotence — a second pass must produce the same output and
	// fire nothing.
	out2, fired2 := Normalize(out, Options{})
	if out2 != out {
		t.Errorf("second-pass mutated output")
	}
	if len(fired2) != 0 {
		t.Errorf("second-pass fired codes %v, want none", fired2)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
