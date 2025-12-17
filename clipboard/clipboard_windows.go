//go:build windows

package clipboard

import (
	"context"
	"strings"
)

// read implements clipboard reading for Windows using PowerShell Get-Clipboard.
func read(ctx context.Context) (string, error) {
	out, err := runCommand(ctx, "powershell.exe", "-command", "Get-Clipboard")
	if err != nil {
		return "", err
	}
	// PowerShell adds CRLF at the end
	result := string(out)
	result = strings.TrimSuffix(result, "\r\n")
	result = strings.TrimSuffix(result, "\n")
	return result, nil
}

// write implements clipboard writing for Windows using clip.exe.
func write(ctx context.Context, text string) error {
	return runCommandWithStdin(ctx, text, "clip.exe")
}
