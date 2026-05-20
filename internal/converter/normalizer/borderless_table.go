package normalizer

import "strings"

// applyBorderlessTable adds leading/trailing pipes to table rows that
// lack them. The fence walker has already grouped LineTable runs
// (header, delimiter, data), so the repair just scans those lines and
// ensures each one starts and ends with `|`.
//
// GFM accepts borderless rows but real-world Slack `markdown` block
// rendering has been observed to reject them. Adding the edges is
// universally safe — already-bordered rows are detected and skipped.
//
// Catalog code: C6. Evidence: GFM spec — pipes on either end optional;
// showdownjs/showdown #230 on cross-parser inconsistency.
func applyBorderlessTable(lines []Line, _ Options) ([]Line, bool) {
	var fired bool
	for i := range lines {
		if lines[i].Kind != LineTable {
			continue
		}
		text := lines[i].Text
		stripped := strings.TrimSpace(text)
		if stripped == "" {
			continue
		}
		startsWithPipe := strings.HasPrefix(stripped, "|")
		endsWithPipe := strings.HasSuffix(stripped, "|")
		if startsWithPipe && endsWithPipe {
			continue
		}
		// Preserve the original leading indent (CommonMark allows up
		// to three spaces) so the table stays in its block context.
		indent := text[:len(text)-len(strings.TrimLeft(text, " \t"))]
		body := stripped
		if !startsWithPipe {
			body = "| " + body
		}
		if !endsWithPipe {
			body += " |"
		}
		lines[i].Text = indent + body
		fired = true
	}
	return lines, fired
}
