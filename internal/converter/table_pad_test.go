package converter

import (
	"encoding/json"
	"strings"
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

// TestEmptyTableCell_WireShapeNeverNull pins the JSON shape of the
// emptyTableCell helper directly. slack-go's RichTextSection has no
// omitempty on Elements, so a nil-Elements section would serialize
// as `"elements":null` and Slack would reject the payload. Mirroring
// renderRowCells' single zero-length text element keeps the shape an
// empty-string-bearing array.
func TestEmptyTableCell_WireShapeNeverNull(t *testing.T) {
	cell := emptyTableCell()
	raw, err := json.Marshal(cell)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(raw)
	if strings.Contains(body, "\"elements\":null") {
		t.Errorf("rich_text_section.elements is null in output: %s", body)
	}
	// Positive check: structural array with the placeholder text element.
	if !strings.Contains(body, "\"type\":\"text\"") {
		t.Errorf("expected zero-length text placeholder in output: %s", body)
	}
}
