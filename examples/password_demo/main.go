package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer terminal.Reset()

	// Clear screen
	terminal.Clear()
	terminal.Flush()

	// Print header
	headerStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	terminal.SetStyle(headerStyle)
	terminal.Println("🔒 Gooey Secure Password Input Demo")
	terminal.Println("====================================")
	terminal.Reset()
	terminal.Println("")

	// Detect terminal type
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram != "" {
		infoStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
		terminal.SetStyle(infoStyle)
		terminal.Print(fmt.Sprintf("Detected terminal: %s\n", termProgram))
		if termProgram == "iTerm.app" {
			terminal.Println("✓ iTerm2 secure input mode will be enabled")
		} else if termProgram == "vscode" {
			terminal.Println("✓ VS Code terminal detected")
		} else {
			terminal.Println("ℹ Using generic secure mode")
		}
		terminal.Reset()
		terminal.Println("")
	}

	// Demo 1: Standard secure password (no echo)
	terminal.Println("")
	terminal.Println("Demo 1: Standard secure password (no echo)")
	terminal.Println("-------------------------------------------")

	pwdInput1 := gooey.NewPasswordInput(terminal)
	pwdInput1.WithPrompt("Enter password: ", gooey.NewStyle().WithForeground(gooey.ColorYellow))
	pwdInput1.ShowCharacters(false) // Disable visual feedback for true no-echo

	password1, err := pwdInput1.Read()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer password1.Clear() // Important: zero memory when done

	terminal.Println("")
	successStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	terminal.SetStyle(successStyle)
	terminal.Print(fmt.Sprintf("✓ Password received (length: %d)\n", password1.Len()))
	terminal.Reset()
	terminal.Println("")

	// Demo 2: Password with masked characters
	terminal.Println("Demo 2: Password with masked characters (*)")
	terminal.Println("--------------------------------------------")

	pwdInput2 := gooey.NewPasswordInput(terminal)
	pwdInput2.WithPrompt("Enter password: ", gooey.NewStyle().WithForeground(gooey.ColorYellow))
	pwdInput2.ShowCharacters(true) // Show asterisks

	password2, err := pwdInput2.Read()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer password2.Clear()

	terminal.Println("")
	terminal.SetStyle(successStyle)
	terminal.Print(fmt.Sprintf("✓ Password received (length: %d)\n", password2.Len()))
	terminal.Reset()
	terminal.Println("")

	// Demo 3: Password with custom mask character
	terminal.Println("Demo 3: Password with custom mask (•)")
	terminal.Println("---------------------------------------")

	pwdInput3 := gooey.NewPasswordInput(terminal)
	pwdInput3.WithPrompt("Enter password: ", gooey.NewStyle().WithForeground(gooey.ColorYellow))
	pwdInput3.ShowCharacters(true)
	pwdInput3.WithMaskChar('•') // Use bullet point

	password3, err := pwdInput3.Read()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer password3.Clear()

	terminal.Println("")
	terminal.SetStyle(successStyle)
	terminal.Print(fmt.Sprintf("✓ Password received (length: %d)\n", password3.Len()))
	terminal.Reset()
	terminal.Println("")

	// Demo 4: Password with max length
	terminal.Println("Demo 4: Password with max length (16 chars)")
	terminal.Println("--------------------------------------------")

	pwdInput4 := gooey.NewPasswordInput(terminal)
	pwdInput4.WithPrompt("Enter password: ", gooey.NewStyle().WithForeground(gooey.ColorYellow))
	pwdInput4.ShowCharacters(true)
	pwdInput4.WithMaxLength(16) // Limit to 16 characters

	password4, err := pwdInput4.Read()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer password4.Clear()

	terminal.Println("")
	terminal.SetStyle(successStyle)
	terminal.Print(fmt.Sprintf("✓ Password received (length: %d)\n", password4.Len()))
	terminal.Reset()
	terminal.Println("")

	// Demo 5: Password with placeholder
	terminal.Println("Demo 5: Password with placeholder")
	terminal.Println("----------------------------------")

	pwdInput5 := gooey.NewPasswordInput(terminal)
	pwdInput5.WithPrompt("Enter password: ", gooey.NewStyle().WithForeground(gooey.ColorYellow))
	pwdInput5.WithPlaceholder("(type to begin)")

	password5, err := pwdInput5.Read()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer password5.Clear()

	terminal.Println("")
	terminal.SetStyle(successStyle)
	terminal.Print(fmt.Sprintf("✓ Password received (length: %d)\n", password5.Len()))
	terminal.Reset()
	terminal.Println("")

	// Summary
	terminal.Println("=====================================")
	terminal.SetStyle(successStyle)
	terminal.Println("✓ All demos completed successfully!")
	terminal.Reset()
	terminal.Println("")

	// Security tips
	infoStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)
	terminal.SetStyle(infoStyle)
	terminal.Println("Security Features:")
	terminal.Reset()
	terminal.Println("  • iTerm2 secure input mode (prevents keylogging)")
	terminal.Println("  • Memory is zeroed after use (via defer password.Clear())")
	terminal.Println("  • Clipboard disable option available")
	terminal.Println("  • Paste confirmation (can be enabled)")
	terminal.Println("  • Max length enforcement")
	terminal.Println("")

	terminal.Println("Press Enter to exit...")
	fmt.Scanln()
}
