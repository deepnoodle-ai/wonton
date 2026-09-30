package terminalstate

import (
	"errors"
	"io"
	"os"
)

// Write checks the complete protocol write and settles it before ownership changes.
func Write(out io.Writer, value string) error {
	n, err := io.WriteString(out, value)
	if err == nil && n != len(value) {
		err = io.ErrShortWrite
	}
	return errors.Join(err, Flush(out))
}
func Flush(out io.Writer) error {
	var err error
	if flush, ok := out.(interface{ Flush() error }); ok {
		err = flush.Flush()
	}
	if file, ok := out.(*os.File); ok {
		err = errors.Join(err, drain(file))
	}
	return err
}
