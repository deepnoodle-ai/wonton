//go:build windows

package clipboard

import (
	"context"
	"strings"
)

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

func write(ctx context.Context, text string) error {
	// Use clip.exe for writing which is simpler
	return runCommandWithStdin(ctx, text, "clip.exe")
}
