package normalizer

import "regexp"

// fenceLangNoNewlineBacktick / fenceLangNoNewlineTilde match the
// one-line construct ```go fmt.Println("hi")``` / ~~~py print()~~~
// where the model crammed language tag, body, and closer onto a
// single line. We split the patterns because Go's regexp engine
// (RE2) does not support backreferences for matching the closer's
// fence character to the opener's.
//
// Captures: (1) leading indent (≤3 spaces), (2) the opening fence
// run, (3) the language token (length 1–32, conservative charset),
// (4) the body. The closing run is matched against the same
// character at end of line.
var (
	fenceLangNoNewlineBacktick = regexp.MustCompile(
		"^( {0,3})(`{3,})([A-Za-z0-9+_.-]{1,32})\\s+(.+?)\\s*`{3,}\\s*$",
	)
	fenceLangNoNewlineTilde = regexp.MustCompile(
		`^( {0,3})(~{3,})([A-Za-z0-9+_.-]{1,32})\s+(.+?)\s*~{3,}\s*$`,
	)
)

// knownLanguages is the whitelist used by applyFenceLangNoNewline to
// decide whether a token after the fence is a real language tag. Kept
// short and conservative — false positives here corrupt prose that
// happens to look like a fence + word + content.
//
// Languages added: every entry from the Linguist language list that
// LLMs commonly emit. Add more if needed; CHANGELOG should mention
// the additions so external callers know.
var knownLanguages = map[string]bool{
	"go": true, "golang": true,
	"js": true, "javascript": true, "ts": true, "typescript": true,
	"py": true, "python": true,
	"rb": true, "ruby": true,
	"rs": true, "rust": true,
	"c": true, "cpp": true, "cxx": true, "cc": true, "c++": true,
	"java": true, "kt": true, "kotlin": true,
	"swift": true,
	"php":   true, "perl": true, "pl": true,
	"sh": true, "bash": true, "zsh": true, "shell": true,
	"sql": true, "json": true, "yaml": true, "yml": true, "toml": true, "xml": true,
	"html": true, "css": true, "scss": true, "sass": true, "less": true,
	"md": true, "markdown": true,
	"r": true, "matlab": true, "scala": true, "clojure": true, "clj": true,
	"haskell": true, "hs": true, "elm": true, "ocaml": true, "ml": true,
	"erlang": true, "elixir": true, "ex": true, "exs": true,
	"dart": true, "lua": true, "vim": true,
	"text": true, "txt": true, "plain": true, "diff": true, "patch": true,
	"dockerfile": true, "makefile": true,
	"powershell": true, "ps1": true, "ps": true,
	"groovy": true, "tf": true, "terraform": true, "hcl": true,
	"proto": true, "graphql": true, "gql": true,
}

// applyFenceLangNoNewline splits a single-line fenced code block into
// the canonical three-line form (opener / body / closer). Conservative:
// fires only when the language token is in `knownLanguages`.
//
// Catalog code: V4. Evidence: Cultman Sachs Medium fragmented /
// missing-newline code blocks; goldmark issue #440.
func applyFenceLangNoNewline(lines []Line, _ Options) ([]Line, bool) {
	var (
		fired bool
		out   = make([]Line, 0, len(lines))
	)
	for i := range lines {
		// V4 must consider LineFenceOpen too: the classifier sees a
		// one-line ```lang body``` as an opener with the body+closer
		// crammed into the "info string". Skipping non-prose would
		// miss those. We still skip LineFenceContent / LineFenceClose
		// / LineIndentedCode (those are inside a real fence and
		// shouldn't be re-split).
		switch lines[i].Kind {
		case LineFenceContent, LineFenceClose, LineIndentedCode:
			out = append(out, lines[i])
			continue
		}
		m := fenceLangNoNewlineBacktick.FindStringSubmatch(lines[i].Text)
		if m == nil {
			m = fenceLangNoNewlineTilde.FindStringSubmatch(lines[i].Text)
		}
		if m == nil {
			out = append(out, lines[i])
			continue
		}
		if !knownLanguages[lowercaseASCII(m[3])] {
			out = append(out, lines[i])
			continue
		}
		indent, fence, lang, body := m[1], m[2], m[3], m[4]
		out = append(
			out,
			Line{Text: indent + fence + lang, Kind: LineFenceOpen, FenceLang: lang},
			Line{Text: body, Kind: LineFenceContent, FenceLang: lang},
			Line{Text: indent + fence, Kind: LineFenceClose},
		)
		fired = true
	}
	return out, fired
}

// lowercaseASCII returns s with ASCII letters lowercased. Avoids the
// strings.ToLower allocation for the common all-lowercase case.
func lowercaseASCII(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if out == nil {
				out = make([]byte, i, len(s))
				copy(out, s[:i])
			}
			out = append(out, c+('a'-'A'))
			continue
		}
		if out != nil {
			out = append(out, c)
		}
	}
	if out == nil {
		return s
	}
	return string(out)
}
