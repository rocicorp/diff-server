package version

// This is updated when we bump the version, by the 'bump' command from repc.
const v = "1.1.0"

// This is injected with the correct value when building a release.
var h = "devbuild"

// Version returns the current version of Replicant.
func Version() string {
	return v + "+" + h
}
