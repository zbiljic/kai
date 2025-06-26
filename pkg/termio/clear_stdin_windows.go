//go:build windows

package termio

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procFlushConsoleInputBuffer = kernel32.NewProc("FlushConsoleInputBuffer")
	procPeekConsoleInput        = kernel32.NewProc("PeekConsoleInputW")
	procReadConsoleInput        = kernel32.NewProc("ReadConsoleInputW")
)

func clearStdinBufferPlatform() {
	handle := syscall.Handle(os.Stdin.Fd())

	ret, _, _ := procFlushConsoleInputBuffer.Call(uintptr(handle)) //nolint:errcheck

	// Don't handle errors - fails safely on non-console handles
	_ = ret
}

func drainRemainingInput() {
	handle := syscall.Handle(os.Stdin.Fd())

	for i := 0; i < 10; i++ {
		var eventsRead uint32
		buffer := make([]byte, 32) // INPUT_RECORD is about 20 bytes

		ret, _, _ := procPeekConsoleInput.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&buffer[0])),
			1,
			uintptr(unsafe.Pointer(&eventsRead)),
		) //nolint:errcheck

		if ret == 0 || eventsRead == 0 {
			break
		}

		procReadConsoleInput.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&buffer[0])),
			1,
			uintptr(unsafe.Pointer(&eventsRead)),
		)

		time.Sleep(5 * time.Millisecond)
	}
}
