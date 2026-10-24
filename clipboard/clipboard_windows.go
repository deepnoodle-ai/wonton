//go:build windows

package clipboard

import (
	"context"
)

// read implements clipboard reading for Windows using PowerShell Get-Clipboard.
// The -Raw flag returns the clipboard contents as a single string instead of an
// array of lines, preserving the original line endings of multiline content.
func read(ctx context.Context) (string, error) {
	out, err := runCommand(ctx, "powershell.exe", "-noprofile", "-command", "Get-Clipboard -Raw")
	if err != nil {
		return "", err
	}
	return trimPowerShellNewline(string(out)), nil
}

// write implements clipboard writing for Windows using clip.exe.
func write(ctx context.Context, text string) error {
	return runCommandWithStdin(ctx, text, "clip.exe")
}
