package terminalstate

import (
	"errors"
	"io"
	"os"
)

// Write checks the complete protocol write and settles it before ownership changes.
func Write(out io.Writer, value string) error {
	_, err := WriteControl(out, value)
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

// WriteControl reports whether a complete control sequence was written, even
// when a writer returns an error with the full byte count. Callers track stack
// pushes/pops before flushing so a drain failure cannot repeat a completed pop.
func WriteControl(out io.Writer, value string) (complete bool, err error) {
	n, err := io.WriteString(out, value)
	if err == nil && n != len(value) {
		err = io.ErrShortWrite
	}
	return n == len(value), err
}
