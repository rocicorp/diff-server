package version

// This is injected with the correct value when building a release.
var v = "0.0.0+devbuild"

// Version returns the current version of Replicant.
func Version() string {
	return v
}
