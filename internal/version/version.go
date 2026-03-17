// Package version exposes the application version, set at build time via ldflags.
package version

// Version is the current application version. Override at build time with:
//
//	go build -ldflags="-X catgoose/dothog/internal/version.Version=v1.2.3"
var Version = "dev"
