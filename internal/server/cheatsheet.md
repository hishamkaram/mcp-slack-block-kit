# Slack Block Kit conversion cheat sheet

This MCP server converts AI-generated Markdown into Slack Block Kit JSON.
Read this before calling `convert_markdown_to_block_kit` so you pass the
right `mode` and options.

## Conversion modes

| `mode` | Behavior |
|---|---|
| `auto` (default) | Picks per input: a single Slack `markdown` block for short, simple output; full `rich_text` decomposition otherwise. |
| `rich_text` | Always decomposes into `rich_text` / `section` / `header` / `image` / `divider` / `table` blocks. |
| `markdown_block` | Always emits one Slack `markdown` block. Errors if input exceeds 12,000 characters. |
| `section_mrkdwn` | Always emits `section` blocks with `mrkdwn` text (older shape). |

## Supported Markdown

Headings, bold, italic, strikethrough, inline code, fenced code blocks,
ordered/unordered lists (including nesting), block quotes, thematic breaks
(`---`), links `[text](url)`, images `![alt](url)`, GFM tables, task lists,
and `:emoji:` shortcodes. Bare URLs are auto-linked.

## Not supported

Footnotes and definition lists are not recognized (emitted as plain text).
Raw HTML is entity-escaped to literal text, not rendered. Rich-text link
elements have no tooltip/title field, so Markdown link titles are dropped
in `rich_text` mode (kept in `markdown_block` mode).

## Mention safety (important)

By default every literal `<!channel>`, `<!here>`, `<!everyone>`, `<@U…>`,
`<#C…>`, and `<!subteam^…>` in the input is HTML-entity-escaped so it
cannot broadcast or ping the workspace. Keep it that way for
LLM-generated content.

- `mention_map` — resolve bare `@handle` text to a Slack ID safely. The
  preferred way to produce real mentions.
- `preserve_mention_tokens` — let already-typed Slack tokens
  (`<@U…>`, `<#C…>`, `<!subteam^S…>`, `<!date^…>`) pass through while
  catastrophic broadcasts still escape. Use when the Markdown came from a
  trusted upstream Slack tool result.
- `allow_broadcasts` — disables all escaping. Only set this when the user
  explicitly intends to ping a channel.

## Slack-documented limits

| Constraint | Limit |
|---|---|
| Blocks per message | 50 |
| Blocks per modal / App Home tab | 100 |
| `section` text | 3,000 chars |
| `section` fields | 10 fields, 2,000 chars each |
| `header` text | 150 chars (plain_text only) |
| `image` alt_text / title | 2,000 chars |
| `image` URL | 3,000 chars |
| `context` block | 10 elements |
| `actions` block | 25 elements |
| Button text / value / url | 75 / 2,000 / 3,000 chars |
| `table` block | 100 rows, 20 columns, one table per message |
| `markdown` blocks (cumulative) | 12,000 chars |
| `block_id` | 255 chars |

## Companion tools

- `validate_block_kit` — check a payload against the limits above; pass
  `surface` (`message` / `modal` / `home`) to set the block ceiling.
- `lint_block_kit` — advisory near-limit and accessibility warnings.
- `split_blocks` — chunk an oversized payload to the 50-block limit.
- `preview_block_kit` — get a Block Kit Builder URL for visual QA.
- `block_kit_to_markdown` — the inverse conversion (lossy).

## Best-effort posting recipe

`convert_markdown_to_block_kit` returns:

- `blocks` — the typed Block Kit array.
- `text_fallback` — a derived plain-text summary (≤ 150 chars).
- `warnings` — repair codes (`normalized input (LLM-mistake repairs
  fired: V8, C3)`) plus, in auto mode, the markdown-block
  fallback-surface advisory.

Wire them into `chat.postMessage` like this:

```
chat.postMessage(
    channel = "...",
    blocks  = response.blocks,         // typed blocks
    text    = response.text_fallback,  // for notifications / search
)
```

Never put the markdown source into `text:` — Slack renders that field
as mrkdwn, so literal `##`/`**`/`[label](url)` characters appear.

## Troubleshooting: literal `##`, `**`, `[label](url)` appear in Slack

Three causes, in order of likelihood:

1. **The caller wired the result wrong.** Pass the returned `blocks`
   array as `chat.postMessage(blocks=...)` and the returned
   `text_fallback` as `chat.postMessage(text=...)`. Never put the
   markdown source in either field.
2. **The LLM input was malformed.** This server auto-repairs 14
   evidenced LLM-emission patterns (link split across lines,
   unclosed emphasis / inline code / fenced code, ATX header
   without space, bullet/numbered list without space, Unicode
   bullet markers like `•`/`◦`/`★` used instead of `-`, smart
   quotes / em-dashes in URLs, single-tilde in word, borderless
   tables, `<br>` tags, plus padded short table rows). The response's
   `warnings` field reports the codes that fired (e.g.
   `normalized input (LLM-mistake repairs fired: V8, C3)`). The
   full catalog with evidence and examples lives at
   `docs/llm-input-recovery.md`.
3. **Fallback surface degradation.** The Slack `markdown` block
   (auto mode's default for short, simple output) renders well in
   the main channel pane but on push notifications, search
   results, screen readers, and the email digest it can show
   literal markdown characters. Fix options:
   - Set `prefer_rich_text: true` on the call. The picker then
     chooses `rich_text` decomposition for the same inputs;
     `rich_text` renders identically on every Slack surface.
   - Or pass `mode: "rich_text"` to force decomposition.
   - Always supply the `text_fallback` as `chat.postMessage(text=)`
     — Slack uses that string verbatim on every fallback surface.

The default of `prefer_rich_text` will flip to `true` in the next
major release. To pin the current behavior across the flip, set it
explicitly to `false`.
