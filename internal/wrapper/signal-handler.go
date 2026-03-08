package wrapper

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
)

// HandleSignals forwards relevant OS signals to the child process and
// keeps the PTY window size in sync. It blocks until ctx is done.
func HandleSignals(ctx context.Context, cancel context.CancelFunc, ptmx *os.File, cmd *exec.Cmd) {
	sigCh := make(chan os.Signal, 8)
	signal.Notify(sigCh, syscall.SIGWINCH, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Initial terminal size sync.
	_ = pty.InheritSize(os.Stdin, ptmx)

	for {
		select {
		case <-ctx.Done():
			return
		case sig, ok := <-sigCh:
			if !ok {
				return
			}
			switch sig {
			case syscall.SIGWINCH:
				_ = pty.InheritSize(os.Stdin, ptmx)
			case syscall.SIGINT, syscall.SIGTERM:
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			}
		}
	}
}
