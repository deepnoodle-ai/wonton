package highlight

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/tui"
)

func TestMarkdown_ImportColorsCodeBlocks(t *testing.T) {
	result, err := tui.NewMarkdownRenderer().Render("```go\nfunc main() {}\n```\n")
	assert.NoError(t, err)
	var colored bool
	for _, seg := range result.Lines[0].Segments {
		if seg.Text == "func" && seg.Style.FgRGB != nil {
			colored = true
		}
	}
	assert.True(t, colored, "func keyword has no syntax color")
}

func TestMarkdown_RowsMatchCodeLines(t *testing.T) {
	code := "func main() {\n\tprintln(\"hi\")\n}\n"
	rows := Markdown(code, "go", "monokai")
	assert.Equal(t, len(rows), 3)
	var lines []string
	for _, row := range rows {
		var line strings.Builder
		for _, seg := range row {
			line.WriteString(seg.Text)
		}
		lines = append(lines, line.String())
	}
	assert.Equal(t, strings.Join(lines, "\n")+"\n", code)
}

func TestMarkdown_UnknownLanguageReturnsNil(t *testing.T) {
	assert.Nil(t, Markdown("x", "no-such-language", "monokai"))
}

func TestMarkdown_UnknownThemeUsesFallback(t *testing.T) {
	assert.NotNil(t, Markdown("func main() {}", "go", "no-such-theme"))
}
