package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	// Initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		log.Fatal(err)
	}
	defer terminal.Close()

	terminal.Clear()
	terminal.Println("╔═══════════════════════════════════════════════╗")
	terminal.Println("║     Gooey Input API Demo                     ║")
	terminal.Println("║     Demonstrating the new simplified API     ║")
	terminal.Println("╚═══════════════════════════════════════════════╝")
	terminal.Println("")

	// Example 1: ReadSimple - Basic line input
	terminal.Println("1. ReadSimple() - Basic line input")
	terminal.Println("   Use for simple, non-interactive line reading")
	terminal.Println("")

	input1 := gooey.NewInput(terminal)
	input1.WithPrompt("Enter your name: ", gooey.NewStyle().WithForeground(gooey.ColorCyan))

	name, err := input1.ReadSimple()
	if err != nil {
		log.Fatal(err)
	}
	terminal.Println(fmt.Sprintf("   → You entered: %s", name))
	terminal.Println("")

	// Example 2: Read - Full-featured input with suggestions
	terminal.Println("2. Read() - Full-featured input")
	terminal.Println("   Try arrow keys, history, and Tab for autocomplete")
	terminal.Println("")

	input2 := gooey.NewInput(terminal)
	input2.WithPrompt("Choose a command: ", gooey.NewStyle().WithForeground(gooey.ColorGreen))
	input2.SetSuggestions([]string{"start", "stop", "restart", "status", "help"})

	command, err := input2.Read()
	if err != nil {
		log.Fatal(err)
	}
	terminal.Println(fmt.Sprintf("   → You chose: %s", command))
	terminal.Println("")

	// Example 3: ReadPassword - Secure password input
	terminal.Println("3. ReadPassword() - Secure password input")
	terminal.Println("   Your input will not be shown on screen")
	terminal.Println("")

	input3 := gooey.NewInput(terminal)
	input3.WithPrompt("Enter password: ", gooey.NewStyle().WithForeground(gooey.ColorYellow))

	password, err := input3.ReadPassword()
	if err != nil {
		log.Fatal(err)
	}
	terminal.Println(fmt.Sprintf("   → Password received (%d characters)", len(password)))
	terminal.Println("")

	// Summary
	terminal.Println("╔═══════════════════════════════════════════════╗")
	terminal.Println("║     Demo Complete!                           ║")
	terminal.Println("╚═══════════════════════════════════════════════╝")
	terminal.Println("")
	terminal.Println("Summary of the new Input API:")
	terminal.Println("  • Read()         - Full-featured interactive input")
	terminal.Println("  • ReadPassword() - Secure password entry")
	terminal.Println("  • ReadSimple()   - Basic line reading")
	terminal.Println("")
	terminal.Println("See documentation/input_api_migration.md for more details")
	terminal.Flush()
}
