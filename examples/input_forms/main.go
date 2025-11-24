package main

import (
	"fmt"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer terminal.Reset()

	// Clear the screen
	terminal.Clear()

	input := gooey.NewInput(terminal)

	terminal.Println("📝 Gooey Input Form Demo")
	terminal.Println("------------------------")
	terminal.Flush()

	// 1. Basic Input
	// We use ReadSimple which works well in standard terminal mode
	input.WithPrompt("What is your name? ", gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold())
	input.WithPlaceholder("John Doe")
	name, err := input.ReadSimple()
	if err != nil {
		terminal.Println(fmt.Sprintf("Error reading input: %v", err))
		terminal.Flush()
		return
	}

	// 2. Secure Input (Password) - Using new PasswordInput for enhanced security
	pwdInput := gooey.NewPasswordInput(terminal)
	pwdInput.WithPrompt("Enter a password:  ", gooey.NewStyle().WithForeground(gooey.ColorRed).WithBold())
	// Masked characters are shown by default for visual feedback

	securePassword, err := pwdInput.Read()
	if err != nil {
		terminal.Println(fmt.Sprintf("Error reading password: %v", err))
		terminal.Flush()
		return
	}
	defer securePassword.Clear() // Important: zero memory when done

	// Convert to string for demo (in production, use Bytes() method)
	password := securePassword.String()

	// 3. Input with Suggestions
	fruits := []string{"Apple", "Banana", "Cherry", "Date", "Elderberry", "Fig", "Grape", "Honeydew"}
	input.WithPrompt("Favorite fruit?    ", gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold())
	input.SetSuggestions(fruits)
	input.WithPlaceholder("Type to search...")

	// Read() provides full-featured input with real-time suggestions and tab completion
	fruit, err := input.Read()
	if err != nil {
		terminal.Println(fmt.Sprintf("Error reading fruit: %v", err))
		terminal.Flush()
		return
	}

	terminal.Println("\n✅ Form Complete!")
	terminal.Println(fmt.Sprintf("Name:     %s", name))
	terminal.Println(fmt.Sprintf("Password: %s (len=%d)", stringsRepeat("*", len(password)), len(password)))
	terminal.Println(fmt.Sprintf("Fruit:    %s", fruit))
	terminal.Flush()
}

func stringsRepeat(s string, count int) string {
	if count < 0 {
		return ""
	}
	res := ""
	for i := 0; i < count; i++ {
		res += s
	}
	return res
}
