package converter

import (
	"reflect"
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

// richTextListItems returns the plain-text content of every item in the
// first rich_text_list found in blocks, plus whether any list was found.
// Only text elements are concatenated, so an item whose markdown parsed
// into a bold/link element contributes the inner text (proving the inline
// parse ran) rather than the raw markdown.
func richTextListItems(blocks []slack.Block) ([]string, bool) {
	for _, b := range blocks {
		rt, ok := b.(*slack.RichTextBlock)
		if !ok {
			continue
		}
		for _, el := range rt.Elements {
			lst, ok := el.(*slack.RichTextList)
			if !ok {
				continue
			}
			items := make([]string, 0, len(lst.Elements))
			for _, it := range lst.Elements {
				s, ok := it.(*slack.RichTextSection)
				if !ok {
					continue
				}
				var sb strings.Builder
				for _, e := range s.Elements {
					if te, ok := e.(*slack.RichTextSectionTextElement); ok {
						sb.WriteString(te.Text)
					}
				}
				items = append(items, sb.String())
			}
			return items, true
		}
	}
	return nil, false
}

func newRichTextRenderer(t *testing.T) *Renderer {
	t.Helper()
	opts := DefaultOptions()
	opts.Mode = ModeRichText
	r, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestUnicodeBulletTransformer_ExoticGlyphBecomesList: a list marked with a
// glyph OUTSIDE the normalizer's curated set (★, U+2605) is rescued by the
// goldmark paragraph transformer into a real rich_text_list.
func TestUnicodeBulletTransformer_ExoticGlyphBecomesList(t *testing.T) {
	r := newRichTextRenderer(t)
	blocks, _, err := r.ConvertWithWarnings("★ first\n★ second")
	if err != nil {
		t.Fatal(err)
	}
	items, ok := richTextListItems(blocks)
	if !ok {
		t.Fatalf("expected a rich_text_list, got %v", blockTypes(blocks))
	}
	if want := []string{"first", "second"}; !reflect.DeepEqual(items, want) {
		t.Errorf("items = %v, want %v", items, want)
	}
}

// TestUnicodeBulletTransformer_LeadInPreserved: a non-bullet lead-in line is
// kept as its own block; only the bullet run becomes a list.
func TestUnicodeBulletTransformer_LeadInPreserved(t *testing.T) {
	r := newRichTextRenderer(t)
	blocks, _, err := r.ConvertWithWarnings("Here are the items:\n▶ alpha\n▶ beta")
	if err != nil {
		t.Fatal(err)
	}
	items, ok := richTextListItems(blocks)
	if !ok {
		t.Fatalf("expected a rich_text_list, got %v", blockTypes(blocks))
	}
	if want := []string{"alpha", "beta"}; !reflect.DeepEqual(items, want) {
		t.Errorf("items = %v, want %v", items, want)
	}
	// The lead-in must survive somewhere in the output.
	if !blocksContainText(blocks, "Here are the items:") {
		t.Errorf("lead-in line lost; blocks = %v", blockTypes(blocks))
	}
}

// TestUnicodeBulletTransformer_InlineContentParsed: markdown inside an item
// is inline-parsed (bold collapses to its inner text, not literal `**`).
func TestUnicodeBulletTransformer_InlineContentParsed(t *testing.T) {
	r := newRichTextRenderer(t)
	blocks, _, err := r.ConvertWithWarnings("★ **bold** tail\n★ plain")
	if err != nil {
		t.Fatal(err)
	}
	items, ok := richTextListItems(blocks)
	if !ok {
		t.Fatalf("expected a rich_text_list, got %v", blockTypes(blocks))
	}
	if items[0] != "bold tail" {
		t.Errorf("item0 = %q, want %q (inline parse should strip the ** markers)", items[0], "bold tail")
	}
}

// TestUnicodeBulletTransformer_SingleLineStaysProse: peer evidence is
// required — a lone symbol-led line is not turned into a list.
func TestUnicodeBulletTransformer_SingleLineStaysProse(t *testing.T) {
	r := newRichTextRenderer(t)
	blocks, _, err := r.ConvertWithWarnings("★ a lone decorative line")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := richTextListItems(blocks); ok {
		t.Errorf("single symbol line became a list; want prose. blocks = %v", blockTypes(blocks))
	}
}

// TestUnicodeBulletTransformer_DashAttributionNotAList: em/en dashes are
// category Pd (excluded), so attribution lines stay prose.
func TestUnicodeBulletTransformer_DashAttributionNotAList(t *testing.T) {
	r := newRichTextRenderer(t)
	blocks, _, err := r.ConvertWithWarnings("— Albert Einstein\n— Someone Else")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := richTextListItems(blocks); ok {
		t.Errorf("em-dash attribution became a list; want prose. blocks = %v", blockTypes(blocks))
	}
}

// TestUnicodeBulletTransformer_MarkerChangeMidRunDeclines: a run whose marker
// rune changes is ambiguous; the transformer declines (stays prose).
func TestUnicodeBulletTransformer_MarkerChangeMidRunDeclines(t *testing.T) {
	r := newRichTextRenderer(t)
	// ★ (U+2605) then ☆ (U+2606): both exotic, different runes.
	blocks, _, err := r.ConvertWithWarnings("★ a\n☆ b")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := richTextListItems(blocks); ok {
		t.Errorf("mixed-marker run became a list; want prose. blocks = %v", blockTypes(blocks))
	}
}

// TestUnicodeBulletTransformer_AutoModeEmitsDashList: in auto mode the
// rescued list is packaged into a markdown block as canonical `- ` items.
func TestUnicodeBulletTransformer_AutoModeEmitsDashList(t *testing.T) {
	r, err := New(DefaultOptions()) // auto
	if err != nil {
		t.Fatal(err)
	}
	blocks, _, err := r.ConvertWithWarnings("★ a\n★ b")
	if err != nil {
		t.Fatal(err)
	}
	mb, ok := blocks[0].(*slack.MarkdownBlock)
	if !ok {
		t.Fatalf("expected MarkdownBlock, got %T", blocks[0])
	}
	if !strings.Contains(mb.Text, "- a\n- b") {
		t.Errorf("markdown text = %q, want it to contain %q", mb.Text, "- a\n- b")
	}
}

// TestUnicodeBullet_ProductionRepro_NormalizerC11: the exact production case
// (curated • marker) is repaired by the normalizer, emits a C11 warning, and
// yields a list with one item per bullet line.
func TestUnicodeBullet_ProductionRepro_NormalizerC11(t *testing.T) {
	r := newRichTextRenderer(t)
	in := "Project setup complete:\n• HubSpot Deal: x\n• Drive Folder: y\n• Master Document: z"
	blocks, warnings, err := r.ConvertWithWarnings(in)
	if err != nil {
		t.Fatal(err)
	}
	items, ok := richTextListItems(blocks)
	if !ok {
		t.Fatalf("expected a rich_text_list, got %v", blockTypes(blocks))
	}
	if len(items) != 3 {
		t.Errorf("got %d list items, want 3: %v", len(items), items)
	}
	foundC11 := false
	for _, w := range warnings {
		if strings.HasPrefix(w, "normalized input") && strings.Contains(w, "C11") {
			foundC11 = true
		}
	}
	if !foundC11 {
		t.Errorf("expected a C11 normalization warning, got %v", warnings)
	}
}

// blocksContainText reports whether any text/section element in blocks
// contains want. Used to confirm a lead-in line survived the transform.
func blocksContainText(blocks []slack.Block, want string) bool {
	for _, b := range blocks {
		switch x := b.(type) {
		case *slack.SectionBlock:
			if x.Text != nil && strings.Contains(x.Text.Text, want) {
				return true
			}
		case *slack.RichTextBlock:
			for _, el := range x.Elements {
				if s, ok := el.(*slack.RichTextSection); ok {
					for _, e := range s.Elements {
						if te, ok := e.(*slack.RichTextSectionTextElement); ok && strings.Contains(te.Text, want) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
