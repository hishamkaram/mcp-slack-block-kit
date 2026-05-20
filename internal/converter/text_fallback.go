package converter

import (
	"regexp"
	"strings"

	"github.com/slack-go/slack"
)

// TextFallbackMaxChars is the upper bound on the plain-text fallback
// string DeriveTextFallback returns. 150 characters matches the
// typical width of Slack's push-notification preview on desktop and
// mobile; anything longer is truncated with a U+2026 ellipsis.
//
// Exported so callers that need a different cap can plan their own
// truncation or detect when the fallback was clipped.
const TextFallbackMaxChars = 150

// DeriveTextFallback walks the converted blocks and returns a short
// plain-text summary suitable for `chat.postMessage(text=)`. Slack
// uses that string verbatim on push notifications, search results,
// screen reader output, and the email digest — the surfaces where
// the `markdown` block's own rendering degrades.
//
// The summary strips both Slack mrkdwn and CommonMark formatting:
// `*bold*`, `_italic_`, `~strike~`, `\`code\“, `[label](url)`,
// `<URL|label>`, and `:emoji:` shortcodes all reduce to their
// textual content. The result is truncated to TextFallbackMaxChars
// runes with a U+2026 suffix when clipped.
//
// Returns an empty string when no block contributes usable text
// (e.g. a payload that's just dividers or actions). The caller can
// then decide to omit the `text:` parameter entirely or supply a
// static fallback.
func DeriveTextFallback(blocks []slack.Block) string {
	if len(blocks) == 0 {
		return ""
	}
	var collected []string
	for _, b := range blocks {
		text := extractBlockText(b)
		if text == "" {
			continue
		}
		collected = append(collected, text)
		// HeaderBlock dominates — once we have one, stop collecting
		// because notification previews lead with the title.
		if _, isHeader := b.(*slack.HeaderBlock); isHeader {
			break
		}
		if totalLen(collected) >= TextFallbackMaxChars {
			break
		}
	}
	if len(collected) == 0 {
		return ""
	}
	out := strings.Join(collected, " — ")
	out = collapseWhitespace(out)
	return clampRunes(out, TextFallbackMaxChars)
}

// extractBlockText pulls a plain-text summary out of a single block.
// Returns empty when the block has no textual surface (divider,
// actions, etc. when there's no other candidate yet).
func extractBlockText(b slack.Block) string {
	switch x := b.(type) {
	case *slack.HeaderBlock:
		if x.Text != nil {
			return x.Text.Text
		}
	case *slack.SectionBlock:
		if x.Text != nil {
			return stripFormatting(x.Text.Text)
		}
	case *slack.RichTextBlock:
		return richTextPlain(x)
	case *slack.MarkdownBlock:
		return stripFormatting(x.Text)
	case *slack.ContextBlock:
		var parts []string
		for _, el := range x.ContextElements.Elements {
			if t, ok := el.(*slack.TextBlockObject); ok && t != nil {
				parts = append(parts, stripFormatting(t.Text))
			}
		}
		return strings.Join(parts, " ")
	case *slack.ImageBlock:
		if x.AltText != "" {
			return "[image: " + x.AltText + "]"
		}
		return "[image]"
	case *slack.TableBlock:
		return "[table]"
	case *slack.DividerBlock, *slack.ActionBlock:
		return ""
	}
	return ""
}

// richTextPlain walks a rich_text block and returns its plain-text
// content. Drops styling, link URLs, and emoji metadata.
func richTextPlain(rt *slack.RichTextBlock) string {
	var b strings.Builder
	for _, el := range rt.Elements {
		switch x := el.(type) {
		case *slack.RichTextSection:
			writeSection(&b, x)
		case *slack.RichTextList:
			for _, item := range x.Elements {
				if s, ok := item.(*slack.RichTextSection); ok {
					if b.Len() > 0 {
						b.WriteString(" · ")
					}
					writeSection(&b, s)
				}
			}
		case *slack.RichTextQuote:
			writeSection(&b, &slack.RichTextSection{Elements: x.Elements})
		case *slack.RichTextPreformatted:
			writeSection(&b, &slack.RichTextSection{Elements: x.Elements})
		}
	}
	return b.String()
}

