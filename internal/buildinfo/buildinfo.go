// Package buildinfo holds build-time information like the version.
// This is a separate package so that other packages can import it without
// worrying about introducing circular dependencies.
package buildinfo

// Updated by linker flags during build.
var (
	Version   string = "0.0.0"
	GitCommit string
	BuiltBy   string
)
