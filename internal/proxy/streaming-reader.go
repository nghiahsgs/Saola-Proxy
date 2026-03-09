// Package proxy provides a streaming reader that rehydrates placeholders
// on-the-fly as chunks flow through, enabling SSE response rehydration.
package proxy

import "io"

// streamingReader wraps an io.ReadCloser and applies a transform function
// to each chunk as it passes through. Used to rehydrate SSE streaming responses.
type streamingReader struct {
	source    io.ReadCloser
	transform func(string) string
	buf       []byte // leftover transformed bytes not yet consumed by caller
}

// newStreamingReader creates a reader that applies transform to each chunk read.
func newStreamingReader(source io.ReadCloser, transform func(string) string) *streamingReader {
	return &streamingReader{
		source:    source,
		transform: transform,
	}
}

// Read implements io.Reader. Reads a chunk from source, transforms it,
// and copies the result into p.
func (sr *streamingReader) Read(p []byte) (int, error) {
	// Return buffered leftover bytes first.
	if len(sr.buf) > 0 {
		n := copy(p, sr.buf)
		sr.buf = sr.buf[n:]
		return n, nil
	}

	// Read a chunk from the source.
	tmp := make([]byte, len(p))
	n, err := sr.source.Read(tmp)
	if n > 0 {
		// Transform the chunk (rehydrate placeholders).
		transformed := sr.transform(string(tmp[:n]))
		sr.buf = []byte(transformed)

		// Copy as much as fits into caller's buffer.
		copied := copy(p, sr.buf)
		sr.buf = sr.buf[copied:]
		return copied, err
	}
	return 0, err
}

// Close closes the underlying source reader.
func (sr *streamingReader) Close() error {
	return sr.source.Close()
}
