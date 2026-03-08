package wrapper

import (
	"bufio"
	"context"
	"io"
)

const chunkSize = 4096
const maxLineSize = 64 * 1024 // 64 KB

// IOBridge copies data from src to dst, applying process() to each unit.
// When lineBuffered is true, data is processed line-by-line (prevents
// chunk-boundary secret splitting). When false, chunk-based reads are used
// (better for interactive character-by-character input).
type IOBridge struct {
	src          io.Reader
	dst          io.Writer
	process      func(string) string
	lineBuffered bool
}

// NewIOBridge creates an IOBridge between src and dst.
// process is applied to each chunk/line before writing; use identity func for passthrough.
func NewIOBridge(src io.Reader, dst io.Writer, process func(string) string) *IOBridge {
	return &IOBridge{src: src, dst: dst, process: process}
}

// NewLineBufferedIOBridge creates an IOBridge that processes data line-by-line,
// preventing secrets from being missed when they straddle chunk boundaries.
func NewLineBufferedIOBridge(src io.Reader, dst io.Writer, process func(string) string) *IOBridge {
	return &IOBridge{src: src, dst: dst, process: process, lineBuffered: true}
}

// Run reads from src, applies process, and writes to dst until
// the context is cancelled or EOF/error is encountered.
func (b *IOBridge) Run(ctx context.Context) error {
	if b.lineBuffered {
		return b.runLineBuffered(ctx)
	}
	return b.runChunked(ctx)
}

// runChunked is the original chunk-based implementation, suitable for
// interactive stdin where data arrives character-by-character.
func (b *IOBridge) runChunked(ctx context.Context) error {
	buf := make([]byte, chunkSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := b.src.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			processed := b.process(chunk)
			if _, werr := io.WriteString(b.dst, processed); werr != nil {
				return werr
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// runLineBuffered processes data line-by-line using bufio.Scanner with ScanLines.
// This ensures secrets that straddle chunk boundaries are still detected.
func (b *IOBridge) runLineBuffered(ctx context.Context) error {
	scanner := bufio.NewScanner(b.src)
	scanner.Buffer(make([]byte, maxLineSize), maxLineSize)
	scanner.Split(bufio.ScanLines)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return nil // EOF
		}

		// scanner.Text() strips the newline; re-add it after processing.
		line := scanner.Text()
		processed := b.process(line)
		if _, werr := io.WriteString(b.dst, processed+"\n"); werr != nil {
			return werr
		}
	}
}
