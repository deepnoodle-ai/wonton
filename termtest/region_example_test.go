package termtest

import "fmt"

// Example of a scroll region (DECSTBM). A line feed at the bottom of the
// region scrolls only the region's rows. A region that starts at row 1
// sends the rows that leave it to scrollback, as a real terminal does.
func ExampleScreen_Scrollback() {
	s := NewScreen(10, 5)
	s.Write([]byte("1\r\n2\r\n3\r\n4\r\n5"))
	s.Write([]byte("\x1b[2;4r"))    // region rows 2-4, cursor goes home
	s.Write([]byte("\x1b[4;1H\nX")) // LF at region bottom scrolls rows 2-4
	for y := 0; y < 5; y++ {
		fmt.Printf("%d: %q\n", y+1, s.Row(y))
	}
	fmt.Printf("scrollback: %q\n", s.Scrollback())

	s.Write([]byte("\x1b[1;2r\x1b[2;1H\n")) // region rows 1-2, LF at row 2
	fmt.Printf("scrollback: %q\n", s.Scrollback())

	// Output:
	// 1: "1"
	// 2: "3"
	// 3: "4"
	// 4: "X"
	// 5: "5"
	// scrollback: []
	// scrollback: ["1"]
}
