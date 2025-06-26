//go:build !windows

package termio

import (
	"os"
	"syscall"
	"time"
)

func clearStdinBufferPlatform() {
	fd := int(os.Stdin.Fd())
	const TCIFLUSH = 0

	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(0x540B), uintptr(TCIFLUSH)) //nolint:errcheck
}

func drainRemainingInput() {
	fd := int(os.Stdin.Fd())

	err := syscall.SetNonblock(fd, true)
	if err != nil {
		return
	}

	defer func() {
		syscall.SetNonblock(fd, false) //nolint:errcheck
	}()

	buffer := make([]byte, 1024)

	for i := 0; i < 10; i++ { // Limit attempts
		n, err := syscall.Read(fd, buffer)
		if err != nil || n == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
}
