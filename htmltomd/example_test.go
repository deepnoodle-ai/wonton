package htmltomd_test

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/htmltomd"
)

// A page laid out with a table, as many older sites are: the table is
// unwrapped, and quotes, code links, and definition lists keep their shape.
func ExampleConvert() {
	page := `<table><tr><td>
<h1>Release notes</h1>
<blockquote>Use <code><a href="/pkg#Client">http.Client</a></code> for <b>all</b> requests.</blockquote>
<dl><dt>Timeout</dt><dd>Default is 30s.</dd></dl>
</td></tr></table>`
	fmt.Println(htmltomd.Convert(page))
	// Output:
	// # Release notes
	//
	// > Use [`http.Client`](/pkg#Client) for **all** requests.
	//
	// **Timeout**
	// : Default is 30s.
}
