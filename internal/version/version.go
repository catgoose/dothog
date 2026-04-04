// Package version exposes the application version, set at build time via ldflags.
package version

// Version is the current application version. Override at build time with:
//
//	go build -ldflags="-X catgoose/dothog/internal/version.Version=v1.2.3"
var Version = "dev"

// BuildDate is the date the binary was built. Override at build time with:
//
//	go build -ldflags="-X catgoose/dothog/internal/version.BuildDate=2024-01-15"
var BuildDate = ""

// Asset appends a cache-busting version query parameter to a static asset path.
func Asset(path string) string {
	return path + "?v=" + Version
}

// Display returns the version string with build date if available.
func Display() string {
	if BuildDate != "" {
		return Version + " (" + BuildDate + ")"
	}
	return Version
}
