package versioninfo

import (
	"strings"

	"github.com/coreos/go-semver/semver"
)

// A Info contains a version.
type Info struct {
	Version string
	Commit  string
	BuiltBy string
}

func (vi Info) String() string {
	var versionElems []string
	if vi.Version != "" {
		version, err := semver.NewVersion(strings.TrimPrefix(vi.Version, "v"))
		if err != nil {
			return vi.Version
		}
		versionElems = append(versionElems, "v"+version.String())
	} else {
		versionElems = append(versionElems, "dev")
	}
	if vi.Commit != "" {
		versionElems = append(versionElems, "commit "+vi.Commit)
	}
	if vi.BuiltBy != "" {
		versionElems = append(versionElems, "built by "+vi.BuiltBy)
	}
	return strings.Join(versionElems, ", ")
}
