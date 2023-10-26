package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	gitCommit  = "unknown"
	commitDate = "unknown"
	repoState  = "unknown"
	version    = "unknown"

	stateMap = map[string]string{
		"true":  "dirty",
		"false": "clean",
	}
)

// isUnset returns true when the provided string should be treated as an
// "unset" value. Builds that inject ldflags such as "-X var=" will set the
// variable to the empty string, which previously prevented the runtime build
// information gathered via debug.ReadBuildInfo from populating the field. For
// the purposes of version reporting we treat both the empty string and the
// literal "unknown" as unset.
func isUnset(s string) bool {
	// trim any surrounding whitespace to ensure accurate unset detection
	s = strings.TrimSpace(s)
	return s == "" || s == "unknown"
}

func String() string {
	return fmt.Sprintf("version: %q, commit: %q, date: %q, state: %q",
		valueOrUnknown(version),
		valueOrUnknown(gitCommit),
		valueOrUnknown(commitDate),
		valueOrUnknown(repoState))
}

func valueOrUnknown(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	return v
}

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if isUnset(gitCommit) {
				gitCommit = setting.Value
			}
		case "vcs.time":
			if isUnset(commitDate) {
				commitDate = setting.Value
			}
		case "vcs.modified":
			if v, ok := stateMap[setting.Value]; ok && isUnset(repoState) {
				repoState = v
			}
		}
	}
	if isUnset(version) && info.Main.Version != "" {
		version = info.Main.Version
	}
}
