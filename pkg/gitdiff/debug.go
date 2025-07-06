package gitdiff

import "fmt"

var DebugEnabled bool

// EnableDebug enables debug logging
func EnableDebug() {
	DebugEnabled = true
}

// DisableDebug disables debug logging
func DisableDebug() {
	DebugEnabled = false
}

// debugLog prints a message only if debug is enabled
func debugLog(format string, args ...any) {
	if DebugEnabled {
		fmt.Printf(format, args...)
	}
}
