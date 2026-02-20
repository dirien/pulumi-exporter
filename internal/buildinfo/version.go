// Package buildinfo holds build-time version information, injected via ldflags.
package buildinfo

import "fmt"

// Build-time variables, set via -ldflags.
var (
	Version   = "dev"
	Revision  = "unknown"
	Branch    = "unknown"
	BuildUser = "unknown"
	BuildDate = "unknown"
)

// Print returns a human-readable version string.
func Print(name string) string {
	return fmt.Sprintf("%s version %s (revision: %s, branch: %s, built by: %s on %s)",
		name, Version, Revision, Branch, BuildUser, BuildDate)
}
