# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog 1.1.1](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

### Changed

### Fixed

---

## [0.4.0] - 2026-05-20

### Added
- **normalizer**: new `internal/converter/normalizer` package that
  repairs 13 evidenced LLM-emission patterns before goldmark parses
  the input. Always-on repairs:
  - **V1** unclosed emphasis (`*italic`, `**bold`) — appends a
    matching closer when there is a single left-flanking opener with
    word content after it.
  - **V2** unclosed inline code (`` `code ``) — appends a closer of
    the unmatched run's length, respecting CommonMark code-span
    nesting (`` ``a `b`` `` is correctly recognized as balanced).
  - **V3** unclosed fenced code block (`` ``` `` or `~~~`) — appends
    a closing fence at EOF to prevent the rest of the document from
    being swallowed.
  - **V4** fence-with-language-no-newline
    (``` ```go fmt.Println("hi")``` ```) — splits onto canonical
    three lines; conservative known-languages whitelist.
  - **V5** ATX header without space (`#Title` → `# Title`).
  - **V7** Unicode look-alikes inside URL paths (en-dash, em-dash,
    curly quotes, ellipsis) → ASCII equivalents; em-dashes in prose
    untouched.
  - **V8** link split across lines (`[label]\n(url)` → `[label](url)`).
  - **C1** stray tilde between word characters escaped (`20~25` →
    `20\~25`); preserves authored `~~` strikethrough.
  - **C3** bullet marker without space (`-item` → `- item`).
  - **C4** numbered marker without space (`1.item` → `1. item`).
  - **C5/C9** trailing whitespace stripped (CommonMark hard-break
    marker preserved).
  - **C6** borderless GFM tables get leading/trailing pipes.
  - **C7** table data rows shorter than the header padded with empty
    cells (layer-B repair in `internal/converter/tables.go`).
  - **R8** `<br>` tags converted to newlines outside table cells and
    spaces inside table cells.

  Two opt-in repairs:
  - **V11** whitelisted HTML entity decode (`&amp; &lt; &gt; &quot;
    &apos;` + numeric refs) gated by `Options.DecodeHTMLEntities`.
  - **V6** asterisk-pair balancer (`**italic*` → `*italic*`) gated
    by `Options.RepairMismatchedEmphasis`.

  Every fired repair surfaces in the response's `warnings` field as
  a single combined string (`normalized input (LLM-mistake repairs
  fired: V8, C3)`). Codes are semver-stable; the full catalog with
  evidence and examples lives at `docs/llm-input-recovery.md`.

  Hardening: 95%+ unit-test coverage on the normalizer package,
  five property tests (idempotence, length bound, no-broadcast-
  smuggling, code-block preservation), a `FuzzNormalize` target run
  for 60s+ across multiple commits with zero failures.

- **converter**: `Options.PreferRichText` (default `false`) — opt-in
  bias toward `rich_text` decomposition over the single Slack
  `markdown` block. `rich_text` renders identically on push
  notifications, search results, screen readers, and the email
  digest, where the `markdown` block's fallback rendering can show
  literal `##` / `**` / `[label](url)` characters. Surfaced as
  `prefer_rich_text` on the MCP convert tool and `--prefer-rich-text`
  on the CLI. Re-exported via `block_kit/`.
- **converter**: `Options.DecodeHTMLEntities` (default `false`) —
  see V11 above. Surfaced as `decode_html_entities` /
  `--decode-html-entities`.
- **converter**: `MarkdownBlockFallbackSurfacesWarning` constant.
  Auto-mode emits this advisory whenever it picks a single
  `markdown` block, naming the fallback surfaces where rendering
  degrades. Re-exported via `block_kit/`.
- **converter**: `DeriveTextFallback([]slack.Block) string` returns
  a 150-char plain-text summary suitable for
  `chat.postMessage(text=)`. Strips Slack mrkdwn, CommonMark links,
  ATX hashes, and blockquote prefixes; header blocks dominate.
  Re-exported via `block_kit/` together with the
  `TextFallbackMaxChars` constant.
- **server**: `ConvertOutput.text_fallback` field carries the
  derived summary in the MCP convert-tool response.

### Changed
- **prompts**: the `format_for_slack` MCP prompt body now instructs
  callers to pass the returned `blocks` as `chat.postMessage(blocks=)`,
  the returned `text_fallback` as `chat.postMessage(text=)`, to
  surface `warnings` to the user (especially normalization repair
  codes and the fallback-surface advisory), and to set
  `prefer_rich_text=true` for accessibility-sensitive channels.

### Deprecated
- The default of `Options.PreferRichText` will flip from `false` to
  `true` in the next major release. When the picker was first
  written, the Slack `markdown` block was the only block type that
  supported headers, tables, task-lists, dividers, and
  code-with-language — so biasing auto-mode toward it was the right
  call. Slack's March 6, 2026 expansion of `rich_text` removed that
  advantage; `rich_text` now renders identically on every surface,
  the `markdown` block does not. To pin the current behavior across
  the flip, set `PreferRichText: false` explicitly.

### Fixed (review feedback)
- **normalizer**: C5 trailing-whitespace repair now skips fenced
  and indented code blocks (CommonMark §4.5 literal content) and
  preserves the §6.7 hard line break in any 2+-space form in
  genuine prose context (not list items, not table delimiter rows
  — those still strip).
- **normalizer**: C3 bullet repair skips lines whose marker
  character repeats (emphasis spans like `**bold**` / `*italic*`
  between two bullet items are no longer misrewritten into
  `* *bold**`).
- **normalizer**: C4 numbered repair regex and peer-check now
  require non-digit content (decimal pairs like `1.5 GB free\n2.3
  GB used` and version triples no longer mutually validate as a
  numbered list).
- **normalizer**: R8 (`<br>`) and V7 (URL Unicode) honor a new
  inline-code-span mask so content inside backticks (CommonMark
  §6.1) survives unchanged. Examples that previously corrupted:
  `` Use `<br>` for HTML breaks `` (R8) and
  `` `array[1](https://x.com/v2—doc)` `` (V7).
- **normalizer**: V4's one-line fence split no longer leaves
  trailing `LineFenceContent` tags that broke idempotence between
  passes. The `classify()` walker now treats a fence-opener whose
  info string already contains a matching closing run as
  `LineProse` (spec-aligns with CommonMark §4.5 — info strings
  cannot contain the fence character).
- **normalizer**: V11 HTML entity decoder is now actually wired
  into the pipeline (was previously a dead Options field). V11
  decodes the five whitelisted XML entities + numeric refs;
  results re-escape through `sanitizeBroadcasts` so broadcast
  tokens cannot round-trip through `&lt;!channel&gt;` → live
  `<!channel>`. The matching `Options.DecodeHTMLEntities`,
  `decode_html_entities` MCP field, and `--decode-html-entities`
  CLI flag now have visible effect.
- **normalizer**: V6 asterisk balancer is now reachable from the
  public API. New `Options.RepairMismatchedEmphasis`,
  `repair_mismatched_emphasis` MCP field, and
  `--repair-mismatched-emphasis` CLI flag thread the existing
  internal flag through every layer.
- **converter (tables)**: `emptyTableCell` now emits a non-null
  `elements` array (mirrors `renderRowCells`' empty-cell fallback
  shape). Was dead code today because goldmark pre-pads short
  rows; future-proofs against any wiring change.

### Docs
- New `docs/llm-input-recovery.md` catalog with evidence for every
  normalizer pattern (issue links, blog references, spec citations).
  Now also documents the inline-code-span guard, the broadcast-
  safety round-trip for V11, and the MCP/CLI knob names for the
  two opt-in repairs.
- `internal/server/cheatsheet.md` (the `block-kit-cheatsheet` MCP
  resource) gained a "Best-effort posting recipe" section and a
  "Troubleshooting: literal `##` / `**` / `[label](url)` appear in
  Slack" section walking through the three diagnostic causes.
- `README.md` gained an "LLM input repairs" section listing every
  normalizer code, a footnote on the modes table explaining the
  `auto`-mode fallback caveat, and a "Troubleshooting" section
  matching the cheatsheet's diagnostic walk.

---

## [0.3.0] - 2026-05-15

### Security
- **converter**: h2–h6 headings — and any heading routed to the
  `section.mrkdwn` fallback (a long h1, or an h1 containing a
  link/image/inline code) — now entity-escape `<`, `>`, `&` in the
  heading text. The fallback previously escaped only the emphasis
  markers `* _ ~ \``, so a heading such as `## <!channel>` in
  `rich_text` mode (or `auto` when the input routes to rich_text
  decomposition) emitted a live `<!channel>` broadcast. The
  `handleFallback` and blockquote unknown-child paths, which also build
  text elements directly, were hardened the same way.
  `allow_broadcasts` and `preserve_mention_tokens` are still honored.

### Added
- **server**: new `block_kit_to_markdown` MCP tool — the inverse of
  `convert_markdown_to_block_kit`. Best-effort and lossy; constructs
  with no Markdown equivalent (buttons, accessories, colors) are
  approximated and reported in `warnings`.
- **server**: all six tools now advertise MCP tool annotations
  (`readOnlyHint`, `openWorldHint`, and a human-readable title).
- **server**: a `block-kit-cheatsheet` MCP resource documenting the
  conversion modes, supported/unsupported Markdown, Slack's documented
  limits, and the mention-safety model.
- **server**: a `format_for_slack` MCP prompt.
- **validator**: per-element rules for buttons (text/value/url/action_id
  lengths), context blocks (≤10 elements), table blocks (≤100 rows, ≤20
  columns, ≤20 column_settings), image-block titles (≤2000 chars), and
  section accessories.
- **validator**: surface-aware validation — `ValidateForSurface` plus a
  `surface` input on `validate_block_kit` / `lint_block_kit` raise the
  block ceiling from 50 (messages) to 100 (modals, App Home tabs).
- **converter**: `Options.MaxNestingDepth` (default 100) rejects
  pathologically deep input with the new `ErrInputTooDeeplyNested`
  sentinel — `MaxInputBytes` bounds bytes but not structural depth.
- **block_kit**: re-exports `BlockKitToMarkdown`, the `Surface` type and
  `SurfaceMessage`/`SurfaceModal`/`SurfaceHomeTab` constants,
  `ErrInputTooDeeplyNested`, and `DefaultMaxNestingDepth`.

### Changed
- **server**: `convert_markdown_to_block_kit`'s `return_preview_url` is
  now a genuine opt-out — pass `false` to skip preview-URL generation
  (the field is a nullable bool, so omitting it still defaults to true).
- **server**: `convert_markdown_to_block_kit` rejects an unknown `split`
  value with a clear error instead of silently ignoring it.
- **server**: the HTTP and SSE transports log a warning when bound to a
  non-loopback address with no bearer token configured.
- **deps**: `slack-go/slack` v0.23.0 → v0.23.1.

### Fixed
- **splitter**: `SplitText` no longer cuts inside a multi-byte UTF-8
  rune when a single un-breakable token (a long CJK run, a non-ASCII
  URL) exceeds the limit — the hard-cut now steps back to the nearest
  rune boundary. The fuzz target gained multibyte seeds and a
  `utf8.ValidString` invariant.
- **converter**: the long-heading `section.mrkdwn` fallback truncation
  is now rune-safe and strips a dangling backslash that could otherwise
  escape the closing `*` and leave the bold run unterminated.

---

## [0.2.1] - 2026-05-13

### Fixed
- **converter**: link conversion in `markdown_block` and `auto` modes.
  The previous emitter ran `entityEscape` over the raw input, which
  destroyed CommonMark autolink syntax (`<https://example.com>` rendered
  as literal `&lt;https://example.com&gt;` text in Slack) and never
  recognized Slack's mrkdwn `<URL|label>` URL-form emitted by Slack tool
  results (so the user's real-world `<https://…|Refa UGC v3 shared-drive>`
  rendered with visible angle brackets and pipe). Replaced with an AST
  walker that re-emits Slack-supported CommonMark text:
  - CommonMark URL autolinks (`<url>`), email autolinks (`<email>`),
    and Linkify-detected bare URLs are promoted to `[url](url)` /
    `[email](mailto:email)` — the only link form Slack's `markdown`
    block documents as supported.
  - Slack's `<URL|label>` mrkdwn URL-form is rewritten to CommonMark
    `[label](URL)` before goldmark parses, fixing the failure for both
    `rich_text` and `markdown_block` modes.
  - Text content still entity-escapes for broadcast safety; URLs and
    code-block contents pass through verbatim.
- **converter**: rich_text mode now recognizes Slack mrkdwn URL-form
  input (`<URL|label>`) and produces a proper `rich_text_section_link`
  element. Previously the construct fell through to the text path and
  was entity-escaped.

### Added
- **converter**: maintainer-facing `TestLinks_PrintBuilderURLs` test
  that prints Block Kit Builder URLs for every link shape × mode so
  visual QA in a Slack workspace is a copy-paste away. Run with
  `go test -v -run TestLinks_PrintBuilderURLs ./internal/converter/`.

---

## [0.2.0] - 2026-05-11

### Added
- **converter**: new `Options.PreserveMentionTokens` flag. When enabled,
  already-typed Slack mention tokens (`<@U…>` / `<@W…>` users, `<#C…>`
  channels, `<!subteam^S…>` usergroups, `<!date^…|fallback>` dates) pass
  through as typed rich_text elements instead of being entity-escaped.
  Catastrophic broadcasts (`<!channel>` / `<!here>` / `<!everyone>`) and
  URL-form tokens still escape — the new flag is strictly additive to
  the safety contract. Surfaced on the `convert_markdown_to_block_kit`
  MCP tool as `preserve_mention_tokens` and on the `convert` CLI
  subcommand as `--preserve-mention-tokens`.
- **server**: streamable-HTTP transport (MCP spec 2025-03) via
  `--http-addr` on the `server` subcommand. Implements graceful
  shutdown, per-session cleanup, body-size cap, slowloris/idle-timeout
  hardening, and DNS-rebinding protection (SDK default).
- **server**: legacy SSE transport (MCP spec 2024-11) via `--sse-addr`
  for older MCP clients. `--http-addr` and `--sse-addr` are mutually
  exclusive.
- **server**: optional bearer-token authentication via `--http-token`
  flag or `MCPSBK_HTTP_TOKEN` environment variable. Applies to both
  HTTP and SSE transports. Constant-time comparison; returns 401 with
  `WWW-Authenticate: Bearer` on mismatch.
- **block_kit**: public `Server` type alias plus `NewServer`,
  `RunStdio`, `RunHTTP`, `RunSSE`, and `HTTPOptions` re-exports so
  external Go consumers can embed the MCP server in their own binary
  without importing `internal/`.
- **repo**: `.mailmap` to canonicalize maintainer commit identities
  across the gmail / GitHub-noreply forms for `git log` and the GitHub
  web UI.

---

## [0.1.0] - 2026-05-09

First public release. A single static binary that exposes a Model
Context Protocol server and a CLI for converting AI-generated markdown
into valid Slack Block Kit JSON — credential-free, zero external runtime
dependencies, supply-chain hardened.

### Highlights

- **Five MCP tools** on top of `modelcontextprotocol/go-sdk` v1.6.0
  (`convert_markdown_to_block_kit`, `validate_block_kit`,
  `preview_block_kit`, `lint_block_kit`, `split_blocks`) — your AI
  assistant generates a Slack message and one tool call returns the
  Block Kit JSON, validated against documented Slack constraints, with
  a one-click Block Kit Builder URL for visual QA.
- **Auto-mode picker** chooses between Slack's new (Feb 2025) `markdown`
  block and full deterministic decomposition (`rich_text` / `section` /
  `header` / `image` / `divider` / `table`) based on input
  characteristics. Five non-representable nesting patterns
  (code-in-quote / code-in-list / table-in-quote / table-in-list /
  list-in-quote) are detected and routed to predictable rich_text
  decomposition with `Offset`-based ordered-list numbering continuation
  across splits.
- **Mention-sanitization is mandatory by default.** Every text run
  emitted into a Slack `text` field is HTML-entity-escaped, so
  AI-generated content containing literal `<!channel>`, `<!here>`,
  `<@U…>`, `<#C…>`, or `<!subteam^…>` cannot broadcast or ping the
  workspace. Opt-in passthrough via `Options.AllowBroadcasts: true`.
- **Supply-chain hardened release**: cosign keyless signing of every
  artifact via GitHub OIDC, CycloneDX SBOMs generated by syft, all
  GitHub Actions pinned to commit SHAs, [OSSF Scorecard 7.1/10][ossf]
  with all of `Pinned-Dependencies`, `Token-Permissions`, `SAST`,
  `Fuzzing`, `Vulnerabilities`, `License`, `Binary-Artifacts`,
  `Dangerous-Workflow`, `Packaging`, and `Dependency-Update-Tool` at
  10/10.
- **293 tests, ≥80% statement coverage on every shipped package**, with
  stdlib fuzz tests on the splitter (`FuzzSplitText`) and a
  `TestNested_PrintBuilderURLs` fixture that emits Block Kit Builder
  URLs for manual visual verification of every nesting pattern × mode
  combination.

[ossf]: https://scorecard.dev/viewer/?uri=github.com/hishamkaram/mcp-slack-block-kit

### Added

- **MCP server (stdio)**: `mcp-slack-block-kit` (default subcommand
  `server`) exposes the five tools above. Compatible with Claude
  Desktop, Cursor, Continue.dev, Zed, Cline, and any MCP client that
  speaks the stdio transport.
- **CLI**: `mcp-slack-block-kit convert` — pipe markdown on stdin, get
  Block Kit JSON on stdout, optional Block Kit Builder URL on stderr.
  Flags: `--mode={auto|rich_text|markdown_block|section_mrkdwn}`,
  `--allow-broadcasts`, `--block-id-prefix`, `--max-input-bytes`,
  `--pretty`, `--preview`.
- **Public Go library** at `github.com/hishamkaram/mcp-slack-block-kit/block_kit`
  re-exports the converter, validator, splitter, and preview engines
  for embedded use without the MCP server dependencies.
  `Renderer.ConvertWithWarnings()` returns blocks plus fallback notes
  (e.g. when auto mode routes away from the markdown block because the
  input contains code-in-blockquote).
- **Markdown coverage**: paragraphs, headings (H1 short → `header`;
  H1 long / links/images/code → bold `section`; H2–H6 always bold
  `section`), thematic breaks → `divider`, blockquotes →
  `rich_text_quote`, ordered + unordered lists with sibling-with-
  incrementing-`indent` nesting (Slack's required pattern), fenced and
  indented code blocks → `rich_text_preformatted` with language tag
  preserved, GFM tables → `slack.TableBlock` with row/col truncation
  and header replication on overflow, inline emphasis with style stack
  + OR-merge for `***bold-italic***`, `:emoji:` shortcode resolution
  with goldmark-fragmentation post-processing, `@handle` → `<@U…>`
  resolution via `Options.MentionMap`.
- **Validator** (`internal/validator`): six cross-block rules
  (50-block-per-message ceiling, unique `block_id`, multiple-tables
  detection, markdown-block 12k cumulative cap, per-block validators
  for section/header/image/actions, deprecated-pattern flagging via
  `ValidateStrict`).
- **Splitter** (`internal/splitter`): `SplitText` — whitespace-aware
  splitter preferring paragraph > sentence > word boundaries,
  fuzz-tested for byte-for-byte round-trip preservation. `ChunkBlocks`
  — enforces the 50-block-per-message ceiling AND
  `only_one_table_allowed` (a second `TableBlock` always opens a new
  chunk).
- **Preview** (`internal/preview`): `BuilderURL(blocks)` produces
  `https://app.slack.com/block-kit-builder/#<URL_ENCODED_JSON>` URLs,
  with truncation flagging above ~8 KiB.
- **GoReleaser pipeline**: multi-arch builds for linux/darwin/windows ×
  amd64/arm64, Homebrew tap auto-published to
  `hishamkaram/homebrew-tap`, cosign keyless signing of `checksums.txt`,
  CycloneDX SBOMs per archive.

### Security

- Mention sanitization is the documented security-critical path. See
  [SECURITY.md](SECURITY.md) and `.claude/rules/security.md`. The
  conformance suite `TestSanitization_BroadcastForms_AllEscapedByDefault`
  covers six broadcast forms (`!channel`, `!here`, `!everyone`, user
  mention, channel reference, subteam) plus nested angle brackets and
  bare ampersand.
- `Options.MaxInputBytes` (default 256 KiB) caps markdown input before
  goldmark allocates an AST. Prevents trivial memory exhaustion on a
  public-facing MCP server.
- Block Kit Builder URLs use `url.QueryEscape` over the entire
  `{"blocks":[...]}` payload — no fragment escape via the JSON encoder
  alone.
- `CodeQL security-and-quality` query suite (the strictest) runs on
  every push and PR.
- `govulncheck` runs in CI and lefthook pre-push.
- `gosec` (G304 excluded; we don't read user-controlled file paths)
  runs in CI via `golangci-lint`.

### Install

```sh
# Homebrew (after this release lands the formula in homebrew-tap)
brew install hishamkaram/tap/mcp-slack-block-kit

# Go install
go install github.com/hishamkaram/mcp-slack-block-kit/cmd/mcp-slack-block-kit@v0.1.0

# Or grab a prebuilt + cosign-verifiable binary from
# https://github.com/hishamkaram/mcp-slack-block-kit/releases/tag/v0.1.0
```

Verify a release:

```sh
cosign verify-blob \
  --certificate-identity-regexp 'https://github\.com/hishamkaram/mcp-slack-block-kit/.+' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  --certificate checksums.txt.pem --signature checksums.txt.sig \
  checksums.txt
```

### Notes / known caveats

- Three claims in the rich_text decomposition design are SCHEMA-ONLY or
  UNVERIFIED: sibling rich_text decomposition rendering (quote-bar
  doesn't visually wrap embedded preformatted), cross-block
  ordered-list numbering continuation via `Offset`, and `markdown`
  block rendering of code-in-quote / list-in-quote / code-in-list. The
  picker conservatively routes around the third one. The
  `TestNested_PrintBuilderURLs` fixture exists for manual visual
  verification by anyone with a Slack workspace.
- `slack-go/slack` v0.23.0's `RichTextPreformatted.Language` field
  serializes into JSON but Slack itself does not syntax-highlight. We
  preserve the tag for tooling.
- Slack Block Kit Builder URLs above ~8 KiB get unreliable in
  browsers/Slack — the preview tool flags those as `Truncated: true`.

[Unreleased]: https://github.com/hishamkaram/mcp-slack-block-kit/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/hishamkaram/mcp-slack-block-kit/releases/tag/v0.4.0
[0.3.0]: https://github.com/hishamkaram/mcp-slack-block-kit/releases/tag/v0.3.0
[0.2.1]: https://github.com/hishamkaram/mcp-slack-block-kit/releases/tag/v0.2.1
[0.2.0]: https://github.com/hishamkaram/mcp-slack-block-kit/releases/tag/v0.2.0
[0.1.0]: https://github.com/hishamkaram/mcp-slack-block-kit/releases/tag/v0.1.0
