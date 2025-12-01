package slog

import "sync"

// buffer is a byte slice with convenience methods for building log output.
type buffer []byte

var bufPool = sync.Pool{
	New: func() any {
		b := make(buffer, 0, 1024)
		return (*buffer)(&b)
	},
}

func newBuffer() *buffer {
	return bufPool.Get().(*buffer)
}

// Free returns the buffer to the pool for reuse.
func (b *buffer) Free() {
	// To reduce peak allocation, return only smaller buffers to the pool.
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}

func (b *buffer) Write(bytes []byte) (int, error) {
	*b = append(*b, bytes...)
	return len(bytes), nil
}

func (b *buffer) WriteByte(char byte) error {
	*b = append(*b, char)
	return nil
}

func (b *buffer) WriteString(str string) (int, error) {
	*b = append(*b, str...)
	return len(str), nil
}
