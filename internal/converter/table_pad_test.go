package converter

import (
	"testing"

	"github.com/slack-go/slack"
)

// TestTableShortRowsPadded: when an LLM emits a table whose data
// rows have fewer cells than the header, the converter pads each
// short row with empty cells so Slack accepts the payload.
func TestTableShortRowsPadded(t *testing.T) {
	in := "| A | B | C |\n| --- | --- | --- |\n| 1 |\n| 2 | x |\n| 3 | y | z |\n"
	opts := DefaultOptions()
	opts.PreferRichText = true // route to rich_text path so a TableBlock is emitted
	r, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	blocks, _, err := r.ConvertWithWarnings(in)
	if err != nil {
		t.Fatal(err)
	}
	// Find the table block.
	var tb *slack.TableBlock
	for _, b := range blocks {
		if x, ok := b.(*slack.TableBlock); ok {
			tb = x
			break
		}
	}
	if tb == nil {
		t.Fatalf("expected a TableBlock, got %v", blockTypes(blocks))
	}
	// Header + 3 data rows.
	if len(tb.Rows) != 4 {
		t.Fatalf("expected 4 rows (header+3 data), got %d", len(tb.Rows))
	}
	// Every row must have exactly 3 cells.
	for i, row := range tb.Rows {
		if len(row) != 3 {
			t.Errorf("row %d has %d cells, want 3", i, len(row))
		}
	}
}
