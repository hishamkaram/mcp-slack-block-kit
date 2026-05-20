package converter

import (
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

// TestPreferRichText_ShortProseRoutesToRichText: with the opt-in flag
// set, auto-mode must route short prose through rich_text decomposition
// instead of emitting a single Slack `markdown` block.
func TestPreferRichText_ShortProseRoutesToRichText(t *testing.T) {
	opts := DefaultOptions()
	opts.PreferRichText = true
	r, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	blocks, _, err := r.ConvertWithWarnings("Just some short prose.")
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range blocks {
		if _, ok := b.(*slack.MarkdownBlock); ok {
			t.Errorf("PreferRichText=true emitted a MarkdownBlock; want rich_text decomposition")
		}
	}
}

// TestPreferRichText_DefaultFalse_PreservesExistingBehavior: with the
// flag at its default (false), auto-mode behavior is unchanged —
// short prose still emits a single MarkdownBlock.
func TestPreferRichText_DefaultFalse_PreservesExistingBehavior(t *testing.T) {
	r, err := New(DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	blocks, _, err := r.ConvertWithWarnings("Just some short prose.")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if _, ok := blocks[0].(*slack.MarkdownBlock); !ok {
		t.Errorf("default PreferRichText=false should emit MarkdownBlock; got %T", blocks[0])
	}
}

// TestSurfaceWarning_AutoMarkdownBlock_Emitted: every auto-mode pick
// of a markdown block now carries the fallback-surface advisory.
func TestSurfaceWarning_AutoMarkdownBlock_Emitted(t *testing.T) {
	r, err := New(DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	_, warnings, err := r.ConvertWithWarnings("Short prose.")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected at least 1 warning, got none")
	}
	found := false
	for _, w := range warnings {
		if w == MarkdownBlockFallbackSurfacesWarning {
			found = true
		}
	}
	if !found {
		t.Errorf("expected MarkdownBlockFallbackSurfacesWarning, got %v", warnings)
	}
}

// TestSurfaceWarning_PreferRichText_Suppressed: when the picker
// returns false because of PreferRichText, no surface advisory fires.
func TestSurfaceWarning_PreferRichText_Suppressed(t *testing.T) {
	opts := DefaultOptions()
	opts.PreferRichText = true
	r, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, warnings, err := r.ConvertWithWarnings("Short prose.")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warnings {
		if w == MarkdownBlockFallbackSurfacesWarning {
			t.Errorf("surface advisory leaked into PreferRichText path: %q", w)
		}
	}
}

// TestSurfaceWarning_ExplicitMarkdownBlockMode_NoWarning: callers
// opting into markdown_block explicitly get no advisory (preserves
// the existing contract).
func TestSurfaceWarning_ExplicitMarkdownBlockMode_NoWarning(t *testing.T) {
	opts := DefaultOptions()
	opts.Mode = ModeMarkdownBlock
	r, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, warnings, err := r.ConvertWithWarnings("Short prose.")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warnings {
		if w == MarkdownBlockFallbackSurfacesWarning {
			t.Errorf("explicit ModeMarkdownBlock should not emit surface advisory, got %q", w)
		}
	}
}

// TestNormalization_AppliedInline: a malformed split-link in the input
// produces a typed link element in the output AND a normalization
// warning is reported to the caller.
func TestNormalization_AppliedInline(t *testing.T) {
	// Use PreferRichText so the auto picker emits decomposed
	// rich_text blocks (and surfaces the link as a typed element)
	// rather than packaging the input into a single MarkdownBlock.
	opts := DefaultOptions()
	opts.PreferRichText = true
	r, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	blocks, warnings, err := r.ConvertWithWarnings(
		"- [Project Folder]\n(https://drive.example.com/folders/abc)",
	)
	if err != nil {
		t.Fatal(err)
	}
	foundNorm := false
	for _, w := range warnings {
		if strings.HasPrefix(w, "normalized input") &&
			strings.Contains(w, "V8") {
			foundNorm = true
		}
	}
	if !foundNorm {
		t.Errorf("expected V8 normalization warning, got %v", warnings)
	}
	// Confirm a typed link element survives — proves the normalizer's
	// repair flowed through to the converter's rich_text walker.
	if !hasLinkWithURL(blocks, "https://drive.example.com/folders/abc") {
		t.Errorf("expected typed link in output; got %v", blockTypes(blocks))
	}
}

// sectionHasLink reports whether s contains a link element whose
// URL equals want. Helper for hasLinkWithURL.
func sectionHasLink(s *slack.RichTextSection, want string) bool {
	for _, sub := range s.Elements {
		if link, ok := sub.(*slack.RichTextSectionLinkElement); ok && link.URL == want {
			return true
		}
	}
	return false
}

// hasLinkWithURL walks rich_text blocks (including list items)
// looking for a link element whose URL matches the want value.
func hasLinkWithURL(blocks []slack.Block, want string) bool {
	for _, b := range blocks {
		rt, ok := b.(*slack.RichTextBlock)
		if !ok {
			continue
		}
		for _, el := range rt.Elements {
			switch x := el.(type) {
			case *slack.RichTextSection:
				if sectionHasLink(x, want) {
					return true
				}
			case *slack.RichTextList:
				for _, item := range x.Elements {
					if s, ok := item.(*slack.RichTextSection); ok && sectionHasLink(s, want) {
						return true
					}
				}
			}
		}
	}
	return false
}
