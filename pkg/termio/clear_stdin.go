package termio

import (
	"os"
	"time"

	"golang.org/x/term"
)

// ClearStdinBuffer clears any pending input from stdin to prevent
// unwanted input consumption during interactive prompts.
func ClearStdinBuffer() {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	clearStdinBufferPlatform()

	// Small delay to let any pending operations complete
	time.Sleep(10 * time.Millisecond)

	drainRemainingInput()
}
