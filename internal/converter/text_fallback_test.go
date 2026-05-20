package converter

import (
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

func TestDeriveTextFallback_TableCases(t *testing.T) {
	cases := []struct {
		name   string
		blocks []slack.Block
		want   string
	}{
		{
			name:   "empty input",
			blocks: nil,
			want:   "",
		},
		{
			name: "header dominates",
			blocks: []slack.Block{
				slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", "Release notes", false, false)),
				slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "more details below", false, false), nil, nil),
			},
			want: "Release notes",
		},
		{
			name: "section mrkdwn stripped",
			blocks: []slack.Block{
				slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "*bold* and _italic_ text", false, false), nil, nil),
			},
			want: "bold and italic text",
		},
		{
			name: "Slack mrkdwn link stripped to label",
			blocks: []slack.Block{
				slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "see <https://x.com|the docs>", false, false), nil, nil),
			},
			want: "see the docs",
		},
		{
			name: "Slack mrkdwn autolink stripped to URL",
			blocks: []slack.Block{
				slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "visit <https://example.com>", false, false), nil, nil),
			},
			want: "visit https://example.com",
		},
		{
			name: "markdown block CommonMark link stripped",
			blocks: []slack.Block{
				slack.NewMarkdownBlock("", "click [here](https://x.com) please"),
			},
			want: "click here please",
		},
		{
			name: "markdown block ATX header stripped",
			blocks: []slack.Block{
				slack.NewMarkdownBlock("", "# Title\n\nbody"),
			},
			want: "Title body",
		},
		{
			name: "image placeholder",
			blocks: []slack.Block{
				slack.NewImageBlock("https://example.com/img.png", "Cute kitten", "", nil),
			},
			want: "[image: Cute kitten]",
		},
		{
			name: "divider yields empty",
			blocks: []slack.Block{
				slack.NewDividerBlock(),
			},
			want: "",
		},
		{
			name: "whitespace collapses",
			blocks: []slack.Block{
				slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "a  \n  b\t\tc", false, false), nil, nil),
			},
			want: "a b c",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DeriveTextFallback(tc.blocks)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDeriveTextFallback_Truncation(t *testing.T) {
	long := strings.Repeat("a", 200)
	blocks := []slack.Block{
		slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", long, false, false), nil, nil),
	}
	got := DeriveTextFallback(blocks)
	// 150 chars max with a trailing ellipsis when clipped.
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis suffix on truncated output: %q", got)
	}
	if rs := []rune(got); len(rs) != TextFallbackMaxChars {
		t.Errorf("truncated length = %d runes, want %d", len(rs), TextFallbackMaxChars)
	}
}

func TestDeriveTextFallback_RichTextWithLink(t *testing.T) {
	// Build a rich_text block: "see " + link[example.com|click here]
	link := slack.NewRichTextSectionLinkElement(
		"https://example.com", "click here", nil,
	)
	text := slack.NewRichTextSectionTextElement("see ", nil)
	section := slack.NewRichTextSection(text, link)
	rt := slack.NewRichTextBlock("", section)
	got := DeriveTextFallback([]slack.Block{rt})
	if !strings.Contains(got, "click here") {
		t.Errorf("expected link text to survive, got %q", got)
	}
	if strings.Contains(got, "https://") {
		t.Errorf("link URL should be dropped, got %q", got)
	}
}
