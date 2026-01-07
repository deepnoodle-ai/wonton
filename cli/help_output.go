package cli

import (
	"io"
	"strings"
)

func writeHelpOutput(w io.Writer, text string) error {
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	_, err := io.WriteString(w, text)
	return err
}

func writeHelpNewline(w io.Writer) error {
	_, err := io.WriteString(w, "\n")
	return err
}
