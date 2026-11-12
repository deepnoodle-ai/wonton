package htmlparse_test

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/htmlparse"
)

// ExampleDocument_Transform shows that Transform removes scripts, event
// handlers, and javascript: URLs from a fetched page.
func ExampleDocument_Transform() {
	page := `<body><script>steal()</script>` +
		`<img src="cat.png" onerror="steal()">` +
		`<a href="javascript:steal()">Click</a>` +
		`<a href="/about">About</a></body>`
	doc, _ := htmlparse.Parse(page)
	fmt.Println(doc.Transform(&htmlparse.TransformOptions{OnlyMainContent: true}))
	// Output:
	// <body><img src="cat.png"><a>Click</a><a href="/about">About</a></body>
}
