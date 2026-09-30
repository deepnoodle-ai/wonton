package tui_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/deepnoodle-ai/wonton/tui"
)

func ExampleRuntime_Handoff() {
	runner := tui.NewRuntime(tui.NewTestTerminal(80, 24, io.Discard), nil, 30)
	operationErr, restoreErr := runner.Handoff(func() error {
		child := exec.Command("vi", "draft.txt")
		child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stdout
		return child.Run() // Wait for the editor before returning ownership.
	})
	if restoreErr != nil {
		fmt.Println("Terminal restoration failed:", restoreErr)
	}
	if errors.Is(operationErr, tui.ErrHandoffNotRunning) {
		fmt.Println("Call Handoff from HandleEvent while Run is active.")
	}
	// Output: Call Handoff from HandleEvent while Run is active.
}

func ExampleInlineApp_Handoff() {
	runner := tui.NewInlineApp()
	operationErr, restoreErr := runner.Handoff(func() error {
		child := exec.Command("vi", "draft.txt")
		child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stdout
		return child.Run()
	})
	if restoreErr != nil {
		fmt.Println("Terminal restoration failed:", restoreErr)
	}
	if errors.Is(operationErr, tui.ErrHandoffNotRunning) {
		fmt.Println("Call Handoff from HandleEvent while Run is active.")
	}
	// Output: Call Handoff from HandleEvent while Run is active.
}