func writeSection(b *strings.Builder, s *slack.RichTextSection) {
	if s == nil {
		return
	}
	for _, sub := range s.Elements {
		switch x := sub.(type) {
		case *slack.RichTextSectionTextElement:
			if b.Len() > 0 && !strings.HasSuffix(b.String(), " ") {
				b.WriteByte(' ')
			}
			b.WriteString(x.Text)
		case *slack.RichTextSectionLinkElement:
			label := x.Text
			if label == "" {
				label = x.URL
			}
			if b.Len() > 0 && !strings.HasSuffix(b.String(), " ") {
				b.WriteByte(' ')
			}
			b.WriteString(label)
		case *slack.RichTextSectionEmojiElement:
			if b.Len() > 0 && !strings.HasSuffix(b.String(), " ") {
				b.WriteByte(' ')
			}
			b.WriteString(":" + x.Name + ":")
		case *slack.RichTextSectionUserElement:
			if b.Len() > 0 && !strings.HasSuffix(b.String(), " ") {
				b.WriteByte(' ')
			}
			b.WriteString("@" + x.UserID)
		}
	}
}

var (
	// stripMrkdwn: Slack mrkdwn `<URL|label>` → label, `<URL>` → URL.
	stripMrkdwnLabeled = regexp.MustCompile(`<[^|>\s]+\|([^>]+)>`)
	stripMrkdwnAuto    = regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9+.\-]{1,31}:[^|>\s]+)>`)
	// stripCommonMarkLink: `[label](url)` → label.
	stripCommonMarkLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	// stripEmphasisChars: `*`, `_`, `~`, `\``, leading `#`s.
	emphasisChars = "*_~`"
	// collapseWS: any run of whitespace → single space.
	collapseWS = regexp.MustCompile(`\s+`)
)

// stripFormatting reduces a mixed mrkdwn / CommonMark string to plain
// text. Conservative: removes only the byte-level markers known to
// have no semantic value in a notification preview.
func stripFormatting(s string) string {
	s = stripMrkdwnLabeled.ReplaceAllString(s, "$1")
	s = stripMrkdwnAuto.ReplaceAllString(s, "$1")
	s = stripCommonMarkLink.ReplaceAllString(s, "$1")
	// Strip emphasis chars, leading ATX hashes, and the `>` blockquote
	// prefix at line start.
	var b strings.Builder
	b.Grow(len(s))
	atLineStart := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if atLineStart {
			if c == '#' {
				for i < len(s) && s[i] == '#' {
					i++
				}
				if i < len(s) && s[i] == ' ' {
					i++
				}
				atLineStart = false
				if i >= len(s) {
					break
				}
				c = s[i]
			} else if c == '>' {
				if i+1 < len(s) && s[i+1] == ' ' {
					i++
				}
				atLineStart = false
				if i+1 >= len(s) {
					break
				}
				continue
			}
		}
		if strings.IndexByte(emphasisChars, c) >= 0 {
			continue
		}
		if c == '\n' {
			b.WriteByte(' ')
			atLineStart = true
			continue
		}
		b.WriteByte(c)
		atLineStart = false
	}
	return b.String()
}

// collapseWhitespace replaces any run of whitespace (spaces, tabs,
// newlines, U+00A0 etc.) with a single ASCII space. Trims edges.
func collapseWhitespace(s string) string {
	return strings.TrimSpace(collapseWS.ReplaceAllString(s, " "))
}

// clampRunes truncates s to at most n runes; when truncated, the
// final rune is replaced with U+2026 (…). Preserves multi-byte
// boundaries.
func clampRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(runes[:n-1]) + "…"
}

// totalLen returns the cumulative byte length of the strings in ss.
// Used for the cap check in DeriveTextFallback.
func totalLen(ss []string) int {
	total := 0
	for _, s := range ss {
		total += len(s)
	}
	return total
}
