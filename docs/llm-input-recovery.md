# LLM input recovery catalog

This document is the public contract for which markdown
malformations `mcp-slack-block-kit`'s normalizer repairs before
goldmark parses. Each pattern has a stable two-character code (V*,
C*, R*) reported in the converter's `warnings` slice when the
repair fires (`normalized input (LLM-mistake repairs fired:
V8, C3)`). The codes are semver-stable; renames need a major
version bump.

The normalizer is **on by default**. Two opt-in flags expand the
pattern set:

- `Options.DecodeHTMLEntities` (MCP: `decode_html_entities`; CLI:
  `--decode-html-entities`) — enables the whitelisted HTML-entity
  decoder (V11). Off by default because the decoded characters
  re-escape through broadcast sanitization; safe to enable.
- `Options.RepairMismatchedEmphasis` (MCP:
  `repair_mismatched_emphasis`; CLI: `--repair-mismatched-emphasis`)
  — enables V6's asterisk balancer. Off by default because V6 is
  the catalog's trickiest pattern and false positives corrupt
  prose with deliberate asymmetric asterisks.

All repairs:
- Are **idempotent**: `Normalize(Normalize(s)) == Normalize(s)`,
  pinned by property test + fuzz.
- Skip content inside fenced code blocks (` ``` ` / `~~~`) and
  indented code blocks (CommonMark §4.4 / §4.5 literal content).
- Skip content inside inline code spans (CommonMark §6.1 literal
  content). Byte-level prose repairs (R8 `<br>`, V7 URL Unicode,
  V11 entity decode) honor an inline-code mask derived from the
  same backtick pair-matching algorithm V2 uses.
- Are **broadcast-safe**: no repair introduces literal
  `<!channel>` / `<!here>` / `<!everyone>` that wasn't in the
  input. The V11 decoder is whitelisted to the five XML entities
  + numeric refs; even when it decodes a smuggled
  `&lt;!channel&gt;` → `<!channel>`, the converter's downstream
  `sanitizeBroadcasts` pass re-escapes it before it reaches Slack.
- Are **O(n)** on input length.

## Repair codes

### V-tier (very common)

| Code | Repair | Default |
|---|---|---|
| **V1** | Unclosed emphasis: append `*` / `**` to close a trailing single opener on the last line of a paragraph (left-flanking + word-content-after guards). | on |
| **V2** | Unclosed inline code: append `` ` `` of matching length when an unmatched backtick opener has non-backtick content after it. Skips lines that end in a backtick or contain a fence-length run. | on |
| **V3** | Unclosed fenced code block: append a closing fence at EOF when the input ended inside a `` ``` `` or `~~~` block. **Highest priority** — without this, an orphaned fence consumes every trailing header, list, and paragraph. | on |
| **V4** | Fence with language but no newline: splits `` ```go fmt.Println("hi")``` `` into the canonical three-line form. Conservative known-languages whitelist. | on |
| **V5** | ATX header without space: `#Title` → `# Title`. | on |
| **V6** | Mismatched asterisks: `**italic*` → `*italic*`. Collapses each adjacent open/close pair to the smaller count. **Opt-in** via `Options.RepairMismatchedEmphasis`. | off |
| **V7** | Unicode look-alikes in URL paths: en-dash, em-dash, curly quotes, ellipsis → ASCII equivalents. Only inside `[text](url)` and `<url>` spans (prose em-dashes untouched). | on |
| **V8** | Link split across lines: `[label]\n(url)` → `[label](url)`. Inner `(...)` must be URL-ish (`:`, `/`, `@`, or `#`). | on |
| **V11** | HTML entity decode (whitelist: `&amp; &lt; &gt; &quot; &apos;` + numeric refs). **Opt-in** via `Options.DecodeHTMLEntities`. | off |

### C-tier (common)

| Code | Repair | Default |
|---|---|---|
| **C1** | Single tilde in word: `20~25°C` → `20\~25°C`. Skips lines that already use `~~` strikethrough. | on |
| **C3** | Bullet without space: `-item` → `- item`. Requires an adjacent sibling list-item line to suppress false positives like `-1 means undefined`. | on |
| **C4** | Numbered list without space: `1.item` → `1. item`. Same peer-presence guard. | on |
| **C5** / **C9** | Trailing whitespace stripped (preserves the two-trailing-spaces CommonMark hard-break marker). | on |
| **C6** | Borderless table: adds leading/trailing `\|` to rows missing them. | on |
| **C7** | Table column-count mismatch: data rows shorter than the header are padded with empty cells (post-AST, in `internal/converter/tables.go`). | on |

### R-tier (rare)

| Code | Repair | Default |
|---|---|---|
| **R8** | `<br>` / `<br/>` / `<br />` (any case): converted to newline outside table cells, single space inside table cells. | on |

## Patterns deliberately NOT repaired

- **R1** setext headings (`====` under text). goldmark handles correctly.
- **R2** `***` thematic break. goldmark handles correctly.
- **R3** mixed bullet markers in one list. Risk > reward.
- **R4** `>quoted` without space. goldmark accepts.
- **R5** `~~~` triple-tilde fence. Handled by V3.
- **R6** unresolved reference-style links. Low priority; would need AST walk.
- **R7** inline triple-backtick. Overlaps with V3; can't safely disambiguate.
- **R9** soft vs hard line breaks. Rendering preference, not a malformation.
- **R10** multiple blank lines in lists. Could be intentional.
- **C2** bare emails. goldmark Linkify already handles.
- **C8** list indent inconsistency. Risk of regex damage; revisit if reports surface.
- **C10** over-escaped characters (`\#`, `\_` etc.). goldmark strips spec-compliantly.

## Evidence sources

Every confirmed pattern has at least one cited source. The full
evidence map (with URLs and dates) was assembled during the
plan-mode research pass and lives in the project's internal
research notes. Key references that drove the catalog:

- [`vercel/streamdown` + `remend`](https://www.npmjs.com/package/remend) — production-grade
  repair library for streaming markdown; their handler list is the
  V-tier benchmark.
- [Vercel changelog: "New npm package for automatic recovery of broken streaming markdown"](https://vercel.com/changelog/new-npm-package-for-automatic-recovery-of-broken-streaming-markdown)
- [Streamdown 2.5 changelog (singleTilde option)](https://vercel.com/changelog/streamdown-2-5)
- [Apify RAG Web Browser — Markdown Links Split Across Multiple Lines](https://apify.com/apify/rag-web-browser/issues/markdown-links-split-nIyBxKXLZrnlpbWc9)
- [OpenAI community — Markdown Formatting Issues with GPT-5](https://community.openai.com/t/markdown-formatting-issues-with-gpt-5/1337570)
- [Claude Code issue #26390 — HTML Entities Not Decoded](https://github.com/anthropics/claude-code/issues/26390)
- [Claude Code issue #19251 — Markdown renderer treats single tilde as strikethrough](https://github.com/anthropics/claude-code/issues/19251)
- [Context-Link.ai — "Claude Em-Dash Problem"](https://context-link.ai/blog/claude-em-dash-remover)
- [Briskly — "How to Stop AI Em Dashes (Claude, ChatGPT, Gemini)"](https://briskly.tools/guides/how-to-stop-ai-em-dashes)
- [Unmarkdown — "Why Do Asterisks Appear When I Paste from ChatGPT"](https://unmarkdown.com/blog/why-do-asterisks-appear-when-i-paste-from-chatgpt)
- [CleanTextTools — "Why Does ChatGPT Add Asterisks"](https://cleantexttools.com/blog/why-does-chatgpt-add-asterisks/)
- [Cultman Sachs, Medium — "Why Can't AI Models Output Clean Markdown"](https://medium.com/@CultmanSachs/why-cant-ai-models-output-clean-markdown-a-technical-mess-that-still-isn-t-fixed-1dc70ff366a3)
- [`streaming-markdown` by thetarnav](https://github.com/thetarnav/streaming-markdown)
- [eslint-markdown `no-missing-atx-heading-space`](https://github.com/eslint/markdown/blob/main/docs/rules/no-missing-atx-heading-space.md)
- [markedjs PR #2201 (trailing whitespace on delimiter row)](https://github.com/markedjs/marked/pull/2201)
- [GFM spec (autolinks, tables, strikethrough)](https://github.github.com/gfm/)
- [CommonMark spec](https://spec.commonmark.org/0.30/)
